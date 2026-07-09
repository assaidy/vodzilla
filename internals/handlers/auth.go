package handlers

import (
	"errors"
	"regexp"
	"strings"
	"time"

	user_service "github.com/assaidy/vodzilla/internals/services/user"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var usernameLettersRule = validation.Match(regexp.MustCompile(`^[A-Za-z0-9_]*$`)).Error("can only contain letters, digits or _")

func (me *Handler) HandleRegister(c fiber.Ctx) error {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.Name = strings.TrimSpace(request.Name)
	request.Username = strings.TrimSpace(request.Username)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Email, validation.Required, is.Email),
		validation.Field(&request.Password, validation.Required, validation.Length(8, 50)),
		validation.Field(&request.Name, validation.Required, validation.Length(1, 256)),
		validation.Field(&request.Username, validation.Required, validation.Length(1, 32), usernameLettersRule),
	); err != nil {
		return extractValidationError(err)
	}

	if err := me.userService.Register(
		c.RequestCtx(),
		request.Email,
		request.Password,
		request.Name,
		request.Username,
	); err != nil {
		if errors.Is(err, user_service.ErrEmailConflict) {
			return errEmailConflict
		}
		if errors.Is(err, user_service.ErrUsernameConflict) {
			return errUsernameConflict
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleSendVerificationEmail(c fiber.Ctx) error {
	var request struct {
		Email   string `json:"email"`
		BaseUrl string `json:"baseUrl"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.BaseUrl = strings.TrimSpace(request.BaseUrl)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Email, validation.Required, is.Email),
		validation.Field(&request.BaseUrl, validation.Required, is.URL),
	); err != nil {
		return extractValidationError(err)
	}

	if err := me.userService.SendVerificationEmail(c.RequestCtx(), request.Email, request.BaseUrl); err != nil {
		if errors.Is(err, user_service.ErrEmailNotFound) {
			return errUserNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleVerifyEmail(c fiber.Ctx) error {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		return errInvalidRequest.details("missing token query")
	}

	if err := me.userService.VerifyEmail(c.RequestCtx(), token); err != nil {
		if errors.Is(err, user_service.ErrTokenNotFound) {
			return errTokenNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleLogin(c fiber.Ctx) error {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Email, validation.Required, is.Email),
		validation.Field(&request.Password, validation.Required, validation.Length(8, 50)),
	); err != nil {
		return extractValidationError(err)
	}

	session, err := me.userService.Login(c.RequestCtx(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, user_service.ErrUnauthorized) {
			return errUnauthorized
		}
		if errors.Is(err, user_service.ErrUnverified) {
			return errEmailNotVerified
		}
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    session.Id.String(),
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    session.SessionToken,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
	})
	c.Cookie(&fiber.Cookie{
		Name:    "csrf_token",
		Value:   session.CsrfToken,
		Expires: session.ExpiresAt,
	})

	return c.SendStatus(fiber.StatusOK)
}

type WithSessionRequest struct {
	SessionId    uuid.UUID `cookie:"session_id"`
	SessionToken string    `cookie:"session_token"`
}

func (me *Handler) WithSession(c fiber.Ctx) error {
	var request WithSessionRequest
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details("malformed cookies")
	}

	if request.SessionId == uuid.Nil || request.SessionToken == "" {
		return errUnauthorized.details("missing auth cookies")
	}

	session, err := me.userService.GetSession(c.RequestCtx(), request.SessionId)
	if err != nil {
		if errors.Is(err, user_service.ErrSessionNotFound) {
			return errUnauthorized.details("invalid cookies")
		}
		return err
	}
	if !session.ExpiresAt.After(time.Now()) {
		return errUnauthorized.details("expired session")
	}

	c.Locals("session_id", session.Id)
	c.Locals("session_token", session.SessionToken)
	c.Locals("csrf_token", session.CsrfToken)
	c.Locals("user_id", session.OwnerId)

	// Session lock: protects concurrent logout (DELETE session) on the same session row.
	if c.Route().Name != "logout" {
		if err := me.lock.RLock(c.RequestCtx(), "session:"+session.Id.String()); err != nil {
			return err
		}
		defer me.lock.RUnlock(c.RequestCtx(), "session:"+session.Id.String())
	}
	// User lock: protects concurrent delete_profile (DELETE user) on the same user row.
	if c.Route().Name != "delete_profile" {
		if err := me.lock.RLock(c.RequestCtx(), "user:"+session.OwnerId.String()); err != nil {
			return err
		}
		defer me.lock.RUnlock(c.RequestCtx(), "user:"+session.OwnerId.String())
	}

	return c.Next()
}

// Request must go through [WithSession] first.
func (me *Handler) WithCsrfToken(c fiber.Ctx) error {
	if c.Locals("csrf_token").(string) != c.Get("X-CSRF-Token") {
		return errUnauthorized.details("missing or invalid CSRF token")
	}
	return c.Next()
}

func (me *Handler) HandleLogout(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)
	currentSessionId := c.Locals("session_id").(uuid.UUID)

	if err := me.lock.Lock(c.RequestCtx(), "session:"+currentSessionId.String()); err != nil {
		return err
	}
	defer me.lock.Unlock(c.RequestCtx(), "session:"+currentSessionId.String())

	if err := me.userService.Logout(c.RequestCtx(), currentUserId, currentSessionId); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleEditCredentials(c fiber.Ctx) error {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Email, validation.Required, is.Email),
		validation.Field(&request.Password, validation.Required, validation.Length(8, 50)),
	); err != nil {
		return extractValidationError(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.userService.EditCredentials(
		c.RequestCtx(),
		currentUserId,
		request.Email,
		request.Password,
	); err != nil {
		if errors.Is(err, user_service.ErrEmailConflict) {
			return errEmailConflict
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
