package handlers

import (
	"log/slog"

	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/vodzilla/internals/services/media"
	"github.com/assaidy/vodzilla/internals/services/reaction"
	"github.com/assaidy/vodzilla/internals/services/social"
	"github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/services/video"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	logger          *slog.Logger
	redis           *redis.Client
	userService     *user.Service
	videoService    *video.Service
	mediaService    *media.Service
	reactionService *reaction.Service
	socialService   *social.Service
}

func New(
	logger *slog.Logger,
	redis *redis.Client,
	userService *user.Service,
	videoService *video.Service,
	mediaService *media.Service,
	reactionService *reaction.Service,
	socialService *social.Service,
) *Handler {
	handler := &Handler{
		logger:          logger,
		redis:           redis,
		userService:     userService,
		videoService:    videoService,
		mediaService:    mediaService,
		reactionService: reactionService,
		socialService:   socialService,
	}

	return handler
}

func render(c fiber.Ctx, node hyper.HyperNode) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, node)
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
