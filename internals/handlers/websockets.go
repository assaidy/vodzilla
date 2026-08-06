package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type websocketMessageType string

const (
	websocketMessageNotification websocketMessageType = "notification"
)

type websocketMessage struct {
	Type websocketMessageType `json:"type"`
	Data any                  `json:"data"`
}

func (me *Handler) withWebsocketEssentials(c fiber.Ctx) error {
	if !c.IsWebSocket() {
		return errUpgradeRequired
	}
	return c.Next()
}

func (me *Handler) handleWebsocket(c *websocket.Conn) {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	sub := me.redis.Subscribe(context.Background(), fmt.Sprintf("ws:%s", currentUserId))
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
