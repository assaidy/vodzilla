package media

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"slices"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/media/queries"
	"github.com/assaidy/workers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/redis/go-redis/v9"
)

const Name = "media"

var _ services.Service = (*Service)(nil)

type Service struct {
	db            *sql.DB
	queries       *queries.Queries
	s3            *s3.Client
	redis         *redis.Client
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, s3 *s3.Client, redis *redis.Client, logger *slog.Logger) *Service {
	service := &Service{
		db:            db,
		queries:       queries.New(db),
		s3:            s3,
		redis:         redis,
		logger:        logger,
		workerManager: workers.NewWorkerManager(workers.WithLogger(logger)),
	}

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
	VideoStatusUploading VideoStatus = "uploading"
	VideoStatusUploaded  VideoStatus = "uploaded"
)

const (
	videosBucket              = "videos"
	maxPartCount              = 10_000
	minPartSize               = 5 << 20 // 5 MB
	presignedUploadExpiration = 24 * time.Hour
)

type PresignedUpload struct {
	UploadId string
	PartSize int64
	Urls     []string
}

func (me *Service) GeneratePresignedPutUrls(ctx context.Context, videoId uuid.UUID, objectKey, contentType string, fileSize int64) (*PresignedUpload, error) {
	createResult, err := me.s3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(videosBucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}
	uploadId := aws.ToString(createResult.UploadId)

	requiredPartSize := (fileSize + maxPartCount - 1) / maxPartCount
	partSize := max(minPartSize, requiredPartSize)
	partCount := (fileSize + partSize - 1) / partSize
	urls := make([]string, 0, partCount)

	presigner := s3.NewPresignClient(me.s3)
	for partNumber := 1; partNumber <= int(partCount); partNumber++ {
		request, err := presigner.PresignUploadPart(
			ctx,
			&s3.UploadPartInput{
				Bucket:     aws.String(videosBucket),
				Key:        aws.String(objectKey),
				UploadId:   aws.String(uploadId),
				PartNumber: aws.Int32(int32(partNumber)),
			},
			func(opts *s3.PresignOptions) {
				opts.Expires = 24 * time.Hour
			},
		)
		if err != nil {
			me.abortUploadWithErrorLogging(ctx, objectKey, uploadId)
			return nil, fmt.Errorf("failed to presign url: %w", err)
		}

		urls = append(urls, request.URL)
	}

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if err := qtx.InsertVideo(ctx, queries.InsertVideoParams{
		Id:        videoId,
		ObjectKey: objectKey,
		Status:    string(VideoStatusUploading),
	}); err != nil {
		me.abortUploadWithErrorLogging(ctx, objectKey, uploadId)
		return nil, fmt.Errorf("failed to insert video: %w", err)
	}

	if err := qtx.InsertUpload(ctx, queries.InsertUploadParams{
		Id:        uploadId,
		VideoId:   videoId,
		ExpiresAt: time.Now().Add(presignedUploadExpiration),
	}); err != nil {
		me.abortUploadWithErrorLogging(ctx, objectKey, uploadId)
		return nil, fmt.Errorf("failed to insert upload: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return &PresignedUpload{
		UploadId: uploadId,
		Urls:     urls,
		PartSize: partSize,
	}, nil
}

func (me *Service) abortUploadWithErrorLogging(ctx context.Context, objectKey, uploadId string) {
	if _, err := me.s3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(videosBucket),
		Key:      aws.String(objectKey),
		UploadId: aws.String(uploadId),
	}); err != nil {
		me.logger.Error("failed to abort multipart upload",
			"object_key", objectKey, "upload_id", uploadId, "error", err)
	}
}

type CompleteUploadPart struct {
	ETag       string
	PartNumber int
}

