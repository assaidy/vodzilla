package handlers

import (
	"errors"

	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/web/templates"
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

	return render(c, templates.FeedPage(templates.PageLayoutProfile{
		Username: user.Username,
	}))
}

func HandleDiscoverPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.DiscoverPage(templates.PageLayoutProfile{
		Username: user.Username,
	}))
}

func HandleWatchLaterPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.WatchLaterPage(templates.PageLayoutProfile{
		Username: user.Username,
	}))
}

func HandlePlaylistsPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.PlaylistsPage(templates.PageLayoutProfile{
		Username: user.Username,
	}))
}

func HandleNotificationsPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.NotificationsPage(templates.PageLayoutProfile{
		Username: user.Username,
	}))
}

func HandleProfilePage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.ProfilePage(templates.PageLayoutProfile{
		Username: user.Username,
	}))
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
