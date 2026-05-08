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
	"regexp"
	"strings"
	"time"

	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/user/queries"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/vodzilla/internals/utils/mailer"
	"github.com/assaidy/workers"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const Name = "user"

var _ services.Service = (*Service)(nil)

type Service struct {
	db            *sql.DB
	redis         *redis.Client
	mailer        *mailer.Mailer
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, redis *redis.Client, mailer *mailer.Mailer, logger *slog.Logger) *Service {
	logger = logger.WithGroup("user service")
	service := &Service{
		db:            db,
		redis:         redis,
		mailer:        mailer,
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

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]*$`)

func validateRegisterParams(email, password, name, username string) error {
	data := struct {
		Email    string
		Password string
		Name     string
		Username string
	}{
		Email:    email,
		Password: password,
		Name:     name,
		Username: username,
	}

	if err := validation.ValidateStruct(&data,
		validation.Field(&data.Email, validation.Required, is.Email),
		validation.Field(&data.Password, validation.Required, validation.Length(8, 50)),
		validation.Field(&data.Name, validation.Required, validation.Length(1, 256)),
		validation.Field(&data.Username, validation.Required, validation.Length(1, 32), validation.Match(usernameRegex).Error("can only cotain letters, digits or _")),
	); err != nil {
		errs := err.(validation.Errors)
		return RegisterValidationErrors{
			Email:    errs["Email"],
			Password: errs["Password"],
			Name:     errs["Name"],
			Username: errs["Username"],
		}
	}

	return nil
}

type RegisterValidationErrors struct {
	Email    error
	Password error
	Name     error
	Username error
}

func (me RegisterValidationErrors) Error() string {
	return fmt.Sprintf("email: %v, password: %v", me.Email, me.Password)
}

func (me *Service) Register(ctx context.Context, email, password, name, username string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	username = strings.TrimSpace(username)

	if err := validateRegisterParams(email, password, name, username); err != nil {
		return fmt.Errorf("%w: %w", ErrValidation, err)
	}

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := queries.New(me.db).WithTx(tx)

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

	userId := ulid.Make().String()
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

const EmailVerificationQueue = "UserService:EmailVerification"

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

	verificationTokenId := ulid.Make().String()
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
	q := queries.New(me.db)

	if n, err := q.VerifyEmailByToken(ctx, string(verificationToken)); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	} else if n == 0 {
		return ErrNotFound
	}

	return nil
}

type Session struct {
	Id           string
	OwnerId      string
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
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}

	if !user.IsVerified {
		return nil, ErrUnverified
	}

	sessionId := ulid.Make().String()
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

func (me *Service) GetSession(ctx context.Context, sessionId string) (*Session, error) {
	q := queries.New(me.db)

	session, err := q.GetSessionById(ctx, sessionId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
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

func (me *Service) Logout(ctx context.Context, userId string, sessionId string) error {
	q := queries.New(me.db)

	if nDeleted, err := q.DeleteSessionForUser(ctx, queries.DeleteSessionForUserParams{
		SessionId: sessionId,
		UserId:    userId,
	}); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	} else if nDeleted == 0 {
		return ErrNotFound
	}

	return nil
}

func (me *Service) DeleteAccount(ctx context.Context, userId string) error {
	q := queries.New(me.db)

	if nDeleted, err := q.DeleteUser(ctx, userId); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	} else if nDeleted == 0 {
		return ErrNotFound
	}

	return nil
}

type User struct {
	Id       string
	Name     string
	Username string
	Email    string
	Bio      string
}

func (me *Service) GetUserById(ctx context.Context, userId string) (*User, error) {
	q := queries.New(me.db)

	user, err := q.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
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
	q := queries.New(me.db)

	user, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
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

func (me *Service) EditProfile(ctx context.Context, userId, name, username, bio string) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := queries.New(me.db).WithTx(tx)

	user, err := qtx.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
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
