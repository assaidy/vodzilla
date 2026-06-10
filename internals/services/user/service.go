package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/user/queries"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/vodzilla/internals/utils/mailer"
	"github.com/assaidy/workers"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const Name = "user"

var _ services.Service = (*Service)(nil)

type Service struct {
	db            *sql.DB
	queries       *queries.Queries
	redis         *redis.Client
	s3            *s3.Client
	mailer        *mailer.Mailer
	logger        *slog.Logger
	workerManager *workers.WorkerManager
	userMutexes   sync.Map
}

func New(db *sql.DB, redis *redis.Client, s3 *s3.Client, mailer *mailer.Mailer, logger *slog.Logger) *Service {
	service := &Service{
		db:            db,
		queries:       queries.New(db),
		redis:         redis,
		s3:            s3,
		mailer:        mailer,
		logger:        logger,
		workerManager: workers.NewWorkerManager(workers.WithLogger(logger)),
	}

	service.workerManager.RegisterWorker(
		workers.NewWorker(
			"verification email sender",
			service.verificationEmailSenderJob,
			workers.WithRetryDelay(time.Second),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
		),
	)
	service.workerManager.RegisterWorker(
		workers.NewWorker(
			"expired email verification tokens cleanup",
			service.emailVerificationTokensCleanupJob,
			workers.WithTick(1*time.Hour),
			workers.WithTimeout(5*time.Minute),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
		),
	)
	service.workerManager.RegisterWorker(
		workers.NewWorker(
			"expired sessions cleanup",
			service.sessionsCleanupJob,
			workers.WithSchedules(workers.WeeklyAt(time.Friday, 2, 0)),
			workers.WithTimeout(10*time.Minute),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
		),
	)

	return service
}

func (me *Service) Start(ctx context.Context) error {
	me.workerManager.Start()
	return nil
}

func (me *Service) Stop(ctx context.Context) error {
	me.workerManager.Stop()
	return nil
}

func (me *Service) verificationEmailSenderJob(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			result, err := me.redis.BLPop(ctx, 5*time.Second, EmailVerificationQueue).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) || ctx.Err() != nil {
					// BLPop timout or context canceled
					continue
				}
				return fmt.Errorf("failed to BLPop email verification queue: %w", err)
			}

			var payload EmailVerificationQueuePayload
			if err := json.Unmarshal([]byte(result[1]), &payload); err != nil {
				return fmt.Errorf("failed to decode email verification queue payload: %w", err)
			}

			if err := me.mailer.SendEmail(ctx, mailer.Message{
				From:        utils.MustGetEnv("EMAIL_FROM"),
				To:          []string{payload.Email},
				Subject:     "Verification email for Video Streaming App",
				ContentType: "text/html; charset=utf-8",
				Body:        fmt.Sprintf(`To verifiy you email click <a href="%s">here</a>`, payload.VerificationLink),
			}); err != nil {
				return fmt.Errorf("failed to send verification email: %w", err)
			}
		}
	}
}

func (me *Service) Register(ctx context.Context, email, password, name, username string) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckEmail(ctx, email); err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	} else if ok {
		return ErrEmailConflict
	}

	if ok, err := qtx.CheckUsername(ctx, username); err != nil {
		return fmt.Errorf("failed to check username: %w", err)
	} else if ok {
		return ErrUsernameConflict
	}

	userId := uuid.Must(uuid.NewV7())
	password_hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := qtx.InsertUser(ctx, queries.InsertUserParams{
		Id:           userId,
		Email:        email,
		PasswordHash: string(password_hash),
		Name:         name,
		Username:     username,
	}); err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

const EmailVerificationQueue = "user_service:email_verification"

type EmailVerificationQueuePayload struct {
	Email            string
	VerificationLink string
}

func (me *Service) SendVerificationEmail(ctx context.Context, email, url string) error {
	email = strings.ToLower(strings.TrimSpace(email))

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	user, err := qtx.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user by email: %w", err)
	}

	verificationTokenId := uuid.Must(uuid.NewV7())
	verificationToken := fmt.Sprintf("%s_%s", verificationTokenId, generateCryptoRandomHex(32))

	if err := qtx.InsertEmailVerificationToken(ctx, queries.InsertEmailVerificationTokenParams{
		Id:        verificationTokenId,
		OwnerId:   user.Id,
		Token:     verificationToken,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}); err != nil {
		return fmt.Errorf("failed to insert email verification token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	paylaod, err := json.Marshal(EmailVerificationQueuePayload{
		Email:            email,
		VerificationLink: fmt.Sprintf("%s?token=%s", url, verificationToken),
	})
	if err != nil {
		return fmt.Errorf("failed to encode email verification queue payload: %w", err)
	}
	if err := me.redis.RPush(ctx, EmailVerificationQueue, paylaod).Err(); err != nil {
		return fmt.Errorf("failed to enqueue email verification queue payload: %w", err)
	}

	return nil
}

func (me *Service) VerifyEmail(ctx context.Context, verificationToken string) error {
	if n, err := me.queries.VerifyEmailByToken(ctx, string(verificationToken)); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	} else if n == 0 {
		return ErrTokenNotFound
	}

	return nil
}

func (me *Service) emailVerificationTokensCleanupJob(ctx context.Context) error {
	if err := me.queries.BatchDeleteExpiredEmailVerificationTokens(ctx); err != nil {
		return fmt.Errorf("failed to batch delete expired or deleted email verification tokens: %w", err)
	}

	return nil
}

type Session struct {
	Id           uuid.UUID
	OwnerId      uuid.UUID
	SessionToken string
	CsrfToken    string
	ExpiresAt    time.Time
}

