package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

func (me *Handler) WithPassClientIdToLocals(c fiber.Ctx) error {
	if clientId := c.Get("X-Client-ID"); clientId != "" {
		c.Locals("client_id", clientId)
	} else {
		me.logger.Warn("request with no client id", "method", c.Method(), "path", c.Path())
	}
	return c.Next()
}

func (me *Handler) WithWebsocketEssentials(c fiber.Ctx) error {
	if !c.IsWebSocket() {
		return errUpgradeRequired
	}

	c.Locals("fiber_app", c.App())
	return c.Next()
}

func (me *Handler) HandleWebsocket(c *websocket.Conn) {
	clientId := c.Params("client_id")

	sub := me.redis.Subscribe(context.Background(), fmt.Sprintf("ws:%s", clientId))
	defer sub.Close()
	subChan := sub.Channel()

	doneChan := make(chan struct{})
	defer close(doneChan)

	wsReadChan := make(chan []byte, 100)
	wsClosedChan := make(chan struct{})

	go func() {
		for {
			close(wsClosedChan)
			close(wsReadChan)

			_, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err) {
					return
				} else {
					me.logger.Error("failed to read websocket message", "error", err)
					continue
				}
			}

			select {
			case <-doneChan:
				return
			case wsReadChan <- msg:
			}
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
				me.logger.Error("failed to write websocket message", "error", err)
			}
		}
	}
}
