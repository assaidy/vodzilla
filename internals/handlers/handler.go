package handlers

import (
	"log/slog"

	media_service "github.com/assaidy/vodzilla/internals/services/media"
	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils/distributed_lock"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	logger              *slog.Logger
	redis               *redis.Client
	userService         user_service.Service
	videoService        video_service.Service
	mediaService        media_service.Service
	reactionService     reaction_service.Service
	socialService       social_service.Service
	notificationService notification_service.Service
	lock                *distributed_lock.DistributedLock
}

func New(
	logger *slog.Logger,
	redis *redis.Client,
	userService user_service.Service,
	videoService video_service.Service,
	mediaService media_service.Service,
	reactionService reaction_service.Service,
	socialService social_service.Service,
	notificationService notification_service.Service,
) *Handler {
	return &Handler{
		logger:              logger,
		redis:               redis,
		userService:         userService,
		videoService:        videoService,
		mediaService:        mediaService,
		reactionService:     reactionService,
		socialService:       socialService,
		notificationService: notificationService,
		lock:                distributed_lock.New(redis, logger),
	}
}
