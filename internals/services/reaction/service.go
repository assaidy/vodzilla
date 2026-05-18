package reaction

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/reaction/queries"
)

const Name = "reaction"

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

func (me *Service) ViewVideo(ctx context.Context, videoId, userId string) error {
	_, err := me.queries.InsertView(ctx, queries.InsertViewParams{
		VideoId: videoId,
		UserId:  userId,
	})
	if err != nil {
		return fmt.Errorf("failed to insert view: %w", err)
	}

	return nil
}

func (me *Service) GetVideoViewsCount(ctx context.Context, videoId string) (int, error) {
	count, err := me.queries.GetViewsCount(ctx, videoId)
	if err != nil {
		return 0, fmt.Errorf("failed to get views count: %w", err)
	}
	return int(count), nil
}
