package reaction

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
	"github.com/assaidy/vodzilla/internals/services/reaction/queries"
	"github.com/assaidy/workers"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

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
			fmt.Sprintf("%q event consumer", events.UserDeletedEvent),
			service.userDeletedEventConsumerJob,
			workers.WithRetryDelay(time.Second),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
		),
	)
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

func (me *Service) ViewVideo(ctx context.Context, videoId, userId uuid.UUID) error {
	_, err := me.queries.InsertView(ctx, queries.InsertViewParams{
		VideoId: videoId,
		UserId:  userId,
	})
	if err != nil {
		return fmt.Errorf("failed to insert view: %w", err)
	}

	return nil
}

func (me *Service) GetVideoViewsCount(ctx context.Context, videoId uuid.UUID) (int, error) {
	count, err := me.queries.GetViewsCount(ctx, videoId)
	if err != nil {
		return 0, fmt.Errorf("failed to get views count: %w", err)
	}

	return int(count), nil
}

func (me *Service) IsVideoViewedByUser(ctx context.Context, videoId, userId uuid.UUID) (bool, error) {
	ok, err := me.queries.CheckVideoViewer(ctx, queries.CheckVideoViewerParams{
		VideoId: videoId,
		UserId:  userId,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check video viewer: %w", err)
	}

	return ok, nil
}

func (me *Service) AddVidoeReaction(ctx context.Context, videoId, userId uuid.UUID, kind string) error {
	if err := me.queries.InsertReaction(ctx, queries.InsertReactionParams{
		VideoId: videoId,
		UserId:  userId,
		Kind:    kind,
	}); err != nil {
		return fmt.Errorf("failed to insert reaction (kind: %s): %w", kind, err)
	}

	return nil
}

func (me *Service) DeleteVidoeReaction(ctx context.Context, videoId, userId uuid.UUID, kind string) error {
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

func (me *Service) GetVideoReactionCounts(ctx context.Context, videoId uuid.UUID) (*VideoReactionCounts, error) {
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

func (me *Service) GetVideoReactionForUser(ctx context.Context, videoId, userId uuid.UUID) (*VideoReactionForUser, error) {
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

func (me *Service) GetCommentById(ctx context.Context, commentId uuid.UUID) (*Comment, error) {
	dbComment, err := me.queries.GetCommentById(ctx, commentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment by id: %w", err)
	}

	return &Comment{
		Id:           commentId,
		OwnerId:      dbComment.OwnerId,
		Content:      dbComment.Content,
		CreatedAt:    dbComment.CreatedAt,
		RepliesCount: int(dbComment.RepliesCount),
	}, nil
}

func (me *Service) CreateComment(ctx context.Context, videoId, userId uuid.UUID, content string, parentId uuid.UUID) (*uuid.UUID, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if parentId != uuid.Nil {
		if ok, err := qtx.CheckComment(ctx, parentId); err != nil {
			return nil, fmt.Errorf("failed to check parent comment: %w", err)
		} else if !ok {
			return nil, ErrParentCommentNotFound
		}
	}

	commentId := uuid.Must(uuid.NewV7())
	if err := qtx.InsertComment(ctx, queries.InsertCommentParams{
		Id:       commentId,
		OwnerId:  userId,
		VideoId:  videoId,
		Content:  content,
		ParentId: uuid.NullUUID{Valid: parentId != uuid.Nil, UUID: parentId},
	}); err != nil {
		return nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return &commentId, nil
}

func (me *Service) EditComment(ctx context.Context, userId, commentId uuid.UUID, newContent string) error {
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

func (me *Service) DeleteComment(ctx context.Context, userId, commentId uuid.UUID) error {
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
	Id           uuid.UUID
	ParentId     uuid.UUID
	OwnerId      uuid.UUID
	Content      string
	CreatedAt    time.Time
	RepliesCount int
}

func (me *Service) GetVideoCommentsCount(ctx context.Context, videoId uuid.UUID) (int, error) {
	count, err := me.queries.GetVideoCommentsCount(ctx, videoId)
	if err != nil {
		return 0, fmt.Errorf("failed to get comments count: %w", err)
	}
	return int(count), nil
}

func (me *Service) GetVideoComments(ctx context.Context, videoId, lastCommentId uuid.UUID, maxTimestamp time.Time) ([]Comment, error) {
	dbComments, err := me.queries.GetVideoComments(ctx, queries.GetVideoCommentsParams{
		VideoId:       videoId,
		LastCommentId: lastCommentId,
		MaxTimestamp:  maxTimestamp,
	})
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

func (me *Service) GetCommentReplies(ctx context.Context, commentId, lastCommentId uuid.UUID, maxTimestamp time.Time) ([]Comment, error) {
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

	dbComments, err := qtx.GetCommentReplies(ctx, queries.GetCommentRepliesParams{
		CommentId:     commentId,
		LastCommentId: lastCommentId,
		MaxTimestamp:  maxTimestamp,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get comment replies: %w", err)
	}

	result := make([]Comment, 0, len(dbComments))
	for _, c := range dbComments {
		result = append(result, Comment{
			Id:           c.Id,
			ParentId:     commentId,
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

			if err := func() error {
				tx, err := me.db.BeginTx(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to begin tx: %w", err)
				}
				defer tx.Rollback()
				qtx := me.queries.WithTx(tx)

				if err := qtx.DeleteAllViewsForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all views for user: %w", err)
				}

				if err := qtx.DeleteAllReactionsForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all reactions for user: %w", err)
				}

				if err := qtx.DeleteAllCommentsForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all comments for user: %w", err)
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

				if err := qtx.DeleteAllViewsForVideo(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all views for video: %w", err)
				}

				if err := qtx.DeleteAllReactionsForVideo(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all reactions for video: %w", err)
				}

				if err := qtx.DeleteAllCommentsForVideo(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all comments for video: %w", err)
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
