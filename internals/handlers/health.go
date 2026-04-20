package handlers

import (
	"github.com/gofiber/fiber/v3"
)

func HandleCheckHealth(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}
