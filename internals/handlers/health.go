package handlers

import "github.com/gofiber/fiber/v3"

func (me *Handler) HandleCheckHealth(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}
