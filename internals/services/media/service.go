package media

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/media/queries"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/workers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/redis/go-redis/v9"
)

var _ services.Service = (*Service)(nil)

type Service struct {
	db            *sql.DB
	queries       *queries.Queries
	s3            *s3.Client
	redis         *redis.Client
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, redis *redis.Client, s3 *s3.Client, logger *slog.Logger) *Service {
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

const (
	videosBucket               = "videos"
	maxVideoPartCount          = 10_000
	minVideoPartSize           = 5 * utils.MegaByte
	presignedVideoUploadExpiry = 24 * time.Hour
	presignedVideoGetExpiry    = 1 * time.Hour

	avatarsBucket            = "avatars"
	presignedAvatarPutExpiry = 5 * time.Minute
	presignedAvatarGetExpiry = 1 * time.Hour
)

const videoUploadRedisPrefix = "media_service:video_upload:"

type VideoPresignedUpload struct {
	UploadId string
	PartSize int64
	Urls     []string
}

func (me *Service) GenerateVideoPresignedPutUrls(
	ctx context.Context,
	videoId uuid.UUID,
	objectKey, contentType string,
	fileSize int64,
) (*VideoPresignedUpload, error) {
	createResult, err := me.s3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(videosBucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}
	uploadId := aws.ToString(createResult.UploadId)

	requiredPartSize := (fileSize + maxVideoPartCount - 1) / maxVideoPartCount
	partSize := max(minVideoPartSize, requiredPartSize)
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
			me.abortVideoUploadWithErrorLogging(ctx, objectKey, uploadId)
			return nil, fmt.Errorf("failed to presign url: %w", err)
		}

		urls = append(urls, request.URL)
	}

	payload, err := json.Marshal(map[string]any{
		"uploadId":  uploadId,
		"objectKey": objectKey,
	})
	if err != nil {
		me.abortVideoUploadWithErrorLogging(ctx, objectKey, uploadId)
		return nil, fmt.Errorf("failed to marshal video upload payload: %w", err)
	}

	if err := me.redis.Set(
		ctx,
		videoUploadRedisPrefix+videoId.String(),
		payload,
		presignedVideoUploadExpiry,
	).Err(); err != nil {
		me.abortVideoUploadWithErrorLogging(ctx, objectKey, uploadId)
		return nil, fmt.Errorf("failed to store pending video upload: %w", err)
	}

	// FIX: instead of returning urls slice, i wanna return a slice of chunks, each with offset and size.
	return &VideoPresignedUpload{
		UploadId: uploadId,
		Urls:     urls,
		PartSize: partSize,
	}, nil
}

func (me *Service) abortVideoUploadWithErrorLogging(ctx context.Context, objectKey, uploadId string) {
	if _, err := me.s3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(videosBucket),
		Key:      aws.String(objectKey),
		UploadId: aws.String(uploadId),
	}); err != nil {
		me.logger.Error("failed to abort multipart upload",
			"object_key", objectKey, "upload_id", uploadId, "error", err)
	}
}

type CompleteVideoUploadPart struct {
	ETag       string
	PartNumber int
}

func (me *Service) ConfirmVideoUpload(
	ctx context.Context,
	videoId uuid.UUID,
	uploadId string,
	parts []CompleteVideoUploadPart,
) error {
	result, err := me.redis.Get(ctx, videoUploadRedisPrefix+videoId.String()).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrNoPendingVideoUpload
		}
		return fmt.Errorf("failed to get pending video upload: %w", err)
	}

	var pending struct {
		UploadId  string `json:"upload_id"`
		ObjectKey string `json:"object_key"`
	}
	if err := json.Unmarshal(result, &pending); err != nil {
		return fmt.Errorf("failed to unmarshal pending video upload: %w", err)
	}

	if pending.UploadId != uploadId {
		return ErrInvalidConfirmVideoUploadData
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
			Key:      aws.String(pending.ObjectKey),
			UploadId: aws.String(uploadId),
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: completedParts,
			},
		},
	); err != nil {
		if e, ok := errors.AsType[smithy.APIError](err); ok {
			return fmt.Errorf("%w: %w", ErrInvalidConfirmVideoUploadData, e)
		}
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	if err := me.queries.InsertVideo(ctx, queries.InsertVideoParams{
		Id:        videoId,
		ObjectKey: pending.ObjectKey,
	}); err != nil {
		return fmt.Errorf("failed to insert video: %w", err)
	}

	if err := me.redis.Del(ctx, videoUploadRedisPrefix+videoId.String()); err != nil {
		me.logger.Error("failed to delete pending video upload from redis", "error", err)
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

func (me *Service) GenerateVideoPresignedGetUrl(ctx context.Context, videoId uuid.UUID) (string, error) {
	objectKey, err := me.queries.GetObjectKeyForVideo(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrVideoNotFound
		}
		return "", fmt.Errorf("failed to get object key: %w", err)
	}

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(videosBucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String("inline"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedVideoGetExpiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign get url: %w", err)
	}

	return request.URL, nil
}

type AvatarPresignedUpload struct {
	UploadUrl string
	ObjectKey string
}

const avatarUploadRedisPrefix = "media_service:avatar_upload:"

func avatarObjectKey(userId uuid.UUID) string {
	return fmt.Sprintf("avatars/%s", userId)
}

func (me *Service) GeneratePresignedAvatarUpload(
	ctx context.Context,
	userId uuid.UUID,
	contentType string,
	fileSize int64,
) (*AvatarPresignedUpload, error) {
	objectKey := avatarObjectKey(userId)

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(avatarsBucket),
		Key:           aws.String(objectKey),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(fileSize),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedAvatarPutExpiry
	})
	if err != nil {
		return nil, fmt.Errorf("failed to presign put url: %w", err)
	}

	if err := me.redis.Set(
		ctx,
		avatarUploadRedisPrefix+userId.String(),
		objectKey,
		presignedAvatarPutExpiry,
	).Err(); err != nil {
		return nil, fmt.Errorf("failed to store pending avatar upload: %w", err)
	}

	return &AvatarPresignedUpload{
		UploadUrl: request.URL,
		ObjectKey: objectKey,
	}, nil
}

