package handlers

import (
	"errors"

	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) handleGetWatchlaters(c fiber.Ctx) error {
	pr, err := parsePaginatedRequest[int](c)
	if err != nil {
		return err
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	videos, err := me.videoService.GetVideosInWatchlater(
		c.RequestCtx(),
		currentUserId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		items = append(items, videoResponse{
			Id:          v.Id,
			UserId:      v.UserId,
			Timestamp:   v.Timestamp,
			Title:       v.Title,
			Description: v.Description,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(videos[len(videos)-1].WatchlaterVideoId)
	}

	return c.JSON(response)
}

func (me *Handler) handleAddToWatchLaters(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}
	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.AddVideoToWatchlater(c.RequestCtx(), videoId, currentUserId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		if errors.Is(err, video_service.ErrWatchlaterConflict) {
			return errWatchlaterConflict
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) handleDeleteFromWatchLaters(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}
	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.DeleteVideoFromWatchlater(c.RequestCtx(), videoId, currentUserId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		if errors.Is(err, video_service.ErrWatchlaterVideoNotFound) {
			return errWatchlaterVideoNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
