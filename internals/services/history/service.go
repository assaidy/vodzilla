package history

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/history/queries"
	"github.com/assaidy/workers"
	"github.com/assaidy/workers/lock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	services.Service

	// AddToWatchHistory records a new watch history entry for the given user and video.
	AddToWatchHistory(ctx context.Context, userId, videoId uuid.UUID) error

	// GetWatchHistory returns the watch history entries of the given user,
	// paginated by the given last entry id and limit.
	GetWatchHistory(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]WatchHistoryEntry, error)

	// DeleteWatchHistoryEntry deletes the watch history entry with the given id owned by the given user.
	//
	// Errors:
	//   - [ErrWatchHistoryEntryNotFound] - no entry exists for the given user with the given id
	DeleteWatchHistoryEntry(ctx context.Context, userId uuid.UUID, entryId int) error

	// ClearWatchHistory deletes all watch history entries of the given user.
	ClearWatchHistory(ctx context.Context, userId uuid.UUID) error
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
	service.workerManager.RegisterWorker(
		workers.NewWorker(
			fmt.Sprintf("%q event consumer", events.VideoDeletedEvent),
			service.videoDeletedEventConsumerJob,
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

type WatchHistoryEntry struct {
	Id        int
	UserId    uuid.UUID
	VideoId   uuid.UUID
	WatchedAt time.Time
}

func (me *impl) AddToWatchHistory(ctx context.Context, userId, videoId uuid.UUID) error {
	if err := me.queries.InsertWatchHistory(ctx, queries.InsertWatchHistoryParams{
		UserId:  userId,
		VideoId: videoId,
	}); err != nil {
		return fmt.Errorf("failed to insert watch history entry: %w", err)
	}

	return nil
}

func (me *impl) GetWatchHistory(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]WatchHistoryEntry, error) {
	rows, err := me.queries.GetWatchHistory(ctx, queries.GetWatchHistoryParams{
		UserId: userId,
		LastId: sql.NullInt64{Int64: int64(lastId), Valid: lastId != 0},
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get watch history: %w", err)
	}

	result := make([]WatchHistoryEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, WatchHistoryEntry{
			Id:        int(row.Id),
			UserId:    row.UserId,
			VideoId:   row.VideoId,
			WatchedAt: row.WatchedAt,
		})
	}

	return result, nil
}

func (me *impl) DeleteWatchHistoryEntry(ctx context.Context, userId uuid.UUID, entryId int) error {
	if n, err := me.queries.DeleteWatchHistoryEntry(ctx, queries.DeleteWatchHistoryEntryParams{
		Id:     int64(entryId),
		UserId: userId,
	}); err != nil {
		return fmt.Errorf("failed to delete watch history entry: %w", err)
	} else if n == 0 {
		return ErrWatchHistoryEntryNotFound
	}

	return nil
}

func (me *impl) ClearWatchHistory(ctx context.Context, userId uuid.UUID) error {
	if err := me.queries.DeleteAllWatchHistoryForUser(ctx, userId); err != nil {
		return fmt.Errorf("failed to clear watch history: %w", err)
	}

	return nil
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

			if err := me.queries.DeleteAllWatchHistoryForUser(ctx, payload.UserId); err != nil {
				return fmt.Errorf("failed to delete all watch history for user: %w", err)
			}
		}
	}
}

func (me *impl) videoDeletedEventConsumerJob(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var payload events.VideoDeletedEventPayload
			if ok, err := events.Dequeue(ctx, me.redis, events.VideoDeletedEvent, &payload); err != nil {
				return fmt.Errorf("failed to dequeue %q event: %w", events.VideoDeletedEvent, err)
			} else if !ok {
				continue
			}

			if err := me.queries.DeleteAllWatchHistoryForVideo(ctx, payload.VideoId); err != nil {
				return fmt.Errorf("failed to delete all watch history for video: %w", err)
			}
		}
	}
}
