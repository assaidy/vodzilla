package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// notify persists a notification for userId via the notification service and
// pushes it in real-time to all of userId's connected websocket clients by
// publishing to the user's redis channel (so it works in a distributed
// environment where clients may be connected to other instances).
func (me *Handler) notify(ctx context.Context, userId uuid.UUID, payload notification_service.Payload) error {
	if err := me.lock.RLock(ctx, "user:"+userId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(ctx, "user:"+userId.String())

	if ok, err := me.userService.DoesUserExist(ctx, userId); err != nil {
		return err
	} else if !ok {
		return nil
	}

	if err := me.notificationService.AddNotification(ctx, userId, payload); err != nil {
		return err
	}

	message, err := json.Marshal(websocketMessage{
		Type: websocketMessageNotification,
		Data: fiber.Map{
			"kind":    payload.Kind(),
			"payload": payload,
		},
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
	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	notifications, err := me.notificationService.GetNotifications(
		c.RequestCtx(),
		currentUserId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items := make([]fiber.Map, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, fiber.Map{
			"id":        n.Id,
			"kind":      n.Kind,
			"payload":   n.Payload,
			"createdAt": n.CreatedAt,
			"isRead":    n.IsRead,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(notifications[len(notifications)-1].Id)
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
		return errNotificationNotFound
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
