package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

// TODO: move form validation to handlers

func HandleRegisterPage(c fiber.Ctx) error {
	return render(c, templates.RegisterPage())
}

func HandleRegister(c fiber.Ctx) error {
	name := c.FormValue("name")
	username := c.FormValue("username")
	email := c.FormValue("email")
	password := c.FormValue("password")

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	if err := userService.Register(c.RequestCtx(), email, password, name, username); err != nil {
		if errors.Is(err, user_service.ErrValidation) {
			if validationErrs, ok := errors.AsType[user_service.RegisterValidationErrors](err); !ok {
				panic("expected user.RegisterValidationErrors")
			} else {
				return render(c, templates.RegisterForm(templates.RegisterFormParams{
					Name:        name,
					NameErr:     validationErrs.Name,
					Username:    username,
					UsernameErr: validationErrs.Username,
					Email:       email,
					EmailErr:    validationErrs.Email,
					Password:    password,
					PasswordErr: validationErrs.Password,
				}))
			}
		}
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
	if err := userService.SendVerificationEmail(c.RequestCtx(), email, url); err != nil {
		return err
	}

	return redirect(c, "/verification_email/sent")
}

func HandleVerificationEmailSentPage(c fiber.Ctx) error {
	return render(c, templates.VerificationEmailSentPage())
}

func HandleVerifyEmailPage(c fiber.Ctx) error {
	token := fiber.Query[string](c, "token")

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	if err := userService.VerifyEmail(c.RequestCtx(), token); err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return render(c, templates.InvalidVerificationLinkPage())
		}
		return err
	}

	return render(c, templates.EmailVerifiedPage())
}

func HandleLoginPage(c fiber.Ctx) error {
	return render(c, templates.LoginPage())
}

func HandleLogin(c fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	session, err := userService.Login(c.RequestCtx(), email, password)
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
		Value:    session.Id,
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

func WithSession(c fiber.Ctx) error {
	sessionId := c.Cookies("session_id")
	sessionToken := c.Cookies("session_token")

	if sessionId == "" || sessionToken == "" {
		return redirect(c, "/login")
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	session, err := userService.GetSession(c.RequestCtx(), sessionId)
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
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
func WithCsrfToken(c fiber.Ctx) error {
	if c.Locals("csrf_token").(string) != c.Get("X-CSRF-Token") {
		return fiber.ErrForbidden
	}

	return c.Next()
}
