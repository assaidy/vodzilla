package social

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/social/queries"
	"github.com/assaidy/workers"
	"github.com/assaidy/workers/lock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	services.Service

	// Follow makes the given follower user follow the given followed user.
	//
	// Errors:
	//   - [ErrSelfFollowNotAllowed] - the follower and followed ids are the same
	//   - [ErrAlreadyFollowing] - the follower already follows the followed user
	Follow(ctx context.Context, followerId, followedId uuid.UUID) error

	// Unfollow makes the given follower user unfollow the given followed user.
	//
	// Errors:
	//   - [ErrNotFollowing] - the follower does not follow the followed user
	Unfollow(ctx context.Context, followerId, followedId uuid.UUID) error

	// GetFollowCounts returns the followers and followeds counts of the given user.
	GetFollowCounts(ctx context.Context, userId uuid.UUID) (*FollowCounts, error)

	// IsFollower reports whether the given follower user follows the given followed user.
	IsFollower(ctx context.Context, followerId, followedId uuid.UUID) (bool, error)

	// GetFollowerIds returns the ids of the followers of the given user,
	// paginated by the given last user id and limit.
	GetFollowerIds(ctx context.Context, userId, lastUserId uuid.UUID, limit int) ([]uuid.UUID, error)

	// GetFollowedIds returns the ids of the users followed by the given user,
	// paginated by the given last user id and limit.
	GetFollowedIds(ctx context.Context, userId, lastUserId uuid.UUID, limit int) ([]uuid.UUID, error)

	// GetAllFollowedIds returns the ids of all users followed by the given user.
	GetAllFollowedIds(ctx context.Context, userId uuid.UUID) ([]uuid.UUID, error)

	// GetAllFollowerIds returns the ids of all followers of the given user.
	GetAllFollowerIds(ctx context.Context, userId uuid.UUID) ([]uuid.UUID, error)
}

type impl struct {
	db            *sql.DB
	queries       *queries.Queries
	redis         *redis.Client
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, redis *redis.Client, logger *slog.Logger) Service {
	service := &impl{
		db:      db,
		queries: queries.New(db),
		redis:   redis,
		logger:  logger,
		workerManager: workers.NewWorkerManager(
			workers.WithLogger(logger),
			workers.WithLockGenerator(lock.NewRedisGenerator(redis)),
		),
	}

	service.workerManager.RegisterWorker(
		workers.NewWorker(
			fmt.Sprintf("%q event consumer", events.UserDeletedEvent),
			service.userDeletedEventConsumerJob,
			workers.WithRetryDelay(time.Second),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
		),
	)

	return service
}

func (me *impl) Start(ctx context.Context) error {
	me.workerManager.Start()
	return nil
}

func (me *impl) Stop(ctx context.Context) error {
	me.workerManager.Stop()
	return nil
}

func (me *impl) Follow(ctx context.Context, followerId, followedId uuid.UUID) error {
	if followerId == followedId {
		return ErrSelfFollowNotAllowed
	}

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckFollow(ctx, queries.CheckFollowParams{
		FollowerId: followerId,
		FollowedId: followedId,
	}); err != nil {
		return fmt.Errorf("failed to check follow: %w", err)
	} else if ok {
		return ErrAlreadyFollowing
	}

	if err := qtx.InsertFollow(ctx, queries.InsertFollowParams{
		FollowerId: followerId,
		FollowedId: followedId,
	}); err != nil {
		return fmt.Errorf("failed to insert follow: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *impl) Unfollow(ctx context.Context, followerId, followedId uuid.UUID) error {
	if n, err := me.queries.DeleteFollow(ctx, queries.DeleteFollowParams{
		FollowerId: followerId,
		FollowedId: followedId,
	}); err != nil {
		return fmt.Errorf("failed to delete follow: %w", err)
	} else if n == 0 {
		return ErrNotFollowing
	}

	return nil
}

type FollowCounts struct {
	Followers int
	Followeds int
}

func (me *impl) GetFollowCounts(ctx context.Context, userId uuid.UUID) (*FollowCounts, error) {
	result, err := me.queries.GetFollowCounts(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get followers count: %w", err)
	}

	return &FollowCounts{
		Followers: int(result.FollowersCount),
		Followeds: int(result.FollowedsCount),
	}, nil
}

func (me *impl) IsFollower(ctx context.Context, followerId, followedId uuid.UUID) (bool, error) {
	ok, err := me.queries.CheckFollow(ctx, queries.CheckFollowParams{
		FollowerId: followerId,
		FollowedId: followedId,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check follow: %w", err)
	}

	return ok, nil
}

func (me *impl) GetFollowerIds(ctx context.Context, userId, lastUserId uuid.UUID, limit int) ([]uuid.UUID, error) {
	ids, err := me.queries.GetFollowerIds(ctx, queries.GetFollowerIdsParams{
		UserId:     userId,
		LastUserId: uuid.NullUUID{UUID: lastUserId, Valid: lastUserId != uuid.Nil},
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get follower ids for user: %w", err)
	}
	return ids, nil
}

func (me *impl) GetFollowedIds(ctx context.Context, userId, lastUserId uuid.UUID, limit int) ([]uuid.UUID, error) {
	ids, err := me.queries.GetFollowedIds(ctx, queries.GetFollowedIdsParams{
		UserId:     userId,
		LastUserId: uuid.NullUUID{UUID: lastUserId, Valid: lastUserId != uuid.Nil},
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get followed ids for user: %w", err)
	}
	return ids, nil
}

func (me *impl) GetAllFollowedIds(ctx context.Context, userId uuid.UUID) ([]uuid.UUID, error) {
	ids, err := me.queries.GetAllFollowedIds(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get all followed ids for user: %w", err)
	}
	return ids, nil
}

func (me *impl) GetAllFollowerIds(ctx context.Context, userId uuid.UUID) ([]uuid.UUID, error) {
	ids, err := me.queries.GetAllFollowerIds(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get all follower ids for user: %w", err)
	}
	return ids, nil
}

func (me *impl) userDeletedEventConsumerJob(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var payload events.UserDeletedEventPayload
			if ok, err := events.Dequeue(ctx, me.redis, events.UserDeletedEvent, &payload); err != nil {
				return fmt.Errorf("failed to dequeue %q event: %w", events.UserDeletedEvent, err)
			} else if !ok {
				continue
			}

			if err := me.queries.DeleteFollowsForUser(ctx, payload.UserId); err != nil {
				return fmt.Errorf("failed to delete follows for user: %w", err)
			}
		}
	}
}
