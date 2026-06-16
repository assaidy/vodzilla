package handlers

import (
	"errors"
	"time"

	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func (me *Handler) WithLogging(c fiber.Ctx) error {
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
	// All errors are handled and the request (body & status code)  is finalizedin [WithErrorResolver].
	return nil
}

func (me *Handler) WithErrorResolver(c fiber.Ctx) error {
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
		me.logger.Error("failed to write error response", "error", writeErr)
	}

	return err
}
