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
	"github.com/assaidy/workers/lock"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	services.Service
	GenerateVideoPresignedPutUrls(ctx context.Context, userId uuid.UUID, contentType string, fileSize int64) (*VideoPresignedUpload, error)
	ConfirmVideoUpload(ctx context.Context, objectKey, uploadId string, parts []CompleteVideoUploadPart) error
	DeleteOrphanUpload(ctx context.Context, objectKey string) error
	PostVideo(ctx context.Context, videoId uuid.UUID, objectKey string) error
	GenerateVideoPresignedGetUrl(ctx context.Context, videoId uuid.UUID) (string, error)
	GeneratePresignedAvatarUpload(ctx context.Context, userId uuid.UUID, contentType string, fileSize int64) (*AvatarPresignedUpload, error)
	ConfirmAvatarUpload(ctx context.Context, userId uuid.UUID) (string, error)
	DeleteAvatar(ctx context.Context, userId uuid.UUID) error
	GetAvatarUrl(ctx context.Context, userId uuid.UUID) (string, error)
	GeneratePresignedThumbnailUpload(ctx context.Context, videoId uuid.UUID, contentType string, fileSize int64) (*ThumbnailPresignedUpload, error)
	ConfirmThumbnailUpload(ctx context.Context, videoId uuid.UUID) (string, error)
	DeleteThumbnail(ctx context.Context, videoId uuid.UUID) error
	GetThumbnailUrl(ctx context.Context, videoId uuid.UUID) (string, error)
}

