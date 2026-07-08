package handlers

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleGetFeed(c fiber.Ctx) error {
	var request struct {
		LastVideoId uuid.UUID `query:"last_video_id"`
		Limit       int       `query:"limit"`
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

	// TODO: Find a better system for feeds.
	ids, err := me.socialService.GetAllFollowedIds(c.RequestCtx(), currentUserId)
	if err != nil {
		return err
	}

	videos, err := me.videoService.GetVideosForMultipleUsers(c.RequestCtx(), ids, request.LastVideoId, request.Limit)
	if err != nil {
		return err
	}

	response := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		response = append(response, videoResponse{
			Id:          v.Id,
			OwnerId:     v.OwnerId,
			Timestamp:   v.Timestamp,
			Title:       v.Title,
			Description: v.Description,
		})
	}

	return c.JSON(response)
}
