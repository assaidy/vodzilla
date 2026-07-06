package handlers

import (
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/services/media"
	"github.com/assaidy/vodzilla/internals/services/reaction"
	"github.com/assaidy/vodzilla/internals/services/social"
	"github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils/keyed_mutex"
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
	sessionMutex    *keyed_mutex.RWMutex
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
		sessionMutex:    keyed_mutex.New(),
		userMutex:       keyed_mutex.New(),
		videoMutex:      keyed_mutex.New(),
	}

	// TODO: GOROUTINE LEAK! implement a stop/cancel mechanism later.
	// Consider adding start() and stop() methods for the handler.
	// They will start/stop necessary jobs and queues.
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			handler.sessionMutex.ClearUnused(1 * time.Hour)
			handler.userMutex.ClearUnused(1 * time.Hour)
			handler.videoMutex.ClearUnused(1 * time.Hour)
		}
	}()

	return handler
}
