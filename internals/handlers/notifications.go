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

func (me *Handler) notify(ctx context.Context, userId uuid.UUID, payload notification_service.Payload) {
	lock := me.newUserLock(userId)
	if err := lock.SpinRLock(ctx, spinLockTimeout); err != nil {
		me.logger.Error("failed to acquire read lock", "error", err, "user_id", userId)
		return
	}
	defer lock.RUnLock(ctx)

	if ok, err := me.userService.DoesUserExist(ctx, userId); err != nil {
		me.logger.Error("failed to check user existence", "error", err, "user_id", userId)
		return
	} else if !ok {
		return
	}

	if err := me.notificationService.AddNotification(ctx, userId, payload); err != nil {
		me.logger.Error("failed to add notification", "error", err, "user_id", userId)
		return
	}

	message, err := json.Marshal(websocketMessage{
		Type: websocketMessageNotification,
		Data: fiber.Map{
			"kind":    payload.Kind(),
			"payload": payload,
		},
	})
	if err != nil {
		me.logger.Error("failed to marshal notification message", "error", err)
		return
	}

	if err := me.redis.Publish(ctx, fmt.Sprintf("ws:%s", userId), message).Err(); err != nil {
		me.logger.Error("failed to publish notification to websocket channel", "error", err, "user_id", userId)
	}
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
