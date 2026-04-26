package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/video_streaming_app/internals/web/templates"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/starfederation/datastar-go/datastar"
)

func ErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := errors.AsType[*fiber.Error](err); ok {
		code = e.Code
	}

	// Hide internal error from client; It has been logged by [WithLogging].
	if code == fiber.StatusInternalServerError {
		if err := render(c, templates.Alert(templates.AlertError, "We had a server error. Please try again later.")); err != nil {
			fiber.MustGetState[*slog.Logger](c.App().State(), "logger").
				Error("couldn't render alert for internal server error", "error", err)
			return fiber.ErrInternalServerError
		}
		return nil
	}

	return c.Status(code).SendString(err.Error())
}

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

	return err
}

func render(c fiber.Ctx, node hyper.HyperNode) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, node)
}

func newDatastarSse(c fiber.Ctx) (*datastar.ServerSentEventGenerator, error) {
	httpRequest, err := adaptor.ConvertRequest(c, false)
	if err != nil {
		panic("wtf are you doing?!")
	}

	// FIX: no escape! time to ditch fiber
	return datastar.NewSSE(/*imposible to get http.ResponseWriter from fiber.ctx*/, httpRequest), nil
}