func (me *Service) ConfirmAvatarUpload(ctx context.Context, userId uuid.UUID) (string, error) {
	objectKey, err := me.redis.Get(ctx, avatarUploadRedisPrefix+userId.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNoPendingAvatarUpload
		}
		return "", fmt.Errorf("failed to get pending avatar upload: %w", err)
	}

	if _, err := me.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(avatarsBucket),
		Key:    aws.String(objectKey),
	}); err != nil {
		return "", fmt.Errorf("failed to head avatar object: %w", err)
	}

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if existingKey, err := qtx.GetAvatarByUserId(ctx, userId); err == nil {
		if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(avatarsBucket),
			Key:    aws.String(existingKey),
		}); err != nil {
			return "", fmt.Errorf("failed to delete old avatar from s3: %w", err)
		}

		if err := qtx.DeleteAvatarByUserId(ctx, userId); err != nil {
			return "", fmt.Errorf("failed to delete old avatar from db: %w", err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("failed to get existing avatar: %w", err)
	}

	if err := qtx.InsertAvatar(ctx, queries.InsertAvatarParams{
		UserId:    userId,
		ObjectKey: objectKey,
	}); err != nil {
		return "", fmt.Errorf("failed to insert avatar: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit tx: %w", err)
	}

	if err := me.redis.Del(ctx, avatarUploadRedisPrefix+userId.String()).Err(); err != nil {
		me.logger.Error("failed to delete pending avatar upload from redis", "error", err)
	}

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(avatarsBucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String("inline"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedAvatarGetExpiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign get url: %w", err)
	}

	return request.URL, nil
}

func (me *Service) DeleteAvatar(ctx context.Context, userId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	objectKey, err := qtx.GetAvatarByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAvatarNotFound
		}
		return fmt.Errorf("failed to get avatar: %w", err)
	}

	if err := qtx.DeleteAvatarByUserId(ctx, userId); err != nil {
		return fmt.Errorf("failed to delete avatar from db: %w", err)
	}

	if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(avatarsBucket),
		Key:    aws.String(objectKey),
	}); err != nil {
		return fmt.Errorf("failed to delete avatar from s3: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *Service) GetAvatarUrl(ctx context.Context, userId uuid.UUID) (string, error) {
	objectKey, err := me.queries.GetAvatarByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrAvatarNotFound
		}
		return "", fmt.Errorf("failed to get avatar: %w", err)
	}

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(avatarsBucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String("inline"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedAvatarGetExpiry
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

				objectKey, err := qtx.GetObjectKeyForVideo(ctx, payload.VideoId)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return nil
					}
					return fmt.Errorf("failed to get object key for video: %w", err)
				}

				if err := qtx.DeleteVideoById(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete video by id: %w", err)
				}

				if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: aws.String(videosBucket),
					Key:    aws.String(objectKey),
				}); err != nil {
					return fmt.Errorf("failed to delete S3 object: %w", err)
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

			if err := me.DeleteAvatar(ctx, payload.UserId); err != nil && !errors.Is(err, ErrAvatarNotFound) {
				return fmt.Errorf("failed to delete avatar for deleted user: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