func (me *Service) CompleteUpload(ctx context.Context, videoId uuid.UUID, uploadId string, parts []CompleteUploadPart) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	objectKey, err := qtx.GetObjectKeyForVideo(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("failed to get object key: %w", err)
	}

	upload, err := qtx.GetUploadForVideo(ctx, videoId)
	if err != nil {
		return fmt.Errorf("failed to check unexpired upload for video: %w", err)
	}

	if upload.CompletedAt.Valid {
		return ErrUploadAlreadyCompleted
	}
	if !upload.ExpiresAt.After(time.Now()) {
		return ErrUploadExpired
	}

	completedParts := make([]types.CompletedPart, 0, len(parts))
	for _, part := range parts {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       aws.String(part.ETag),
			PartNumber: aws.Int32(int32(part.PartNumber)),
		})
	}

	// S3 requires parts ordered ascending
	slices.SortFunc(completedParts, func(a, b types.CompletedPart) int {
		return int(aws.ToInt32(a.PartNumber) - aws.ToInt32(b.PartNumber))
	})

	if _, err := me.s3.CompleteMultipartUpload(
		ctx,
		&s3.CompleteMultipartUploadInput{
			Bucket:   aws.String(videosBucket),
			Key:      aws.String(objectKey),
			UploadId: aws.String(uploadId),
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: completedParts,
			},
		},
	); err != nil {
		if e, ok := errors.AsType[smithy.APIError](err); ok {
			return fmt.Errorf("%w: %w", ErrInvalidCompleteUploadData, e)
		}
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	if err := qtx.MarkUploadAsCompleted(ctx, videoId); err != nil {
		return fmt.Errorf("failed to delete upload: %w", err)
	}

	if err := qtx.UpdateVideoStatus(ctx, queries.UpdateVideoStatusParams{
		Id:     videoId,
		Status: string(VideoStatusUploaded),
	}); err != nil {
		return fmt.Errorf("failed to update video status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	payload, err := json.Marshal(events.VideoIsReadyEventPayload{VideoId: videoId})
	if err != nil {
		return fmt.Errorf("failed to marshal %q event payload: %w", events.VideoIsReadyEvent, err)
	}
	if err := me.redis.Publish(ctx, events.VideoIsReadyEvent, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish %q event: %w", events.VideoIsReadyEvent, err)
	}

	return nil
}

const presignedGetExpiration = 1 * time.Hour

func (me *Service) GeneratePresignedGetUrl(ctx context.Context, videoId uuid.UUID) (string, error) {
	objectKey, err := me.queries.GetObjectKeyForVideo(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrObjectNotFound
		}
		return "", fmt.Errorf("failed to get object key: %w", err)
	}

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(videosBucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String("inline"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedGetExpiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign get url: %w", err)
	}

	return request.URL, nil
}

func (me *Service) videoDeletedEventConsumerJob(ctx context.Context) error {
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

			if err := func() error {
				tx, err := me.db.BeginTx(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to begin tx: %w", err)
				}
				defer tx.Rollback()
				qtx := me.queries.WithTx(tx)

				upload, err := qtx.GetUploadForVideo(ctx, payload.VideoId)
				if err != nil {
					return fmt.Errorf("failed to get upload for object: %w", err)
				}

				objectKey, err := qtx.GetObjectKeyForVideo(ctx, payload.VideoId)
				if err != nil {
					return fmt.Errorf("failed to get object key for video: %w", err)
				}

				if upload.CompletedAt.Valid {
					if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
						Bucket: aws.String(videosBucket),
						Key:    aws.String(objectKey),
					}); err != nil {
						// DeleteObject never returns NoSuchKey in S3.
						return fmt.Errorf("failed to delete S3 object: %w", err)
					}
				} else if upload.ExpiresAt.After(time.Now()) {
					if _, err := me.s3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
						Bucket:   aws.String(videosBucket),
						Key:      aws.String(objectKey),
						UploadId: aws.String(upload.Id),
					}); err != nil {
						return fmt.Errorf("failed to abort multipart upload: %w", err)
					}
				}

				if err := qtx.DeleteVideoById(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete video by id: %w", err)
				}

				if err := tx.Commit(); err != nil {
					return fmt.Errorf("failed to commit tx: %w", err)
				}

				return nil
			}(); err != nil {
				return err
			}

		case <-ctx.Done():
			return nil
		}
	}
}
