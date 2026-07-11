package handlers

import (
	"errors"

	video_service "github.com/assaidy/vodzilla/internals/services/video"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleGetWatchlaters(c fiber.Ctx) error {
	var request struct {
		LastId int64 `query:"last_id"`
		Limit  int   `query:"limit"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if request.Limit == 0 {
		request.Limit = 15
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Limit, validation.Min(15), validation.Max(100)),
	); err != nil {
		return extractValidationError(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	videos, err := me.videoService.GetVideosInWatchlater(
		c.RequestCtx(),
		currentUserId,
		request.LastId,
		request.Limit,
	)
	if err != nil {
		return err
	}

	response := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		response = append(response, videoResponse{
			Id:                v.Id,
			OwnerId:           v.OwnerId,
			Timestamp:         v.Timestamp,
			Title:             v.Title,
			Description:       v.Description,
			WatchlaterVideoId: v.WatchlaterVideoId,
		})
	}

	return c.JSON(response)
}

func (me *Handler) HandleAddToWatchLaters(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}
	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.AddVideoToWatchlater(c.RequestCtx(), videoId, userId); err != nil {
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

func (me *Handler) HandleDeleteFromWatchLaters(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}
	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.DeleteVideoFromWatchlater(c.RequestCtx(), videoId, userId); err != nil {
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
