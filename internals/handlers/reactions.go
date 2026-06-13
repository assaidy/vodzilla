package handlers

import (
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleViewVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	userId := c.Locals("user_id").(uuid.UUID)

	me.videoMutex.RLock(videoId.String())
	defer me.videoMutex.RUnlock(videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	if err := me.reactionService.ViewVideo(c.RequestCtx(), videoId, userId); err != nil {
		return err
	}

	return nil
}

const (
	ReactionLike    = "like"
	ReactionDislike = "dislike"
)

func (me *Handler) HandleAddVideoReaction(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	userId := c.Locals("user_id").(uuid.UUID)
	kind := c.Query("kind")

	if kind != ReactionLike && kind != ReactionDislike {
		return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
	}

	me.videoMutex.RLock(videoId.String())
	defer me.videoMutex.RUnlock(videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	if err := me.reactionService.AddVidoeReaction(c.RequestCtx(), videoId, userId, kind); err != nil {
		return err
	}

	reactinCounts, err := me.reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
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

func (me *Handler) HandleDeleteVideoReaction(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	userId := c.Locals("user_id").(uuid.UUID)
	kind := c.Query("kind")

	if kind != ReactionLike && kind != ReactionDislike {
		return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
	}

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	if err := me.reactionService.DeleteVidoeReaction(c.RequestCtx(), videoId, userId, kind); err != nil {
		return err
	}

	reactinCounts, err := me.reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	return render(c, templates.ReactionsWidget(templates.ReactionsWidgetParams{
		VideoId:       videoId,
		LikesCount:    reactinCounts.Likes,
		DislikesCount: reactinCounts.Dislikes,
	}))
}
