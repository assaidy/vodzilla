package handlers

import (
	"log/slog"

	"github.com/assaidy/vodzilla/internals/services/media"
	"github.com/assaidy/vodzilla/internals/services/reaction"
	"github.com/assaidy/vodzilla/internals/services/social"
	"github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils"
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
	lock            *utils.DistributedLock
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
	return &Handler{
		logger:          logger,
		redis:           redis,
		userService:     userService,
		videoService:    videoService,
		mediaService:    mediaService,
		reactionService: reactionService,
		socialService:   socialService,
		lock:            utils.NewDistributedLock(redis, logger),
	}
}
