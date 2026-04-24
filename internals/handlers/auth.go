package handlers

import (
	"errors"
	"fmt"
	"net/url"

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

func HandleLoginPage(c fiber.Ctx) error {
	return render(c, templates.LoginPage())
}

func HandleLogin(c fiber.Ctx) error {
	println(c.FormValue("email"))
	println(c.FormValue("password"))
	return nil
}
