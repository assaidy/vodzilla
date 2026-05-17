package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
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

	// I don't return handlers chain error intentionally.
	// All errors were handled and request was finalized in [WithErrorResolver].
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

	if code == fiber.StatusInternalServerError {
		// Hide internal error from client; It has been catched by [WithLogging].
		return render(c, templates.Alert(templates.AlertError, "We had a server error. Please try again later."))
	}

	return c.SendString(err.Error())
}

func HandleCheckHealth(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

func WithPassClientIdToLocals(c fiber.Ctx) error {
	if clientId := c.Get("X-Client-ID"); clientId != "" {
		c.Locals("client_id", clientId)
	} else {
		fiber.MustGetState[*slog.Logger](c.App().State(), "logger").
			Warn("request with not client id", "method", c.Method(), "path", c.Path())
	}
	return c.Next()
}

func WithWebsocketEssentials(c fiber.Ctx) error {
	if !c.IsWebSocket() {
		return fiber.ErrUpgradeRequired
	}

	c.Locals("fiber_app", c.App())
	return c.Next()
}

func HandleWebsocket(c *websocket.Conn) {
	clientId := c.Params("client_id")
	app := c.Locals("fiber_app").(*fiber.App)

	sub := fiber.MustGetState[*redis.Client](app.State(), "redis").
		Subscribe(context.Background(), fmt.Sprintf("ws:%s", clientId))
	defer sub.Close()
	subChan := sub.Channel()

	wsReadChan := make(chan []byte, 100)
	defer close(wsReadChan)
	wsClosedChan := make(chan struct{})

	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err) {
					close(wsClosedChan)
					return
				} else {
					fiber.MustGetState[*slog.Logger](app.State(), "logger").
						Error("failed to read websocket message", "error", err)
					continue
				}
			}
			wsReadChan <- msg
		}
	}()

	for {
		select {
		case <-wsClosedChan:
			return
		case msg := <-wsReadChan:
			_ = msg
		case msg, ok := <-subChan:
			if !ok {
				return
			}
			if err := c.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				fiber.MustGetState[*slog.Logger](app.State(), "logger").
					Error("failed to write websocket message", "error", err)
			}
		}
	}
}

func render(c fiber.Ctx, node hyper.HyperNode) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, node)
}

func renderToWebsocket(c fiber.Ctx, node hyper.HyperNode) error {
	clientId, ok := c.Locals("client_id").(string)
	if !ok {
		return fmt.Errorf("couldn't find client id in locals")
	}

	return hyper.RenderThen(node, func(data []byte) error {
		return fiber.MustGetState[*redis.Client](c.App().State(), "redis").
			Publish(c, fmt.Sprintf("ws:%s", clientId), data).Err()
	})
}

func redirect(c fiber.Ctx, endpoint string) error {
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", endpoint)
	} else if c.Get("HX-Boosted") == "true" {
		c.Set("HX-Location", endpoint)
	} else {
		return c.Redirect().To(endpoint)
	}
	return nil
}
