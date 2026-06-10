package handlers

import (
	"errors"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func WithLogging(c fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	took := time.Since(start)

	fiber.MustGetState[*slog.Logger](c.App().State(), "logger").
		Info("request handled",
			"took", took,
			"ip", c.IP(),
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"error", err,
		)

	// Handler error is intentionally not returned.
	// All errors are handled and the request is finalized in [WithErrorResolver].
	return nil
}

func WithErrorResolver(c fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	code := fiber.StatusInternalServerError
	if e, ok := errors.AsType[*fiber.Error](err); ok {
		code = e.Code
	}
	c.Status(code)

	var writeErr error
	if code == fiber.StatusInternalServerError {
		// Hide internal error from client; it will be caught by [WithLogging].
		writeErr = render(c, templates.Alert(templates.AlertError, "We had a server error. Please try again later."))
	} else {
		writeErr = c.SendString(err.Error())
	}

	// Log write error without passing it to logger middleware;
	// we only want [WithLogging] to log the handler error.
	if writeErr != nil {
		fiber.MustGetState[*slog.Logger](c.App().State(), "logger").
			Error("failed to write error response", "error", writeErr)
	}

	return err
}
