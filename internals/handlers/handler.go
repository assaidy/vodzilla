package handlers

import (
	"log/slog"
	"sync"

	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/vodzilla/internals/services/media"
	"github.com/assaidy/vodzilla/internals/services/reaction"
	"github.com/assaidy/vodzilla/internals/services/social"
	"github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/services/video"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
	// TODO: I need to delete mutexes from the map if they're no longer aquired.
	userMutexes  sync.Map
	videoMutexes sync.Map
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

// TODO: think more about these locks.
// why locking all other handlers that are similar to me if there's no
// deletion route is aquiring?
func (me *Handler) userLock(userId uuid.UUID) {
	mu, _ := me.userMutexes.LoadOrStore(userId, new(sync.Mutex))
	mu.(*sync.Mutex).Lock()
}

func (me *Handler) userUnlock(userId uuid.UUID) {
	if mu, ok := me.userMutexes.Load(userId); ok {
		mu.(*sync.Mutex).Unlock()
	}
}

func (me *Handler) videoLock(videoId uuid.UUID) {
	mu, _ := me.videoMutexes.LoadOrStore(videoId, new(sync.Mutex))
	mu.(*sync.Mutex).Lock()
}

func (me *Handler) videoUnlock(videoId uuid.UUID) {
	if mu, ok := me.videoMutexes.Load(videoId); ok {
		mu.(*sync.Mutex).Unlock()
	}
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
