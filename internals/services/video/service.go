package video

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/video/queries"
	"github.com/assaidy/workers"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
)

const Name = "video"

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
			fmt.Sprintf("%q event consumer", events.VideoUploadedEvent),
			service.videoUploadedEventConsumer,
			workers.WithNRuns(1),
			workers.WithNRetries(0),
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

type VideoStatus string

const (
	// do not put this into db; it's for application to know if video is ready or not
	// this status might change if we added or removed layers of processing to video
	VideoStatusReady = VideoStatusUploaded

	VideoStatusUploading VideoStatus = "uploading"
	VideoStatusUploaded  VideoStatus = "uploaded"
)

type CreateVideoParams struct {
	OwnerId     string
	Title       string
	Description string
}

type CreateVideoReuslt struct {
	VideoId   string
	ObjectKey string
}

func (me *Service) CreateVideo(ctx context.Context, params CreateVideoParams) (*CreateVideoReuslt, error) {
	videoId := ulid.Make().String()
	objectKey := fmt.Sprintf("%s/%s", params.OwnerId, videoId)

	if err := me.queries.InsertVideo(ctx, queries.InsertVideoParams{
		Id:          videoId,
		ObjectKey:   objectKey,
		OwnerId:     params.OwnerId,
		Title:       params.Title,
		Description: sql.NullString{Valid: params.Description != "", String: params.Description},
		Status:      string(VideoStatusUploading),
	}); err != nil {
		return nil, fmt.Errorf("failed to insert video: %w", err)
	}

	return &CreateVideoReuslt{
		VideoId:   videoId,
		ObjectKey: objectKey,
	}, nil
}

func (me *Service) videoUploadedEventConsumer(ctx context.Context) error {
	sub := me.redis.Subscribe(ctx, events.VideoUploadedEvent)
	defer sub.Close()
	ch := sub.Channel()

	for {
		select {
		case message := <-ch:
			var payload events.VideoUploadedEventPayload
			if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
				me.logger.Error("failed to unmarshal event payload", "event", events.VideoUploadedEvent, "error", err)
				continue
			}

			video, err := me.queries.GetVideoByObjectKey(ctx, payload.ObjectKey)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					me.logger.Error("failed to get video by object key", "error", err)
				}
				// else, video was removed for expiration or something
				continue
			}

			if err := me.queries.UpdateVideoStatus(ctx, queries.UpdateVideoStatusParams{
				Id:     video.Id,
				Status: string(VideoStatusUploaded),
			}); err != nil {
				me.logger.Error("failed to update video status", "error", err)
				continue
			}

		case <-ctx.Done():
			return nil
		}
	}
}

type Video struct {
	Id          string
	OwnerId     string
	ObjectKey   string
	Timestamp   time.Time
	Title       string
	Description string
}

func (me *Service) GetVideoById(ctx context.Context, id string) (*Video, error) {
	video, err := me.queries.GetVideoById(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get video by id: %w", err)
	}

	return &Video{
		Id:          video.Id,
		OwnerId:     video.OwnerId,
		ObjectKey:   video.ObjectKey,
		Timestamp:   video.CreatedAt,
		Title:       video.Title,
		Description: video.Description.String,
	}, nil
}
