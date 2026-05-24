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

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/media/queries"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/redis/go-redis/v9"
)

const Name = "media"

var _ services.Service = (*Service)(nil)

type Service struct {
	db      *sql.DB
	queries *queries.Queries
	s3      *s3.Client
	redis   *redis.Client
	logger  *slog.Logger
}

func New(db *sql.DB, s3 *s3.Client, redis *redis.Client, logger *slog.Logger) *Service {
	return &Service{
		db:      db,
		queries: queries.New(db),
		s3:      s3,
		redis:   redis,
		logger:  logger,
	}
}

func (me *Service) Start(ctx context.Context) error { return nil }
func (me *Service) Stop(ctx context.Context) error  { return nil }

const (
	videosBucket = "videos"
	maxPartCount = 10_000
	minPartSize  = 5 << 20 // 5 MB
)

type PresignedUpload struct {
	UploadId string
	PartSize int64
	Urls     []string
}

// TODO: after generating presigned PUT URLs, store video_id in Redis with TTL
//	SET upload_ttl:{video_id} "" EX <timeout>
//
// TODO: register cleanup worker for expired uploads:
//	- scan Redis for expired/missing upload_ttl keys
//	- for expired entries: abort S3 multipart upload, delete object_keys DB record
//	- publish UploadExpiredEvent{VideoId} for video service to delete the pending video metadata
func (me *Service) GeneratePresignedPutUrls(ctx context.Context, videoId, objectKey, contentType string, fileSize int64) (*PresignedUpload, error) {
	createOut, err := me.s3.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(videosBucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}
	uploadId := aws.ToString(createOut.UploadId)

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
				opts.Expires = 15 * time.Minute
			},
		)
		if err != nil {
			me.abortUpload(ctx, objectKey, uploadId)
			return nil, fmt.Errorf("failed to presign url: %w", err)
		}

		urls = append(urls, request.URL)
	}

	if err := me.queries.InsertObjectKey(ctx, queries.InsertObjectKeyParams{
		VideoId:   videoId,
		ObjectKey: objectKey,
	}); err != nil {
		me.abortUpload(ctx, objectKey, uploadId)
		return nil, fmt.Errorf("failed to insert object key: %w", err)
	}

	return &PresignedUpload{
		UploadId: uploadId,
		Urls:     urls,
		PartSize: partSize,
	}, nil
}

type CompleteUploadPart struct {
	ETag       string
	PartNumber int
}

func (me *Service) CompleteUpload(ctx context.Context, videoId, uploadId string, parts []CompleteUploadPart) error {
	objectKey, err := me.queries.GetObjectKeyForVideo(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("failed to get object key: %w", err)
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

	payload, err := json.Marshal(events.VideoUploadedEventPayload{
		VideoId:   videoId,
		Timestamp: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal %q event payload: %w", events.VideoUploadedEvent, err)
	}
	if err := me.redis.Publish(ctx, events.VideoUploadedEvent, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish %q event: %w", events.VideoUploadedEvent, err)
	}

	return nil
}

func (me *Service) abortUpload(ctx context.Context, objectKey, uploadId string) {
	if _, err := me.s3.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(videosBucket),
		Key:      aws.String(objectKey),
		UploadId: aws.String(uploadId),
	}); err != nil {
		me.logger.Error("failed to abort multipart upload",
			"object_key", objectKey, "upload_id", uploadId, "error", err)
	}
}

func (me *Service) GeneratePresignedGetUrl(ctx context.Context, videoId string) (string, error) {
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
		opts.Expires = 1 * time.Hour
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign get url: %w", err)
	}

	return request.URL, nil
}
