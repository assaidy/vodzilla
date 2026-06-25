package handlers

import (
	"log/slog"
	"time"

	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/vodzilla/internals/services/media"
	"github.com/assaidy/vodzilla/internals/services/reaction"
	"github.com/assaidy/vodzilla/internals/services/social"
	"github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils/keyed_mutex"
	"github.com/assaidy/vodzilla/internals/web/templates"
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
	userMutex       *keyed_mutex.RWMutex
	videoMutex      *keyed_mutex.RWMutex
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
		userMutex:       keyed_mutex.New(),
		videoMutex:      keyed_mutex.New(),
	}

	// TODO: GOROUTINE LEAK! impl a stop/cancel mechanism later.
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			handler.userMutex.ClearUnused(1 * time.Hour)
			handler.videoMutex.ClearUnused(1 * time.Hour)
		}
	}()

	return handler
}

func render(c fiber.Ctx, node hyper.HyperNode) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, node)
}

func redirect(c fiber.Ctx, endpoint string) error {
	if c.Get(templates.HeaderHxRequest) == "true" {
		c.Set(templates.HeaderHxRedirect, endpoint)
	} else if c.Get(templates.HeaderHxBoosted) == "true" {
		c.Set(templates.HeaderHxLocation, endpoint)
	} else {
		return c.Redirect().To(endpoint)
	}
	return nil
}