type impl struct {
	db            *sql.DB
	queries       *queries.Queries
	s3            *s3.Client
	redis         *redis.Client
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, redis *redis.Client, s3 *s3.Client, logger *slog.Logger) Service {
	service := &impl{
		db:      db,
		queries: queries.New(db),
		s3:      s3,
		redis:   redis,
		logger:  logger,
		workerManager: workers.NewWorkerManager(
			workers.WithLogger(logger),
			workers.WithLockGenerator(lock.NewRedisGenerator(redis)),
		),
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
	service.workerManager.RegisterWorker(
		workers.NewWorker(
			"orphan video uploads cleanup",
			service.orphanVideoUploadsCleanupJob,
			workers.WithSchedules(workers.WeeklyAt(time.Friday, 2, 0)),
			workers.WithTimeout(10*time.Minute),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
			workers.WithSingleInstance(),
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

const (
	videosBucket               = "videos"
	maxVideoPartCount          = 10_000
	minVideoPartSize           = 5 * utils.MegaByte
	presignedVideoUploadExpiry = 24 * time.Hour
	presignedVideoGetExpiry    = 1 * time.Hour

	avatarsBucket            = "avatars"
	presignedAvatarPutExpiry = 5 * time.Minute
	presignedAvatarGetExpiry = 1 * time.Hour

	thumbnailsBucket            = "thumbnails"
	presignedThumbnailPutExpiry = 5 * time.Minute
	presignedThumbnailGetExpiry = 1 * time.Hour
)

const videoUploadRedisPrefix = "media_service:video_upload:"

type VideoUploadChunk struct {
	Url    string
	Offset int64
	Size   int64
}

type VideoPresignedUpload struct {
	UploadId  string
	ObjectKey string
	Chunks    []VideoUploadChunk
}

// GenerateVideoPresignedPutUrls creates a multipart upload and returns presigned URLs for each part.
// Multipart upload works for all file sizes — files smaller than [minVideoPartSize]
// result in a single part, and S3 allows the last (and only) part to be smaller than 5 MB.
func (me *impl) GenerateVideoPresignedPutUrls(
	ctx context.Context,
	userId uuid.UUID,
	contentType string,
	fileSize int64,
) (*VideoPresignedUpload, error) {
	objectKey := uuid.Must(uuid.NewV7()).String()

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
	chunks := make([]VideoUploadChunk, 0, partCount)

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

		offset := int64(partNumber-1) * partSize
		size := partSize
		if offset+size > fileSize {
			size = fileSize - offset
		}

		chunks = append(chunks, VideoUploadChunk{
			Url:    request.URL,
			Offset: offset,
			Size:   size,
		})
	}

	payload, err := json.Marshal(map[string]any{
		"uploadId":  uploadId,
		"objectKey": objectKey,
		"userId":    userId.String(),
	})
	if err != nil {
		me.abortVideoUploadWithErrorLogging(ctx, objectKey, uploadId)
		return nil, fmt.Errorf("failed to marshal video upload payload: %w", err)
	}

	if err := me.redis.Set(
		ctx,
		videoUploadRedisPrefix+objectKey,
		payload,
		presignedVideoUploadExpiry,
	).Err(); err != nil {
		me.abortVideoUploadWithErrorLogging(ctx, objectKey, uploadId)
		return nil, fmt.Errorf("failed to store pending video upload: %w", err)
	}

	return &VideoPresignedUpload{
		UploadId:  uploadId,
		ObjectKey: objectKey,
		Chunks:    chunks,
	}, nil
}

func (me *impl) abortVideoUploadWithErrorLogging(ctx context.Context, objectKey, uploadId string) {
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

func (me *impl) ConfirmVideoUpload(
	ctx context.Context,
	objectKey, uploadId string,
	parts []CompleteVideoUploadPart,
) error {
	result, err := me.redis.Get(ctx, videoUploadRedisPrefix+objectKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrNoPendingVideoUpload
		}
		return fmt.Errorf("failed to get pending video upload: %w", err)
	}

	var pending struct {
		UploadId  string `json:"uploadId"`
		ObjectKey string `json:"objectKey"`
		UserId    string `json:"userId"`
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
			Key:      aws.String(objectKey),
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

	userId, err := uuid.Parse(pending.UserId)
	if err != nil {
		return fmt.Errorf("failed to parse user id: %w", err)
	}

	if err := me.queries.InsertOrphanUpload(ctx, queries.InsertOrphanUploadParams{
		ObjectKey: objectKey,
		UserId:    userId,
	}); err != nil {
		return fmt.Errorf("failed to insert orphan upload: %w", err)
	}

	if err := me.redis.Del(ctx, videoUploadRedisPrefix+objectKey).Err(); err != nil {
		me.logger.Error("failed to delete pending video upload from redis", "error", err)
	}

	return nil
}

func (me *impl) DeleteOrphanUpload(ctx context.Context, objectKey string) error {
	if err := me.queries.DeleteOrphanUpload(ctx, objectKey); err != nil {
		return fmt.Errorf("failed to delete orphan upload: %w", err)
	}
	return nil
}

func (me *impl) PostVideo(ctx context.Context, videoId uuid.UUID, objectKey string) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	ok, err := qtx.CheckOrphanUpload(ctx, objectKey)
	if err != nil {
		return fmt.Errorf("failed to check orphan upload: %w", err)
	}
	if !ok {
		return ErrOrphanUploadNotFound
	}

	if err := qtx.InsertVideo(ctx, queries.InsertVideoParams{
		Id:        videoId,
		ObjectKey: objectKey,
	}); err != nil {
		return fmt.Errorf("failed to insert video: %w", err)
	}

	if err := qtx.DeleteOrphanUpload(ctx, objectKey); err != nil {
		return fmt.Errorf("failed to delete orphan upload: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *impl) GenerateVideoPresignedGetUrl(ctx context.Context, videoId uuid.UUID) (string, error) {
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
const thumbnailUploadRedisPrefix = "media_service:thumbnail_upload:"

func avatarObjectKey(userId uuid.UUID) string {
	return fmt.Sprintf("avatars/%s", userId)
}

func (me *impl) GeneratePresignedAvatarUpload(
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

func (me *impl) ConfirmAvatarUpload(ctx context.Context, userId uuid.UUID) (string, error) {
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

func (me *impl) DeleteAvatar(ctx context.Context, userId uuid.UUID) error {
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

func (me *impl) GetAvatarUrl(ctx context.Context, userId uuid.UUID) (string, error) {
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

type ThumbnailPresignedUpload struct {
	UploadUrl string
	ObjectKey string
}

func thumbnailObjectKey(videoId uuid.UUID) string {
	return fmt.Sprintf("thumbnails/%s", videoId)
}

func (me *impl) GeneratePresignedThumbnailUpload(
	ctx context.Context,
	videoId uuid.UUID,
	contentType string,
	fileSize int64,
) (*ThumbnailPresignedUpload, error) {
	objectKey := thumbnailObjectKey(videoId)

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(thumbnailsBucket),
		Key:           aws.String(objectKey),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(fileSize),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedThumbnailPutExpiry
	})
	if err != nil {
		return nil, fmt.Errorf("failed to presign put url: %w", err)
	}

	if err := me.redis.Set(
		ctx,
		thumbnailUploadRedisPrefix+videoId.String(),
		objectKey,
		presignedThumbnailPutExpiry,
	).Err(); err != nil {
		return nil, fmt.Errorf("failed to store pending thumbnail upload: %w", err)
	}

	return &ThumbnailPresignedUpload{
		UploadUrl: request.URL,
		ObjectKey: objectKey,
	}, nil
}

func (me *impl) ConfirmThumbnailUpload(ctx context.Context, videoId uuid.UUID) (string, error) {
	objectKey, err := me.redis.Get(ctx, thumbnailUploadRedisPrefix+videoId.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNoPendingThumbnailUpload
		}
		return "", fmt.Errorf("failed to get pending thumbnail upload: %w", err)
	}

	if _, err := me.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(thumbnailsBucket),
		Key:    aws.String(objectKey),
	}); err != nil {
		return "", fmt.Errorf("failed to head thumbnail object: %w", err)
	}

	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if existingKey, err := qtx.GetThumbnailByVideoId(ctx, videoId); err == nil {
		if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(thumbnailsBucket),
			Key:    aws.String(existingKey),
		}); err != nil {
			return "", fmt.Errorf("failed to delete old thumbnail from s3: %w", err)
		}

		if err := qtx.DeleteThumbnailByVideoId(ctx, videoId); err != nil {
			return "", fmt.Errorf("failed to delete old thumbnail from db: %w", err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("failed to get existing thumbnail: %w", err)
	}

	if err := qtx.InsertThumbnail(ctx, queries.InsertThumbnailParams{
		VideoId:   videoId,
		ObjectKey: objectKey,
	}); err != nil {
		return "", fmt.Errorf("failed to insert thumbnail: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit tx: %w", err)
	}

	if err := me.redis.Del(ctx, thumbnailUploadRedisPrefix+videoId.String()).Err(); err != nil {
		me.logger.Error("failed to delete pending thumbnail upload from redis", "error", err)
	}

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(thumbnailsBucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String("inline"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedThumbnailGetExpiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign get url: %w", err)
	}

	return request.URL, nil
}

func (me *impl) DeleteThumbnail(ctx context.Context, videoId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	objectKey, err := qtx.GetThumbnailByVideoId(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrThumbnailNotFound
		}
		return fmt.Errorf("failed to get thumbnail: %w", err)
	}

	if err := qtx.DeleteThumbnailByVideoId(ctx, videoId); err != nil {
		return fmt.Errorf("failed to delete thumbnail from db: %w", err)
	}

	if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(thumbnailsBucket),
		Key:    aws.String(objectKey),
	}); err != nil {
		return fmt.Errorf("failed to delete thumbnail from s3: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *impl) GetThumbnailUrl(ctx context.Context, videoId uuid.UUID) (string, error) {
	objectKey, err := me.queries.GetThumbnailByVideoId(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrThumbnailNotFound
		}
		return "", fmt.Errorf("failed to get thumbnail: %w", err)
	}

	presigner := s3.NewPresignClient(me.s3)
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(thumbnailsBucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String("inline"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignedThumbnailGetExpiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign get url: %w", err)
	}

	return request.URL, nil
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

				if thumbnailKey, err := qtx.GetThumbnailByVideoId(ctx, payload.VideoId); err == nil {
					if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
						Bucket: aws.String(thumbnailsBucket),
						Key:    aws.String(thumbnailKey),
					}); err != nil {
						me.logger.Error("failed to delete thumbnail from s3", "error", err)
					}

					if err := qtx.DeleteThumbnailByVideoId(ctx, payload.VideoId); err != nil {
						me.logger.Error("failed to delete thumbnail from db", "error", err)
					}
				} else if !errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("failed to get thumbnail for video: %w", err)
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

// FIX: reform the upload setup by first creating the video in video service then waiting for it to be uploaded.
// - media service will controle the expiration and will publish events: VideoUploadExpired, VideoUploadCompleted
// - video service will have multiple states for a video: uploading, transcoding, ..., published
// - remove redundant cleanup workers and tables in both services.
func (me *impl) orphanVideoUploadsCleanupJob(ctx context.Context) error {
	objectKeys, err := me.queries.GetExpiredOrphanUploads(ctx)
	if err != nil {
		return fmt.Errorf("failed to list expired orphan uploads: %w", err)
	}

	for _, key := range objectKeys {
		if _, err := me.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(videosBucket),
			Key:    aws.String(key),
		}); err != nil {
			return fmt.Errorf("failed to delete expired orphan object: %w", err)
		}

		if err := me.queries.DeleteOrphanUpload(ctx, key); err != nil {
			return fmt.Errorf("failed to delete expired orphan upload: %w", err)
		}
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

			if err := me.DeleteAvatar(ctx, payload.UserId); err != nil && !errors.Is(err, ErrAvatarNotFound) {
				return fmt.Errorf("failed to delete avatar for deleted user: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
