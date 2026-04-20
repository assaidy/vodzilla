package handlers

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v3"
)

func WithLogging(c fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	took := time.Since(start)

	logger := fiber.MustGetState[*log.Logger](c.App().State(), "logger")
	logger.Info("request handled",
		"took", took,
		"ip", c.IP(),
		"method", c.Method(),
		"path", c.Path(),
		"status", c.Response().StatusCode(),
		"error", err,
	)

	return err
}

func ErrorHandler(c fiber.Ctx, err error) error {
	err = fiber.DefaultErrorHandler(c, err)
	// Hide internal error from client; It has been logged by [WithLogging].
	if c.Response().StatusCode() == fiber.StatusInternalServerError {
		return fiber.ErrInternalServerError
	}
	return err
}
