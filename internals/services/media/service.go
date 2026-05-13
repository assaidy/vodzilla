package media

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/media/queries"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const Name = "media"

var _ services.Service = (*Service)(nil)

type Service struct {
	db      *sql.DB
	queries *queries.Queries
	s3      *s3.Client
	logger  *slog.Logger
}

func New(db *sql.DB, s3 *s3.Client, logger *slog.Logger) *Service {
	return &Service{
		db:      db,
		queries: queries.New(db),
		s3:      s3,
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
	Urls     []string
}

func (me *Service) GeneratePresignedPutUrls(ctx context.Context, objectKey, contentType string, fileSize int64) (*PresignedUpload, error) {
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
			return nil, fmt.Errorf("failed to presign url: %w", err)
		}

		urls = append(urls, request.URL)
	}

	return &PresignedUpload{
		UploadId: uploadId,
		Urls:     urls,
	}, nil
}

type CompleteUploadPart struct {
	PartNumber int32
	ETag       string
}

func (me *Service) CompleteUpload(ctx context.Context, uploadId string, objectKey string, parts []CompleteUploadPart) error {
	completedParts := make([]types.CompletedPart, 0, len(parts))
	for _, part := range parts {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       aws.String(part.ETag),
			PartNumber: aws.Int32(part.PartNumber),
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
		return fmt.Errorf("faield to complete multipart upload: %w", err)
	}

	// TODO: publish video_uploaded event so video service can change video status

	return nil
}
