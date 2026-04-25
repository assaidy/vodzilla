package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/assaidy/video_streaming_app/internals/services/user"
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

	userService := fiber.MustGetState[*user.Service](c.App().State(), user.Name)

	if err := userService.Register(c, email, password, name, username); err != nil {
		if errors.Is(err, user.ErrValidation) {
			if validationErrs, ok := errors.AsType[user.RegisterValidationErrors](err); !ok {
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
		if errors.Is(err, user.ErrEmailConflict) {
			return render(c, templates.RegisterForm(templates.RegisterFormParams{
				Name:     name,
				Username: username,
				Email:    email,
				EmailErr: errors.New("email already exists"),
				Password: password,
			}))
		}
		if errors.Is(err, user.ErrUsernameConflict) {
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
	if err := userService.SendVerificationEmail(c, email, url); err != nil {
		return err
	}

	return c.Redirect().To("/verification_email/sent")
}

func HandleVerificationEmailSentPage(c fiber.Ctx) error {
	return render(c, templates.VerificationEmailSentPage())
}

func HandleVerifyEmailPage(c fiber.Ctx) error {
	token := fiber.Query[string](c, "token")

	userService := fiber.MustGetState[*user.Service](c.App().State(), user.Name)
	if err := userService.VerifyEmail(c, token); err != nil {
		if errors.Is(err, user.ErrNotFound) {
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
	userService := fiber.MustGetState[*user.Service](c.App().State(), user.Name)
	session, err := userService.Login(c, email, password)
	if err != nil {
		if errors.Is(err, user.ErrUnauthorized) {
			return render(c, templates.LoginForm(templates.LoginFormParams{
				Email:    email,
				Password: password,
				Err:      templates.ErrInvalidCredentials,
			}))
		}
		if errors.Is(err, user.ErrUnverified) {
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

	userService := fiber.MustGetState[*user.Service](c.App().State(), user.Name)
	session, err := userService.GetSession(c, sessionID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
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
	c.Locals("userID", session.OwnerID)

	return c.Next()
}

// must go through [WithSessionToken] first
func WithCsrfToken(c fiber.Ctx) error {
	if c.Locals("csrfToken").(string) != c.Get("X-CSRF-Token") {
		return fiber.ErrForbidden
	}

	return c.Next()
}
