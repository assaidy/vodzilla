package social

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/social/queries"
	"github.com/assaidy/workers"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var _ services.Service = (*Service)(nil)

type Service struct {
	db            *sql.DB
	queries       *queries.Queries
	redis         *redis.Client
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, redis *redis.Client, logger *slog.Logger) *Service {
	service := &Service{
		db:            db,
		queries:       queries.New(db),
		redis:         redis,
		logger:        logger,
		workerManager: workers.NewWorkerManager(workers.WithLogger(logger)),
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

func (me *Service) Start(ctx context.Context) error {
	me.workerManager.Start()
	return nil
}

func (me *Service) Stop(ctx context.Context) error {
	me.workerManager.Stop()
	return nil
}

func (me *Service) Follow(ctx context.Context, followerId, followedId uuid.UUID) error {
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

func (me *Service) Unfollow(ctx context.Context, followerId, followedId uuid.UUID) error {
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
	Followers int64
	Followeds int64
}

func (me *Service) GetFollowCounts(ctx context.Context, userId uuid.UUID) (*FollowCounts, error) {
	result, err := me.queries.GetFollowCounts(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get followers count: %w", err)
	}

	return &FollowCounts{
		Followers: result.FollowersCount,
		Followeds: result.FollowedsCount,
	}, nil
}

func (me *Service) IsFollower(ctx context.Context, followerId, followedId uuid.UUID) (bool, error) {
	ok, err := me.queries.CheckFollow(ctx, queries.CheckFollowParams{
		FollowerId: followerId,
		FollowedId: followedId,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check follow: %w", err)
	}

	return ok, err
}

func (me *Service) GetFollowerIds(ctx context.Context, userId, lastUserId uuid.UUID, limit int) ([]uuid.UUID, error) {
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

func (me *Service) GetFollowedIds(ctx context.Context, userId, lastUserId uuid.UUID, limit int) ([]uuid.UUID, error) {
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

func (me *Service) userDeletedEventConsumerJob(ctx context.Context) error {
	sub := me.redis.Subscribe(ctx, events.UserDeletedEvent)
	defer sub.Close()
	ch := sub.Channel()

	for {
		select {
		case message := <-ch:
			var payload events.UserDeletedEventPayload
			if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
				return fmt.Errorf("failed to unmarshal %q event payload: %w", events.UserDeletedEvent, err)
			}

			if err := me.queries.DeleteFollowsForUser(ctx, payload.UserId); err != nil {
				return fmt.Errorf("failed to delete follows for user: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
