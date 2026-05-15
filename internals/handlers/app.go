package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/assaidy/hyper/v2"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
)

func HandleHomePage(c fiber.Ctx) error {
	return redirect(c, "/feed")
}

func HandleFeedPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.FeedPage(currentUser.Username))
}

func HandleFeedPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.FeedPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageFeed,
			}),
		),
	))
}

func HandleDiscoverPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.DiscoverPage(currentUser.Username))
}

func HandleDiscoverPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.DiscoverPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageDiscover,
			}),
		),
	))
}

func HandleWatchLaterPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.WatchLaterPage(currentUser.Username))
}

func HandleWatchLaterPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.WatchLaterPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageWatchLater,
			}),
		),
	))
}

func HandlePlaylistsPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.PlaylistsPage(currentUser.Username))
}

func HandlePlaylistsPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.PlaylistsPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PagePlaylists,
			}),
		),
	))
}

func HandleNotificationsPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.NotificationsPage(currentUser.Username))
}

func HandleNotificationsPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.NotificationsPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
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

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videos, err := videoService.GetUserVideos(c.RequestCtx(), user.Id)
	if err != nil {
		return err
	}

	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     user.Name,
			OwnerUsername: user.Username,
		})
	}

	return render(c, templates.ProfilePage(templates.ProfilePageContentParams{
		Username: user.Username,
		Name:     user.Name,
		Bio:      user.Bio,
		IsOwner:  user.Username == currentUser.Username,
		Videos:   templateVideos,
	}))
}

func HandleProfilePageContent(c fiber.Ctx) error {
	user, currentUser, err := getProfileUserAndCurrentUser(c)
	if err != nil {
		return err
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videos, err := videoService.GetUserVideos(c.RequestCtx(), user.Id)
	if err != nil {
		return err
	}

	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     user.Name,
			OwnerUsername: user.Username,
		})
	}

	return render(c, hyper.Group(
		templates.ProfilePageContent(templates.ProfilePageContentParams{
			Username: user.Username,
			Name:     user.Name,
			Bio:      user.Bio,
			IsOwner:  user.Username == currentUser.Username,
			Videos:   templateVideos,
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

func HandleVideoPage(c fiber.Ctx) error {
	user, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	videoId := c.Params("video_id")

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	video, err := videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	owner, err := userService.GetUserById(c.RequestCtx(), video.OwnerId)
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	sourceUrl, err := mediaService.GeneratePresignedGetUrl(c.RequestCtx(), video.ObjectKey)
	if err != nil {
		return err
	}

	return render(c, templates.VideoPage(templates.VideoPageParams{
		Username: user.Username,
		ContentParams: templates.VideoPageContentParams{
			Id:            video.Id,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			SourceUrl:     sourceUrl,
			Title:         video.Title,
			Description:   video.Description,
			Timestamp:     video.Timestamp,
		},
	}))
}

func HandleVideoPageContent(c fiber.Ctx) error {
	videoId := c.Params("video_id")

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	video, err := videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	owner, err := userService.GetUserById(c.RequestCtx(), video.OwnerId)
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	sourceUrl, err := mediaService.GeneratePresignedGetUrl(c.RequestCtx(), video.ObjectKey)
	if err != nil {
		return err
	}

	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.VideoPageContent(templates.VideoPageContentParams{
			Id:            video.Id,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			SourceUrl:     sourceUrl,
			Title:         video.Title,
			Description:   video.Description,
			Timestamp:     video.Timestamp,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username: currentUser.Username,
			}),
		),
	))
}
