package history

import (
	"context"
	"database/sql"
	"encoding/json"
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
	AddToWatchHistory(ctx context.Context, userId, videoId uuid.UUID) error
	GetWatchHistory(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]WatchHistoryEntry, error)
	DeleteWatchHistoryEntry(ctx context.Context, userId uuid.UUID, entryId int) error
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

			if err := me.queries.DeleteAllWatchHistoryForUser(ctx, payload.UserId); err != nil {
				return fmt.Errorf("failed to delete all watch history for user: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (me *impl) videoDeletedEventConsumerJob(ctx context.Context) error {
	sub := me.redis.Subscribe(ctx, events.VideoDeletedEvent)
	defer sub.Close()
	ch := sub.Channel()

	for {
		select {
		case message := <-ch:
			var payload events.VideoDeletedEventPayload
			if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
				return fmt.Errorf("failed to unmarshal %q event payload: %w", events.VideoDeletedEvent, err)
			}

			if err := me.queries.DeleteAllWatchHistoryForVideo(ctx, payload.VideoId); err != nil {
				return fmt.Errorf("failed to delete all watch history for video: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
