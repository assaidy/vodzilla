package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	pubsub "github.com/assaidy/pubsubs"
	"github.com/assaidy/video_streaming_app/internals/services"
	"github.com/assaidy/video_streaming_app/internals/services/auth/queries"
	"github.com/assaidy/video_streaming_app/internals/utils"
	"github.com/assaidy/video_streaming_app/internals/utils/mailer"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const Name = "auth"

var _ services.Service = (*Service)(nil)

type Service struct {
	db        *sql.DB
	mailer    *mailer.Mailer
	pubsub    pubsub.Pubsub
	logger    *slog.Logger
	stopChan  chan struct{} // used to communicate stop signal with goroutines
	workersWg sync.WaitGroup
}

func New(db *sql.DB, mailer *mailer.Mailer, pubsub pubsub.Pubsub, logger *slog.Logger) *Service {
	return &Service{db: db, mailer: mailer, pubsub: pubsub, logger: logger}
}

func (me *Service) Start(ctx context.Context) error {
	// FIX: this ctx is used mainly for start timeout.
	// it shouldn't be passed to workers as it will stop them event if we started all of them before the timeout.
	// use worker manager from workers after adding a stop mechanism to it: WorkerManager.Stop()
	me.workersWg.Go(func() { me.startVerificationEmailSenderWorker(ctx) })
	return nil
}

func (me *Service) Stop(ctx context.Context) error {
	stoppedChan := make(chan struct{})
	go func() {
		close(me.stopChan)
		me.workersWg.Wait()
		close(stoppedChan)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stoppedChan:
		return nil
	}
}

func (me *Service) startVerificationEmailSenderWorker(ctx context.Context) {
	sub := pubsub.SubscribeWithCodec(
		ctx,
		me.pubsub,
		EmailVerificationEvent,
		func(ctx context.Context, payload EmailVerificationEventPayload) error {
			if err := me.mailer.SendEmail(ctx, mailer.Message{
				From:        utils.MustGetEnv("EMAIL_FROM"),
				To:          []string{payload.Email},
				Subject:     "Verification email for Video Streaming App",
				ContentType: "text/plain; charset=utf-8",
				Body:        fmt.Sprintf(`To verifiy you email click <a href="%s">here</a>`, payload.VerificationLink),
			}); err != nil {
				return fmt.Errorf("failed to verification send email: %w", err)
			}
			return nil
		},
		pubsub.CodecJson,
	)
	defer sub.Close()

	for {
		select {
		case err := <-sub.Errs():
			me.logger.Error("failed to handle event", "service", Name, "event", EmailVerificationEvent, "error", err)
		case <-me.stopChan:
			return
		}
	}
}

func validateRegisterParams(email, password string) error {
	var errs SignupValidationErrors
	if err := validation.Validate(email, validation.Required, is.Email, validation.Length(0, 255)); err != nil {
		// validate length because is.Email doesn't check the length
		errs.Email = err
	}
	if err := validation.Validate(password, validation.Required, validation.Length(8, 50)); err != nil {
		errs.Password = err
	}
	return errs
}

type SignupValidationErrors struct {
	Email    error
	Password error
}

func (me SignupValidationErrors) Error() string {
	return fmt.Sprintf("email: %v, password: %v", me.Email, me.Password)
}

func (me *Service) Signup(ctx context.Context, email, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))

	if err := validateRegisterParams(email, password); err != nil {
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

	userID := uuid.Must(uuid.NewV7())
	password_hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := qtx.InsertUser(ctx, queries.InsertUserParams{
		ID:           userID,
		Email:        email,
		PasswordHash: string(password_hash),
	}); err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *Service) SendEmailVerificationEmail(ctx context.Context, userID uuid.UUID, email, url string) error {
	verificationTokenID := uuid.Must(uuid.NewV7())
	verificationToken := fmt.Sprintf("%s_%s", verificationTokenID, generateCryptoRandomHex(32))

	q := queries.New(me.db)
	if err := q.InsertEmailVerificationToken(ctx, queries.InsertEmailVerificationTokenParams{
		ID:        verificationTokenID,
		OwnerID:   userID,
		Token:     verificationToken,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}); err != nil {
		return fmt.Errorf("failed to insert email verification token: %w", err)
	}

	if err := pubsub.PublishWithCodec(
		ctx,
		me.pubsub,
		EmailVerificationEvent,
		EmailVerificationEventPayload{
			Email:            email,
			VerificationLink: fmt.Sprintf("%s?token=%s", url, verificationToken),
		},
		pubsub.CodecJson,
	); err != nil {
		return fmt.Errorf("failed to publish %s event: %w", EmailVerificationEvent, err)
	}

	return nil
}

func (me *Service) VerifyEmail(ctx context.Context, verificationToken string) error {
	q := queries.New(me.db)
	if n, err := q.VerifyEmailByToken(ctx, verificationToken); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

type Session struct {
	ID           uuid.UUID
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

	if !user.IsVerified {
		return nil, ErrUnverified
	}
	if user.Email != email || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}

	sessionID := uuid.Must(uuid.NewV7())
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

func (me *Service) GetSessionByToken(ctx context.Context, sessionToken string) (*Session, error) {
	q := queries.New(me.db)
	session, err := q.GetSessionByToken(ctx, sessionToken)
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

func (me *Service) Logout(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
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

func (me *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	q := queries.New(me.db)
	if nDeleted, err := q.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	} else if nDeleted == 0 {
		return ErrNotFound
	}
	return nil
}
