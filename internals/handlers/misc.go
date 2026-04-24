package handlers

import (
	"log/slog"
	"time"

	"github.com/assaidy/hyper/v2"
	"github.com/gofiber/fiber/v3"
)

func ErrorHandler(c fiber.Ctx, err error) error {
	err = fiber.DefaultErrorHandler(c, err)
	// Hide internal error from client; It has been logged by [WithLogging].
	if c.Response().StatusCode() == fiber.StatusInternalServerError {
		// TODO: return error toast component
		return fiber.ErrInternalServerError
	}
	return err
}

func WithLogging(c fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	took := time.Since(start)

	logger := fiber.MustGetState[*slog.Logger](c.App().State(), "logger")
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

func render(c fiber.Ctx, node hyper.HyperNode) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, node)
}
