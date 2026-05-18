package reaction

import (
	"context"
	"database/sql"
	"errors"
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

func (me *Service) AddVidoeReaction(ctx context.Context, videoId, userId, kind string) error {
	if err := me.queries.InsertReaction(ctx, queries.InsertReactionParams{
		VideoId: videoId,
		UserId:  userId,
		Kind:    kind,
	}); err != nil {
		return fmt.Errorf("failed to insert reaction (kind: %s): %w", kind, err)
	}

	return nil
}

func (me *Service) DeleteVidoeReaction(ctx context.Context, videoId, userId, kind string) error {
	if err := me.queries.DeleteReaction(ctx, queries.DeleteReactionParams{
		VideoId: videoId,
		UserId:  userId,
		Kind:    kind,
	}); err != nil {
		return fmt.Errorf("failed to delete reaction (kind: %s): %w", kind, err)
	}

	return nil
}

type VideoReactionCounts struct {
	Likes    int
	Dislikes int
}

func (me *Service) GetVideoReactionCounts(ctx context.Context, videoId string) (*VideoReactionCounts, error) {
	reactions, err := me.queries.GetVideoReactions(ctx, videoId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get video reactions: %w", err)
	}

	return &VideoReactionCounts{
		Likes:    int(reactions.Likes),
		Dislikes: int(reactions.Dislikes),
	}, nil
}

type VideoReactionForUser struct {
	IsLike    bool
	IsDislike bool
}

func (me *Service) GetVideoReactionForUser(ctx context.Context, videoId string, userId string) (*VideoReactionForUser, error) {
	reaction, err := me.queries.GetVideoReactionForUser(ctx, queries.GetVideoReactionForUserParams{
		VideoId: videoId,
		UserId:  userId,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get reaction for user: %w", err)
	}

	return &VideoReactionForUser{
		IsLike:    reaction.IsLike,
		IsDislike: reaction.IsDislike,
	}, nil
}
