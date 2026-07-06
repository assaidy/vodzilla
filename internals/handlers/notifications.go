package handlers

import "github.com/gofiber/fiber/v3"

// TODO: implement mark notifications as read
func (me *Handler) HandleMarkNotificationAsRead(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// TODO: implement get unread notifications count
func (me *Handler) HandleGetUnreadNotoficationsCount(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// TODO: implement get notifications
func (me *Handler) HandleGetNotofications(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}
