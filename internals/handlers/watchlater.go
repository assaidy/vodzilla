package handlers

import (
	"errors"

	"github.com/assaidy/hyper/v2"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func HandleAddToWatchLater(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.AddVideoToWatchlater(c.RequestCtx(), videoId, userId); err != nil {
		if errors.Is(err, video_service.ErrWatchlaterConflict) {
			return fiber.NewError(fiber.StatusConflict, "already in watch later")
		}
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		}
		return err
	}

	return render(c, templates.WatchLaterButton(templates.WatchLaterButtonParams{
		VideoId:  videoId,
		IsActive: true,
	}))
}

func HandleDeleteFromWatchLater(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.DeleteVideoFromWatchlater(c.RequestCtx(), videoId, userId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "not in watch later")
		}
		return err
	}

	return render(c, templates.WatchLaterButton(templates.WatchLaterButtonParams{
		VideoId:  videoId,
		IsActive: false,
	}))
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

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videos, err := videoService.GetAllVideosInWatchlater(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)

	ownerCache := make(map[uuid.UUID]*user_service.User)
	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		owner, ok := ownerCache[v.OwnerId]
		if !ok {
			owner, err = userService.GetUserById(c.RequestCtx(), v.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return err
			}
			ownerCache[v.OwnerId] = owner
		}

		viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			ViewsCount:    viewsCount,
		})
	}

	return render(c, hyper.Group(
		templates.WatchLaterPageContent(templates.WatchLaterPageContentParams{
			Username: currentUser.Username,
			Videos:   templateVideos,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageWatchLater,
			}),
		),
	))
}
