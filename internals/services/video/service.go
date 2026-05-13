package video

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/video/queries"
	"github.com/oklog/ulid/v2"
)

const Name = "video"

var _ services.Service = (*Service)(nil)

type Service struct {
	db      *sql.DB
	queries *queries.Queries
	logger  *slog.Logger
}

func New(db *sql.DB, logger *slog.Logger) *Service {
	return &Service{
		db:      db,
		queries: queries.New(db),
		logger:  logger,
	}
}

func (me *Service) Start(ctx context.Context) error { return nil }
func (me *Service) Stop(ctx context.Context) error  { return nil }

type CreateVideoParams struct {
	OwnerId     string
	Title       string
	Description string
}

type CreateVideoReuslt struct {
	VideoId   string
	ObjectKey string
}

// returns object key
func (me *Service) CreateVideo(ctx context.Context, params CreateVideoParams) (*CreateVideoReuslt, error) {
	videoId := ulid.Make().String()
	objectKey := fmt.Sprintf("%s/%s", params.OwnerId, videoId)

	if err := me.queries.InsertVideo(ctx, queries.InsertVideoParams{
		Id:          videoId,
		ObjectKey:   objectKey,
		OwnerId:     params.OwnerId,
		Title:       params.Title,
		Description: sql.NullString{Valid: params.Description != "", String: params.Description},
	}); err != nil {
		return nil, fmt.Errorf("failed to insert video: %w", err)
	}

	return &CreateVideoReuslt{
		VideoId:   videoId,
		ObjectKey: objectKey,
	}, nil
}
