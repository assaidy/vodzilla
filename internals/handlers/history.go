package handlers

import (
	"errors"
	"time"

	history_service "github.com/assaidy/vodzilla/internals/services/history"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type watchHistoryResponse struct {
	EntryId     int       `json:"entryId"`
	VideoId     uuid.UUID `json:"videoId"`
	OwnerId     uuid.UUID `json:"ownerId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	WatchedAt   time.Time `json:"watchedAt"`
}

func (me *Handler) HandleGetWatchHistory(c fiber.Ctx) error {
	pr, err := parsePaginatedRequest[int](c)
	if err != nil {
		return err
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	entries, err := me.historyService.GetWatchHistory(
		c.RequestCtx(),
		currentUserId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items := make([]watchHistoryResponse, 0, len(entries))
	for _, e := range entries {
		video, err := me.videoService.GetVideoById(c.RequestCtx(), e.VideoId)
		if err != nil {
			if errors.Is(err, video_service.ErrVideoNotFound) {
				continue
			}
			return err
		}

		items = append(items, watchHistoryResponse{
			EntryId:     e.Id,
			VideoId:     e.VideoId,
			OwnerId:     video.OwnerId,
			Title:       video.Title,
			Description: video.Description,
			WatchedAt:   e.WatchedAt,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(entries[len(entries)-1].Id)
	}

	return c.JSON(response)
}

func (me *Handler) HandleAddToWatchHistory(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	lockKey := "video:" + videoId.String()
	if err := me.lock.RLock(c.RequestCtx(), lockKey); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), lockKey)

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	if err := me.historyService.AddToWatchHistory(c.RequestCtx(), currentUserId, videoId); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteWatchHistoryEntry(c fiber.Ctx) error {
	entryId := fiber.Params[int](c, "entry_id")
	if entryId == 0 {
		return errWatchHistoryEntryNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.historyService.DeleteWatchHistoryEntry(c.RequestCtx(), currentUserId, entryId); err != nil {
		if errors.Is(err, history_service.ErrWatchHistoryEntryNotFound) {
			return errWatchHistoryEntryNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleClearWatchHistory(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.historyService.ClearWatchHistory(c.RequestCtx(), currentUserId); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