func (me *Service) Login(ctx context.Context, email, password string) (*Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	user, err := qtx.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}

	if !user.IsVerified {
		return nil, ErrUnverified
	}

	sessionId := uuid.Must(uuid.NewV7())
	// session id prefix ensures uniqueness
	sessionToken := fmt.Sprintf("%s_%s", sessionId, generateCryptoRandomHex(32))
	csrfToken := fmt.Sprintf("%s_%s", sessionId, generateCryptoRandomHex(32))
	// Set cookie max-age to 400 days: https://developer.chrome.com/blog/cookie-max-age-expires
	sessionExpirationDate := time.Now().Add(400 * 24 * time.Hour)

	if err := qtx.InsertSession(ctx, queries.InsertSessionParams{
		Id:           sessionId,
		OwnerId:      user.Id,
		SessionToken: sessionToken,
		CsrfToken:    csrfToken,
		ExpiresAt:    sessionExpirationDate,
	}); err != nil {
		return nil, fmt.Errorf("failed to insert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return &Session{
		Id:           sessionId,
		SessionToken: sessionToken,
		CsrfToken:    csrfToken,
		ExpiresAt:    sessionExpirationDate,
	}, nil
}

func generateCryptoRandomHex(nBytes uint) string {
	buf := make([]byte, nBytes)
	rand.Reader.Read(buf)
	return hex.EncodeToString(buf)
}

func (me *Service) GetSession(ctx context.Context, sessionId uuid.UUID) (*Session, error) {
	session, err := me.queries.GetSessionById(ctx, sessionId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	return &Session{
		Id:           session.Id,
		OwnerId:      session.OwnerId,
		SessionToken: session.SessionToken,
		CsrfToken:    session.CsrfToken,
		ExpiresAt:    session.ExpiresAt,
	}, nil
}

func (me *Service) Logout(ctx context.Context, userId uuid.UUID, sessionId uuid.UUID) error {
	if nDeleted, err := me.queries.DeleteSessionForUser(ctx, queries.DeleteSessionForUserParams{
		SessionId: sessionId,
		UserId:    userId,
	}); err != nil {
		return fmt.Errorf("failed to soft delete session: %w", err)
	} else if nDeleted == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (me *Service) sessionsCleanupJob(ctx context.Context) error {
	if err := me.queries.BatchDeleteExpiredSessions(ctx); err != nil {
		return fmt.Errorf("failed to batch delete expired sessions: %w", err)
	}

	return nil
}

type User struct {
	Id       uuid.UUID
	Name     string
	Username string
	Email    string
	Bio      string
}

func (me *Service) GetUserById(ctx context.Context, userId uuid.UUID) (*User, error) {
	user, err := me.queries.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &User{
		Id:       user.Id,
		Name:     user.Name,
		Username: user.Username,
		Email:    user.Email,
		Bio:      user.Bio.String,
	}, nil
}

func (me *Service) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user, err := me.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &User{
		Id:       user.Id,
		Name:     user.Name,
		Username: user.Username,
		Email:    user.Email,
		Bio:      user.Bio.String,
	}, nil
}

func (me *Service) DoesUserExist(ctx context.Context, userId uuid.UUID) (bool, error) {
	ok, err := me.queries.CheckUserId(ctx, userId)
	if err != nil {
		return false, fmt.Errorf("failed to check user id: %w", err)
	}

	return ok, nil
}

func (me *Service) EditProfile(ctx context.Context, userId uuid.UUID, name, username, bio string) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	user, err := qtx.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to get user by id: %w", err)
	}

	if user.Username != username {
		if ok, err := qtx.CheckUsername(ctx, username); err != nil {
			return fmt.Errorf("failed to check username: %w", err)
		} else if ok {
			return ErrUsernameConflict
		}
	}

	if err := qtx.UpdateProfile(ctx, queries.UpdateProfileParams{
		UserId:   userId,
		Name:     name,
		Username: username,
		Bio:      sql.NullString{Valid: true, String: bio},
	}); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

// FIX: move lcoks to handler
// - also add locks to vidoes
func (me *Service) AquireUserLock(userId uuid.UUID) {
	mu, _ := me.userMutexes.LoadOrStore(userId, new(sync.Mutex))
	mu.(*sync.Mutex).Lock()
}

// TODO: I need to delete the lock from the map if it's no longer aquired.
func (me *Service) ReleaseUserLock(userId uuid.UUID) {
	if mu, ok := me.userMutexes.Load(userId); ok {
		mu.(*sync.Mutex).Unlock()
	}
}

func (me *Service) DeleteUser(ctx context.Context, userId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if n, err := qtx.SoftDeleteUserById(ctx, userId); err != nil {
		return fmt.Errorf("failed to soft delete user by id: %w", err)
	} else if n == 0 {
		return ErrUserNotFound
	}

	if err := qtx.DeleteAllSessionsForUser(ctx, userId); err != nil {
		return fmt.Errorf("failed to soft delete all sessions for user: %w", err)
	}

	if err := qtx.DeleteAllEmailVerificationTokensForUser(ctx, userId); err != nil {
		return fmt.Errorf("failed to soft delete all email verification tokens for user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	payload, err := json.Marshal(events.UserDeletedEventPayload{UserId: userId})
	if err != nil {
		return fmt.Errorf("failed to marshal %q event payload: %w", events.UserDeletedEvent, err)
	}
	if err := me.redis.Publish(ctx, events.UserDeletedEvent, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish %q event: %w", events.UserDeletedEvent, err)
	}

	return nil
}
