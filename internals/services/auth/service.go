package auth

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
	"time"

	pubsub "github.com/assaidy/pubsubs"
	"github.com/assaidy/video_streaming_app/internals/services"
	"github.com/assaidy/video_streaming_app/internals/services/auth/queries"
	"github.com/assaidy/video_streaming_app/internals/utils"
	"github.com/assaidy/video_streaming_app/internals/utils/mailer"
	"github.com/assaidy/workers"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const Name = "auth"

var _ services.Service = (*Service)(nil)

type Service struct {
	db            *sql.DB
	redis         *redis.Client
	mailer        *mailer.Mailer
	pubsub        pubsub.Pubsub
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, redis *redis.Client, mailer *mailer.Mailer, pubsub pubsub.Pubsub, logger *slog.Logger) *Service {
	logger = logger.WithGroup("auth service")
	service := &Service{
		db:            db,
		redis:         redis,
		mailer:        mailer,
		pubsub:        pubsub,
		logger:        logger,
		workerManager: workers.NewWorkerManager(workers.WithLogger(logger)),
	}

	service.workerManager.RegisterWorker(
		workers.NewWorker("verification email sender", service.verificationEmailSenderJob,
			workers.WithNRuns(1),
			workers.WithNRetries(0),
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
				if !errors.Is(err, redis.Nil) && ctx.Err() == nil { // not a BLPop timout or context canceled
					me.logger.Error("failed to LPop email verification queue", "error", err)
				}
				continue
			}

			var payload EmailVerificationQueuePayload
			if err := json.Unmarshal([]byte(result[1]), &payload); err != nil {
				me.logger.Error("failed to decode email verification queue payload", "error", err)
				continue
			}

			if err := me.mailer.SendEmail(ctx, mailer.Message{
				From:        utils.MustGetEnv("EMAIL_FROM"),
				To:          []string{payload.Email},
				Subject:     "Verification email for Video Streaming App",
				ContentType: "text/html; charset=utf-8",
				Body:        fmt.Sprintf(`To verifiy you email click <a href="%s">here</a>`, payload.VerificationLink),
			}); err != nil {
				me.logger.Error("failed to send verification email", "error", err)
			}
		}
	}
}

func validateRegisterParams(email, password string) error {
	data := struct {
		Email    string
		Password string
	}{
		Email:    email,
		Password: password,
	}

	if err := validation.ValidateStruct(&data,
		// validate length because is.Email doesn't check the length
		validation.Field(&data.Email, validation.Required, is.Email, validation.Length(0, 255)),
		validation.Field(&data.Password, validation.Required, validation.Length(8, 50)),
	); err != nil {
		errs := err.(validation.Errors)
		return RegisterValidationErrors{
			Email:    errs["Email"],
			Password: errs["Password"],
		}
	}

	return nil
}

type RegisterValidationErrors struct {
	Email    error
	Password error
}

func (me RegisterValidationErrors) Error() string {
	return fmt.Sprintf("email: %v, password: %v", me.Email, me.Password)
}

func (me *Service) Register(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if err := validateRegisterParams(email, password); err != nil {
		return "", fmt.Errorf("%w: %w", ErrValidation, err)
	}

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := queries.New(me.db).WithTx(tx)

	if ok, err := qtx.CheckEmail(ctx, email); err != nil {
		return "", fmt.Errorf("failed to check email: %w", err)
	} else if ok {
		return "", ErrEmailConflict
	}

	userID := ulid.Make().String()
	password_hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	if err := qtx.InsertUser(ctx, queries.InsertUserParams{
		ID:           userID,
		Email:        email,
		PasswordHash: string(password_hash),
	}); err != nil {
		return "", fmt.Errorf("failed to insert user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit tx: %w", err)
	}

	return userID, nil
}

const EmailVerificationQueue = "auth.EmailVerification"

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
	qtx := queries.New(me.db).WithTx(tx)

	user, err := qtx.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get user by email: %w", err)
	}

	verificationTokenID := ulid.Make().String()
	verificationToken := fmt.Sprintf("%s_%s", verificationTokenID, generateCryptoRandomHex(32))

	if err := qtx.InsertEmailVerificationToken(ctx, queries.InsertEmailVerificationTokenParams{
		ID:        verificationTokenID,
		OwnerID:   user.ID,
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
		VerificationLink: fmt.Sprintf("%s?token=%s", url, verificationTokenID),
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
	q := queries.New(me.db)

	if n, err := q.VerifyEmailByToken(ctx, string(verificationToken)); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	} else if n == 0 {
		return ErrNotFound
	}

	return nil
}

type Session struct {
	ID           string
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
	qtx := queries.New(me.db).WithTx(tx)

	user, err := qtx.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if !user.IsVerified || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnverified
	}

	sessionID := ulid.Make().String()
	// session id prefix ensures uniqueness
	sessionToken := fmt.Sprintf("%s_%s", sessionID, generateCryptoRandomHex(32))
	csrfToken := fmt.Sprintf("%s_%s", sessionID, generateCryptoRandomHex(32))
	// Set cookie max-age to 400 days: https://developer.chrome.com/blog/cookie-max-age-expires
	sessionExpirationDate := time.Now().Add(400 * 24 * time.Hour)

	if err := qtx.InsertSession(ctx, queries.InsertSessionParams{
		ID:           sessionID,
		OwnerID:      user.ID,
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
		ID:           sessionID,
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

func (me *Service) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	q := queries.New(me.db)

	session, err := q.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	return &Session{
		ID:           session.ID,
		SessionToken: session.SessionToken,
		CsrfToken:    session.CsrfToken,
		ExpiresAt:    session.ExpiresAt,
	}, nil
}

func (me *Service) Logout(ctx context.Context, userID string, sessionID string) error {
	q := queries.New(me.db)

	if nDeleted, err := q.DeleteSessionForUser(ctx, queries.DeleteSessionForUserParams{
		SessionID: sessionID,
		UserID:    userID,
	}); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	} else if nDeleted == 0 {
		return ErrNotFound
	}

	return nil
}

func (me *Service) DeleteAccount(ctx context.Context, userID string) error {
	q := queries.New(me.db)

	if nDeleted, err := q.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	} else if nDeleted == 0 {
		return ErrNotFound
	}

	return nil
}
