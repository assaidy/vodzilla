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

func (me *Handler) handleRegister(c fiber.Ctx) error {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
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
		return errInvalidData.details(err)
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

func (me *Handler) handleSendVerificationEmail(c fiber.Ctx) error {
	var request struct {
		Email   string `json:"email"`
		BaseUrl string `json:"baseUrl"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.BaseUrl = strings.TrimSpace(request.BaseUrl)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Email, validation.Required, is.Email),
		validation.Field(&request.BaseUrl, validation.Required, is.URL),
	); err != nil {
		return errInvalidData.details(err)
	}

	if err := me.userService.SendVerificationEmail(c.RequestCtx(), request.Email, request.BaseUrl); err != nil {
		if errors.Is(err, user_service.ErrEmailNotFound) {
			return errUserNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) handleVerifyEmail(c fiber.Ctx) error {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		return errTokenNotFound
	}

	if err := me.userService.VerifyEmail(c.RequestCtx(), token); err != nil {
		if errors.Is(err, user_service.ErrTokenNotFound) {
			return errTokenNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) handleLogin(c fiber.Ctx) error {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Email, validation.Required, is.Email),
		validation.Field(&request.Password, validation.Required, validation.Length(8, 50)),
	); err != nil {
		return errInvalidData.details(err)
	}

	session, err := me.userService.Login(c.RequestCtx(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, user_service.ErrUnauthorized) {
			return errUnauthorized
		}
		if errors.Is(err, user_service.ErrEmailNotVerified) {
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

func (me *Handler) withSession(c fiber.Ctx) error {
	sessionIdStr := c.Cookies("session_id")
	sessionToken := c.Cookies("session_token")

	sessionId, err := uuid.Parse(sessionIdStr)
	if err != nil {
		return errUnauthorized.details("missing or invalid auth cookies")
	}

	if sessionToken == "" {
		return errUnauthorized.details("missing auth cookies")
	}

	session, err := me.userService.GetSession(c.RequestCtx(), sessionId)
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
	c.Locals("user_id", session.UserId)

	// Session lock: protects concurrent logout (DELETE session) on the same session row.
	sessionLock := me.newSessionLock(sessionId)
	if c.Route().Name != "logout" {
		if err := sessionLock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
			return err
		}
		defer sessionLock.RUnLock(c.RequestCtx())
	}
	// User lock: protects concurrent delete_profile (DELETE user) on the same user row.
	userLock := me.newUserLock(session.UserId)
	if c.Route().Name != "delete_profile" {
		if err := userLock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
			return err
		}
		defer userLock.RUnLock(c.RequestCtx())
	}

	return c.Next()
}

// Request must go through [WithSession] first.
func (me *Handler) withCsrfToken(c fiber.Ctx) error {
	if c.Locals("csrf_token").(string) != c.Get("X-CSRF-Token") {
		return errUnauthorized.details("missing or invalid CSRF token")
	}
	return c.Next()
}

func (me *Handler) handleLogout(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)
	currentSessionId := c.Locals("session_id").(uuid.UUID)

	lock := me.newSessionLock(currentSessionId)
	if err := lock.SpinWLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.WUnLock(c.RequestCtx())

	if err := me.userService.Logout(c.RequestCtx(), currentUserId, currentSessionId); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) handleEditPassword(c fiber.Ctx) error {
	var request struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.CurrentPassword, validation.Required, validation.Length(8, 50)),
		validation.Field(&request.NewPassword, validation.Required, validation.Length(8, 50)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.userService.EditPassword(
		c.RequestCtx(),
		currentUserId,
		request.CurrentPassword,
		request.NewPassword,
	); err != nil {
		if errors.Is(err, user_service.ErrUnauthorized) {
			return errInvalidPassword
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
