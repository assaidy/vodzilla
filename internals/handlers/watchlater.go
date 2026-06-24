package handlers

import (
	"errors"

	"github.com/assaidy/hyper/v2"
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
		switch {
		case errors.Is(err, video_service.ErrWatchlaterConflict):
			return fiber.NewError(fiber.StatusConflict, "already in watch later")
		case errors.Is(err, video_service.ErrVideoNotFound):
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		default:
			return err
		}
	}

	return render(c, templates.WatchlaterButton(templates.WatchlaterButtonParams{
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

	return render(c, templates.WatchlaterButton(templates.WatchlaterButtonParams{
		VideoId:  videoId,
		IsActive: false,
	}))
}

func (me *Handler) HandleWatchLaterPage(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.WatchlaterPage(currentUser.Username))
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

	templateVideos, err := me.toTemplateVideos(c, videos)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.WatchlaterPageContent(templates.WatchlaterPageContentParams{
			Username: currentUser.Username,
			Videos:   templateVideos,
		}),

		hyper.DIV(hyper.AttrId("navbar"), templates.AttrHxSwapOob(templates.SwapOuterHtml))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageWatchLater,
			}),
		),
	))
}
