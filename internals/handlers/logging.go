package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

func (me *Handler) withLogging(c fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	took := time.Since(start)

	me.logger.Info("request handled",
		"took", took,
		"ip", c.IP(),
		"method", c.Method(),
		"path", c.Path(),
		"status", c.Response().StatusCode(),
		"error", err,
	)

	// Handler error is intentionally not returned.
	// All errors are handled and the request (body & status code) is finalizedin [Handler.WithErrorResolver].
	return nil
}
