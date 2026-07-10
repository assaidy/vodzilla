package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// notify persists a notification for userId via the notification service and
// pushes it in real-time to all of userId's connected websocket clients by
// publishing to the user's redis channel (so it works in a distributed
// environment where clients may be connected to other instances).
func (me *Handler) notify(ctx context.Context, userId uuid.UUID, payload notification_service.Payload) error {
	if err := me.notificationService.AddNotification(ctx, userId, payload); err != nil {
		return err
	}

	// TODO: create a websocket message type
	message, err := json.Marshal(fiber.Map{
		"kind":    payload.Kind(),
		"payload": payload,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification message: %w", err)
	}

	if err := me.redis.Publish(ctx, fmt.Sprintf("ws:%s", userId), message).Err(); err != nil {
		me.logger.Error("failed to publish notification to websocket channel", "error", err, "user_id", userId)
	}

	return nil
}

func (me *Handler) HandleGetNotifications(c fiber.Ctx) error {
	var request struct {
		LastNotificationId uuid.UUID `query:"last_notification_id"`
		Limit              int       `query:"limit"`
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

	notifications, err := me.notificationService.GetNotifications(
		c.RequestCtx(),
		currentUserId,
		request.LastNotificationId,
		request.Limit,
	)
	if err != nil {
		return err
	}

	type notificationResponse struct {
		Id        uuid.UUID       `json:"id"`
		Kind      string          `json:"kind"`
		Payload   json.RawMessage `json:"payload"`
		CreatedAt time.Time       `json:"createdAt"`
		IsRead    bool            `json:"isRead"`
	}

	response := make([]notificationResponse, 0, len(notifications))
	for _, n := range notifications {
		response = append(response, notificationResponse{
			Id:        n.Id,
			Kind:      string(n.Kind),
			Payload:   n.Payload,
			CreatedAt: n.CreatedAt,
			IsRead:    n.IsRead,
		})
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetUnreadNotificationsCount(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	count, err := me.notificationService.GetUnreadNotificationsCount(c.RequestCtx(), currentUserId)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"count": count})
}

func (me *Handler) HandleMarkNotificationAsRead(c fiber.Ctx) error {
	notificationId, err := uuid.Parse(c.Params("notification_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.notificationService.MarkNotificationAsRead(c.RequestCtx(), currentUserId, notificationId); err != nil {
		if errors.Is(err, notification_service.ErrNotificationNotFound) {
			return errNotificationNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
