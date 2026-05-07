package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/assaidy/hyper/v2"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
)

func HandleHomePage(c fiber.Ctx) error {
	return redirect(c, "/feed")
}

func HandleFeedPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.FeedPage(templates.NavbarProfile{
		Username: user.Username,
	}))
}

func HandleDiscoverPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.DiscoverPage(templates.NavbarProfile{
		Username: user.Username,
	}))
}

func HandleWatchLaterPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.WatchLaterPage(templates.NavbarProfile{
		Username: user.Username,
	}))
}

func HandlePlaylistsPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.PlaylistsPage(templates.NavbarProfile{
		Username: user.Username,
	}))
}

func HandleNotificationsPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.NotificationsPage(templates.NavbarProfile{
		Username: user.Username,
	}))
}

func HandleProfilePage(c fiber.Ctx) error {
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	user, err := userService.GetUserByUsername(c.RequestCtx(), c.Params("username"))
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	return render(c, templates.ProfilePage(templates.ProfilePageParams{
		NavbarProfile: templates.NavbarProfile{
			Username: user.Username,
		},
		Name:     user.Name,
		Username: user.Username,
		Bio:      user.Bio,
		IsOwner:  user.ID == c.Locals("user_id"),
	}))
}

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]*$`)

func HandleEditProfile(c fiber.Ctx) error {
	name := strings.TrimSpace(c.FormValue("name"))
	username := strings.TrimSpace(c.FormValue("username"))
	bio := strings.TrimSpace(c.FormValue("bio"))

	nameErr := validation.Validate(&name, validation.Required, validation.Length(1, 256))
	usernameErr := validation.Validate(&username, validation.Required, validation.Length(1, 32),
		validation.Match(usernameRegex).Error("can only cotain letters, digits or _"))
	bioErr := validation.Validate(&bio, validation.Length(0, 500))

	if errors.Join(nameErr, usernameErr, bioErr) != nil {
		return render(c, templates.EditProfileFrom(templates.EditProfileFromParams{
			Name:        name,
			NameErr:     nameErr,
			Username:    username,
			UsernameErr: usernameErr,
			Bio:         bio,
			BioErr:      bioErr,
		}))
	}

	userID := c.Locals("user_id").(string)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	if err := userService.EditProfile(c.RequestCtx(), userID, name, username, bio); err != nil {
		switch {
		case errors.Is(err, fiber.ErrNotFound):
			return redirect(c, "/login")
		case errors.Is(err, user_service.ErrUsernameConflict):
			return render(c, templates.EditProfileFrom(templates.EditProfileFromParams{
				Name:        name,
				Username:    username,
				UsernameErr: fmt.Errorf("username already exists"),
				Bio:         bio,
			}))
		default:
			return err
		}
	}

	c.Set("HX-Replace-Url", fmt.Sprintf("/profiles/%s", username))
	return render(c, hyper.Group(
		templates.EditProfileFrom(templates.EditProfileFromParams{
			Name:     name,
			Username: username,
			Bio:      bio,
		}),

		hyper.H1(hyper.AttrID("profileCardName"), hyper.Attr("hx-swap-oob", "innerHTML"))(name),
		hyper.P(hyper.AttrID("profileCardUsername"), hyper.Attr("hx-swap-oob", "innerHTML"))("@"+username),
		hyper.P(hyper.AttrID("profileCardBio"), hyper.Attr("hx-swap-oob", "innerHTML"))(bio),
		templates.Alert(templates.AlertInfo, "Profile was updated successfully."),
	))
}

func getCurrentUser(c fiber.Ctx) (*user_service.User, error) {
	userID := c.Locals("user_id").(string)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	user, err := userService.GetUserByID(c.RequestCtx(), userID)
	if err != nil {
		switch {
		case errors.Is(err, user_service.ErrNotFound):
			return nil, redirect(c, "/login")
		default:
			return nil, err
		}
	}

	return user, nil
}
