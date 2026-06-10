package handlers

import (
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func HandleViewVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if ok, err := videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.ViewVideo(c.RequestCtx(), videoId, userId); err != nil {
		return err
	}

	return nil
}

const (
	ReactionLike    = "like"
	ReactionDislike = "dislike"
)

func HandleAddVideoReaction(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)
	kind := c.Query("kind")

	if kind != ReactionLike && kind != ReactionDislike {
		return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if ok, err := videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.AddVidoeReaction(c.RequestCtx(), videoId, userId, kind); err != nil {
		return err
	}

	reactinCounts, err := reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	return render(c, templates.ReactionsWidget(templates.ReactionsWidgetParams{
		VideoId:       videoId,
		LikesCount:    reactinCounts.Likes,
		DislikesCount: reactinCounts.Dislikes,
		IsLiked:       kind == ReactionLike,
		IsDisliked:    kind == ReactionDislike,
	}))
}

func HandleDeleteVideoReaction(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)
	kind := c.Query("kind")

	if kind != ReactionLike && kind != ReactionDislike {
		return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if ok, err := videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.DeleteVidoeReaction(c.RequestCtx(), videoId, userId, kind); err != nil {
		return err
	}

	reactinCounts, err := reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	return render(c, templates.ReactionsWidget(templates.ReactionsWidgetParams{
		VideoId:       videoId,
		LikesCount:    reactinCounts.Likes,
		DislikesCount: reactinCounts.Dislikes,
	}))
}
