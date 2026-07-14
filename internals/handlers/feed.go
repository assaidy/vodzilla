package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleGetFeed(c fiber.Ctx) error {
	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	ids, err := me.socialService.GetAllFollowedIds(c.RequestCtx(), currentUserId)
	if err != nil {
		return err
	}

	videos, err := me.videoService.GetVideosForMultipleUsers(
		c.RequestCtx(),
		ids,
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
			OwnerId:     v.OwnerId,
			Timestamp:   v.Timestamp,
			Title:       v.Title,
			Description: v.Description,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(videos[len(videos)-1].Id)
	}

	return c.JSON(response)
}
