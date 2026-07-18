package handlers

import (
	"log/slog"
	"time"

	history_service "github.com/assaidy/vodzilla/internals/services/history"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/google/uuid"
	redislock "github.com/jefferyjob/go-redislock"
	redislock_adapter "github.com/jefferyjob/go-redislock/adapter/go-redis/V9"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	logger              *slog.Logger
	redis               *redis.Client
	redisLockInterface  redislock.RedisInter
	userService         user_service.Service
	videoService        video_service.Service
	mediaService        media_service.Service
	reactionService     reaction_service.Service
	socialService       social_service.Service
	notificationService notification_service.Service
	historyService      history_service.Service
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
	historyService history_service.Service,
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
		historyService:      historyService,
		redisLockInterface:  redislock_adapter.New(redis),
	}
}

const spinLockTimeout = 5 * time.Second

func (me *Handler) newSessionLock(sessionId uuid.UUID) redislock.RedisLockInter {
	return redislock.New(me.redisLockInterface, "session:"+sessionId.String(), redislock.WithAutoRenew())
}

func (me *Handler) newUserLock(userId uuid.UUID) redislock.RedisLockInter {
	return redislock.New(me.redisLockInterface, "user:"+userId.String(), redislock.WithAutoRenew())
}

func (me *Handler) newVideoLock(videoId uuid.UUID) redislock.RedisLockInter {
	return redislock.New(me.redisLockInterface, "video:"+videoId.String(), redislock.WithAutoRenew())
}

func (me *Handler) newPlaylistLock(playlistId uuid.UUID) redislock.RedisLockInter {
	return redislock.New(me.redisLockInterface, "playlist:"+playlistId.String(), redislock.WithAutoRenew())
}

func (me *Handler) newCommentLock(commentId uuid.UUID) redislock.RedisLockInter {
	return redislock.New(me.redisLockInterface, "comment:"+commentId.String(), redislock.WithAutoRenew())
}
