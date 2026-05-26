package reaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/reaction/queries"
	"github.com/oklog/ulid/v2"
)

// TODO: consume UserDeletedEvent to mark user's reactions and views as deleted
// TODO: consume VideoDeletedEvent to mark video's reactions and views as deleted
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

func (me *Service) CreateComment(ctx context.Context, videoId string, userId string, content string, parentId string) (string, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if parentId != "" {
		if ok, err := qtx.CheckComment(ctx, parentId); err != nil {
			return "", fmt.Errorf("failed to check parent comment: %w", err)
		} else if !ok {
			return "", ErrParentCommentNotFound
		}
	}

	commentId := ulid.Make().String()
	if err := qtx.InsertComment(ctx, queries.InsertCommentParams{
		Id:       commentId,
		OwnerId:  userId,
		VideoId:  videoId,
		Content:  content,
		ParentId: sql.NullString{Valid: parentId != "", String: parentId},
	}); err != nil {
		return "", fmt.Errorf("failed to insert comment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit tx: %w", err)
	}

	return commentId, nil
}

func (me *Service) EditComment(ctx context.Context, userId string, commentId string, newContent string) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckCommentForUser(ctx, queries.CheckCommentForUserParams{
		Id:      commentId,
		OwnerId: userId,
	}); err != nil {
		return fmt.Errorf("failed to check comment: %w", err)
	} else if !ok {
		return ErrCommentNotFound
	}

	if err := qtx.UpdateComment(ctx, queries.UpdateCommentParams{
		Content: newContent,
		Id:      commentId,
	}); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *Service) DeleteComment(ctx context.Context, userId string, commentId string) error {
	if n, err := me.queries.DeleteComment(ctx, queries.DeleteCommentParams{
		Id:      commentId,
		OwnerId: userId,
	}); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	} else if n == 0 {
		return ErrCommentNotFound
	}

	return nil
}

type Comment struct {
	Id           string
	OwnerId      string
	Content      string
	CreatedAt    time.Time
	RepliesCount int
}

func (me *Service) GetAllVideoComments(ctx context.Context, videoId string) ([]Comment, error) {
	dbComments, err := me.queries.GetAllVideoComments(ctx, videoId)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments on video: %w", err)
	}

	result := make([]Comment, 0, len(dbComments))
	for _, c := range dbComments {
		result = append(result, Comment{
			Id:           c.Id,
			OwnerId:      c.OwnerId,
			Content:      c.Content,
			CreatedAt:    c.CreatedAt,
			RepliesCount: int(c.RepliesCount),
		})
	}

	return result, nil
}

func (me *Service) GetAllCommentReplies(ctx context.Context, commentId string) ([]Comment, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckComment(ctx, commentId); err != nil {
		return nil, fmt.Errorf("failed to check comment: %w", err)
	} else if !ok {
		return nil, ErrCommentNotFound
	}

	dbComments, err := qtx.GetAllCommentReplies(ctx, commentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment replies: %w", err)
	}

	result := make([]Comment, 0, len(dbComments))
	for _, c := range dbComments {
		result = append(result, Comment{
			Id:           c.Id,
			OwnerId:      c.OwnerId,
			Content:      c.Content,
			CreatedAt:    c.CreatedAt,
			RepliesCount: int(c.RepliesCount),
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return result, nil
}
