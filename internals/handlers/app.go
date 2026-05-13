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

	return render(c, templates.FeedPage(user.Username))
}

func HandleFeedPageContent(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.FeedPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    user.Username,
				CurrentPage: templates.PageFeed,
			}),
		),
	))
}

func HandleDiscoverPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.DiscoverPage(user.Username))
}

func HandleDiscoverPageContent(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.DiscoverPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    user.Username,
				CurrentPage: templates.PageDiscover,
			}),
		),
	))
}

func HandleWatchLaterPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.WatchLaterPage(user.Username))
}

func HandleWatchLaterPageContent(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.WatchLaterPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    user.Username,
				CurrentPage: templates.PageWatchLater,
			}),
		),
	))
}

func HandlePlaylistsPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.PlaylistsPage(user.Username))
}

func HandlePlaylistsPageContent(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.PlaylistsPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    user.Username,
				CurrentPage: templates.PagePlaylists,
			}),
		),
	))
}

func HandleNotificationsPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.NotificationsPage(user.Username))
}

func HandleNotificationsPageContent(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.NotificationsPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    user.Username,
				CurrentPage: templates.PageNotifications,
			}),
		),
	))
}

func HandleProfilePage(c fiber.Ctx) error {
	user, currentUser, err := getProfileUserAndCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.ProfilePage(templates.ProfilePageContentParams{
		Username: user.Username,
		Name:     user.Name,
		Bio:      user.Bio,
		IsOwner:  user.Username == currentUser.Username,
	}))
}

func HandleProfilePageContent(c fiber.Ctx) error {
	user, currentUser, err := getProfileUserAndCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.ProfilePageContent(templates.ProfilePageContentParams{
			Username: user.Username,
			Name:     user.Name,
			Bio:      user.Bio,
			IsOwner:  user.Username == currentUser.Username,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageProfile,
			}),
		),
	))
}

func getProfileUserAndCurrentUser(c fiber.Ctx) (*user_service.User, *user_service.User, error) {
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	user, err := userService.GetUserByUsername(c.RequestCtx(), c.Params("username"))
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return nil, nil, fiber.ErrNotFound
		}
		return nil, nil, fmt.Errorf("failed to get profile user: %w", err)
	}

	currentUser, err := getCurrentUser(c)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return user, currentUser, nil
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

	userId := c.Locals("user_id").(string)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	if err := userService.EditProfile(c.RequestCtx(), userId, name, username, bio); err != nil {
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

	c.Set("HX-Replace-Url", fmt.Sprintf("/@%s", username))
	return render(c, hyper.Group(
		templates.EditProfileFrom(templates.EditProfileFromParams{
			Name:     name,
			Username: username,
			Bio:      bio,
		}),

		hyper.H1(hyper.AttrId("PROFILE_CARD_NAME"), hyper.Attr("hx-swap-oob", "innerHTML"))(name),
		hyper.P(hyper.AttrId("PROFILE_CARD_USERNAME"), hyper.Attr("hx-swap-oob", "innerHTML"))("@"+username),
		hyper.P(hyper.AttrId("PROFILE_CARD_BIO"), hyper.Attr("hx-swap-oob", "innerHTML"))(bio),
		templates.Alert(templates.AlertInfo, "Profile was updated successfully."),
	))
}

func getCurrentUser(c fiber.Ctx) (*user_service.User, error) {
	userId := c.Locals("user_id").(string)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	user, err := userService.GetUserById(c.RequestCtx(), userId)
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return nil, redirect(c, "/login")
		}
		return nil, err
	}

	return user, nil
}
