package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]*$`)

func (me *Handler) HandleRegisterPage(c fiber.Ctx) error {
	return render(c, templates.RegisterPage())
}

func (me *Handler) HandleRegister(c fiber.Ctx) error {
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	password := c.FormValue("password")
	name := strings.TrimSpace(c.FormValue("name"))
	username := strings.TrimSpace(c.FormValue("username"))

	emailErr := validation.Validate(email, validation.Required, is.Email)
	passwordErr := validation.Validate(password, validation.Required, validation.Length(8, 50))
	nameErr := validation.Validate(name, validation.Required, validation.Length(1, 256))
	usernameErr := validation.Validate(username, validation.Required, validation.Length(1, 32),
		validation.Match(usernameRegex).Error("can only contain letters, digits or _"))

	if errors.Join(emailErr, passwordErr, nameErr, usernameErr) != nil {
		return render(c, templates.RegisterForm(templates.RegisterFormParams{
			Name:        name,
			NameErr:     nameErr,
			Username:    username,
			UsernameErr: usernameErr,
			Email:       email,
			EmailErr:    emailErr,
			Password:    password,
			PasswordErr: passwordErr,
		}))
	}

	if err := me.userService.Register(c.RequestCtx(), email, password, name, username); err != nil {
		if errors.Is(err, user_service.ErrEmailConflict) {
			return render(c, templates.RegisterForm(templates.RegisterFormParams{
				Name:     name,
				Username: username,
				Email:    email,
				EmailErr: errors.New("email already exists"),
				Password: password,
			}))
		}
		if errors.Is(err, user_service.ErrUsernameConflict) {
			return render(c, templates.RegisterForm(templates.RegisterFormParams{
				Name:        name,
				Username:    username,
				UsernameErr: errors.New("username already exists"),
				Email:       email,
				Password:    password,
			}))
		}
		return err
	}

	url, err := url.JoinPath(utils.MustGetEnv("APP_BASE_URL"), "/verification_email/verify")
	if err != nil {
		return fmt.Errorf("failed to general email verification url")
	}
	if err := me.userService.SendVerificationEmail(c.RequestCtx(), email, url); err != nil {
		return err
	}

	return redirect(c, "/verification_email/sent")
}

func (me *Handler) HandleVerificationEmailSentPage(c fiber.Ctx) error {
	return render(c, templates.VerificationEmailSentPage())
}

func (me *Handler) HandleVerifyEmailPage(c fiber.Ctx) error {
	token := fiber.Query[string](c, "token")

	if err := me.userService.VerifyEmail(c.RequestCtx(), token); err != nil {
		if errors.Is(err, user_service.ErrTokenNotFound) {
			return render(c, templates.InvalidVerificationLinkPage())
		}
		return err
	}

	return render(c, templates.EmailVerifiedPage())
}

func (me *Handler) HandleLoginPage(c fiber.Ctx) error {
	return render(c, templates.LoginPage())
}

func (me *Handler) HandleLogin(c fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	session, err := me.userService.Login(c.RequestCtx(), email, password)
	if err != nil {
		if errors.Is(err, user_service.ErrUnauthorized) {
			return render(c, templates.LoginForm(templates.LoginFormParams{
				Email:    email,
				Password: password,
				Err:      templates.ErrInvalidCredentials,
			}))
		}
		if errors.Is(err, user_service.ErrUnverified) {
			return render(c, templates.LoginForm(templates.LoginFormParams{
				Email:    email,
				Password: password,
				Err:      templates.ErrEmailNotVerified,
			}))
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

	return redirect(c, "/")
}

func (me *Handler) WithSession(c fiber.Ctx) error {
	sessionIdStr := c.Cookies("session_id")
	sessionToken := c.Cookies("session_token")

	if sessionIdStr == "" || sessionToken == "" {
		return redirect(c, "/login")
	}

	sessionId, err := uuid.Parse(sessionIdStr)
	if err != nil {
		return redirect(c, "/login")
	}

	session, err := me.userService.GetSession(c.RequestCtx(), sessionId)
	if err != nil {
		if errors.Is(err, user_service.ErrSessionNotFound) {
			return redirect(c, "/login")
		}
		return err
	}
	if !session.ExpiresAt.After(time.Now()) {
		return redirect(c, "/login")
	}

	c.Locals("session_id", session.Id)
	c.Locals("session_token", session.SessionToken)
	c.Locals("csrf_token", session.CsrfToken)
	c.Locals("user_id", session.OwnerId)

	return c.Next()
}

// must go through [WithSession] first
func (me *Handler) WithCsrfToken(c fiber.Ctx) error {
	if c.Locals("csrf_token").(string) != c.Get("X-CSRF-Token") {
		return fiber.NewError(fiber.StatusForbidden, "missing CSRF token")
	}
	return c.Next()
}
