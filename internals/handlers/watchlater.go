package handlers

import (
	"errors"

	"github.com/assaidy/hyper/v2"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleAddToWatchLater(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.AddVideoToWatchlater(c.RequestCtx(), videoId, userId); err != nil {
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

func (me *Handler) HandleDeleteFromWatchLater(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.DeleteVideoFromWatchlater(c.RequestCtx(), videoId, userId); err != nil {
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

func (me *Handler) HandleWatchLaterPage(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.WatchLaterPage(currentUser.Username))
}

func (me *Handler) HandleWatchLaterPageContent(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	videos, err := me.videoService.GetAllVideosInWatchlater(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	ownerCache := make(map[uuid.UUID]*user_service.User)
	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		owner, ok := ownerCache[v.OwnerId]
		if !ok {
			owner, err = me.userService.GetUserById(c.RequestCtx(), v.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return err
			}
			ownerCache[v.OwnerId] = owner
		}

		viewsCount, err := me.reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
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
