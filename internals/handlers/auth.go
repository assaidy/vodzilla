package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/assaidy/video_streaming_app/internals/services/auth"
	"github.com/assaidy/video_streaming_app/internals/utils"
	"github.com/assaidy/video_streaming_app/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func HandleRegisterPage(c fiber.Ctx) error {
	return render(c, templates.RegisterPage())
}

func HandleRegister(c fiber.Ctx) error {
	name := c.FormValue("name")
	username := c.FormValue("username")
	email := c.FormValue("email")
	password := c.FormValue("password")

	authService := fiber.MustGetState[*auth.Service](c.App().State(), auth.Name)

	userID, err := authService.Register(c, email, password)
	if err != nil {
		if errors.Is(err, auth.ErrValidation) {
			if validationErrs, ok := errors.AsType[auth.RegisterValidationErrors](err); !ok {
				panic("expected auth.RegisterValidationErrors")
			} else {
				return render(c, templates.RegisterForm(templates.RegisterFormParams{
					Name:        name,
					Username:    username,
					Email:       email,
					EmailErr:    validationErrs.Email,
					Password:    password,
					PasswordErr: validationErrs.Password,
				}))
			}
		}
		if errors.Is(err, auth.ErrEmailConflict) {
			return render(c, templates.RegisterForm(templates.RegisterFormParams{
				Name:     name,
				Username: username,
				Email:    email,
				EmailErr: errors.New("email already exists"),
				Password: password,
			}))
		}
		return err
	}

	url, err := url.JoinPath(utils.MustGetEnv("APP_BASE_URL"), "/verification_email/verify")
	if err != nil {
		return fmt.Errorf("failed to general email verification url")
	}
	if err := authService.SendVerificationEmail(c, email, url); err != nil {
		return err
	}

	// TODO: create a profile using profile service
	_ = userID

	return c.Redirect().To("/verification_email/sent")
}

func HandleVerificationEmailSentPage(c fiber.Ctx) error {
	return render(c, templates.VerificationEmailSentPage())
}

func HandleVerifyEmailPage(c fiber.Ctx) error {
	token := fiber.Query[string](c, "token")

	authService := fiber.MustGetState[*auth.Service](c.App().State(), auth.Name)
	if err := authService.VerifyEmail(c, token); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
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
	authService := fiber.MustGetState[*auth.Service](c.App().State(), auth.Name)
	session, err := authService.Login(c, email, password)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			return render(c, templates.LoginForm(templates.LoginFormParams{
				Email:    email,
				Password: password,
				Err:      templates.ErrInvalidCredentials,
			}))
		}
		if errors.Is(err, auth.ErrUnverified) {
			return render(c, templates.LoginForm(templates.LoginFormParams{
				Email:    email,
				Password: password,
				Err:      templates.ErrEmailNotVerified,
			}))
		}
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     "sessionID",
		Value:    session.ID,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "sessionToken",
		Value:    session.SessionToken,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
	})
	c.Cookie(&fiber.Cookie{
		Name:    "csrfToken",
		Value:   session.CsrfToken,
		Expires: session.ExpiresAt,
	})

	return c.Redirect().To("/")
}

func WithSessionToken(c fiber.Ctx) error {
	sessionID := c.Cookies("sessionID")
	sessionToken := c.Cookies("sessionToken")

	if sessionID == "" || sessionToken == "" {
		return c.Redirect().To("/login")
	}

	authService := fiber.MustGetState[*auth.Service](c.App().State(), auth.Name)
	session, err := authService.GetSession(c, sessionID)
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			return c.Redirect().To("/login")
		}
		return err
	}
	if !session.ExpiresAt.After(time.Now()) {
		return c.Redirect().To("/login")
	}

	c.Locals("sessionID", session.ID)
	c.Locals("sessionToken", session.SessionToken)
	c.Locals("csrfToken", session.CsrfToken)

	return c.Next()
}

// must go through [WithSessionToken] first
func WithCsrfToken(c fiber.Ctx) error {
	if c.Locals("csrfToken").(string) != c.Get("X-CSRF-Token") {
		return fiber.ErrForbidden
	}

	return c.Next()
}
