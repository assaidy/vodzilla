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

type Service interface {
	services.Service
	ViewVideo(ctx context.Context, videoId, userId uuid.UUID) error
	GetVideoViewsCount(ctx context.Context, videoId uuid.UUID) (int, error)
	IsVideoViewedByUser(ctx context.Context, videoId, userId uuid.UUID) (bool, error)
	AddVideoFeeling(ctx context.Context, userId, videoId uuid.UUID, kind FeelingKind) error
	DeleteVideoFeeling(ctx context.Context, userId, videoId uuid.UUID) error
	AddCommentFeeling(ctx context.Context, userId, commentId uuid.UUID, kind FeelingKind) error
	DeleteCommentFeeling(ctx context.Context, userId, commentId uuid.UUID) error
	GetFeelingCounts(ctx context.Context, forId uuid.UUID) (*FeelingCounts, error)
	GetUserFeeling(ctx context.Context, forId, userId uuid.UUID) (FeelingKind, error)
	GetCommentOwner(ctx context.Context, commentId uuid.UUID) (uuid.UUID, error)
	CreateVideoComment(ctx context.Context, userId, videoId uuid.UUID, content string) (uuid.UUID, error)
	EditComment(ctx context.Context, userId, commentId uuid.UUID, newContent string) error
	DeleteComment(ctx context.Context, userId, commentId uuid.UUID) error
	GetVideoCommentsCount(ctx context.Context, videoId uuid.UUID) (int, error)
	GetVideoComments(ctx context.Context, videoId, lastCommentId uuid.UUID, limit int) ([]Comment, error)
	CreateCommentReply(ctx context.Context, userId, commentId uuid.UUID, content string) (uuid.UUID, error)
	GetCommentReplies(ctx context.Context, commentId uuid.UUID, lastCommentId uuid.UUID, limit int) ([]Comment, error)
}

type impl struct {
	db            *sql.DB
	queries       *queries.Queries
	redis         *redis.Client
	logger        *slog.Logger
	workerManager *workers.WorkerManager
}

func New(db *sql.DB, redis *redis.Client, logger *slog.Logger) Service {
	service := &impl{
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

func (me *impl) Start(ctx context.Context) error {
	me.workerManager.Start()
	return nil
}

func (me *impl) Stop(ctx context.Context) error {
	me.workerManager.Stop()
	return nil
}

func (me *impl) ViewVideo(ctx context.Context, videoId, userId uuid.UUID) error {
	_, err := me.queries.InsertView(ctx, queries.InsertViewParams{
		VideoId: videoId,
		UserId:  userId,
	})
	if err != nil {
		return fmt.Errorf("failed to insert view: %w", err)
	}

	return nil
}

func (me *impl) GetVideoViewsCount(ctx context.Context, videoId uuid.UUID) (int, error) {
	count, err := me.queries.GetViewsCount(ctx, videoId)
	if err != nil {
		return 0, fmt.Errorf("failed to get views count: %w", err)
	}

	return int(count), nil
}

func (me *impl) IsVideoViewedByUser(ctx context.Context, videoId, userId uuid.UUID) (bool, error) {
	ok, err := me.queries.CheckVideoViewer(ctx, queries.CheckVideoViewerParams{
		VideoId: videoId,
		UserId:  userId,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check video viewer: %w", err)
	}

	return ok, nil
}

type FeelingKind string

const (
	FeelingLike    FeelingKind = "like"
	FeelingDislike FeelingKind = "dislike"
)

func (k FeelingKind) isValid() bool {
	return k == FeelingLike || k == FeelingDislike
}

func (me *impl) AddVideoFeeling(ctx context.Context, userId, videoId uuid.UUID, kind FeelingKind) error {
	if !kind.isValid() {
		return fmt.Errorf("invalid feeling kind: %q", kind)
	}
	if err := me.queries.UpsertFeeling(ctx, queries.UpsertFeelingParams{
		ForId:  videoId,
		UserId: userId,
		Kind:   string(kind),
	}); err != nil {
		return fmt.Errorf("failed to upsert video feeling (kind: %s): %w", kind, err)
	}

	return nil
}

func (me *impl) DeleteVideoFeeling(ctx context.Context, userId, videoId uuid.UUID) error {
	n, err := me.queries.DeleteFeeling(ctx, queries.DeleteFeelingParams{
		ForId:  videoId,
		UserId: userId,
	})
	if err != nil {
		return fmt.Errorf("failed to delete video feeling: %w", err)
	} else if n == 0 {
		return ErrFeelingNotFound
	}

	return nil
}

func (me *impl) AddCommentFeeling(ctx context.Context, userId, commentId uuid.UUID, kind FeelingKind) error {
	if !kind.isValid() {
		return fmt.Errorf("invalid feeling kind: %q", kind)
	}
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckComment(ctx, commentId); err != nil {
		return fmt.Errorf("failed to check comment: %w", err)
	} else if !ok {
		return ErrCommentNotFound
	}

	if err := qtx.UpsertFeeling(ctx, queries.UpsertFeelingParams{
		ForId:  commentId,
		UserId: userId,
		Kind:   string(kind),
	}); err != nil {
		return fmt.Errorf("failed to upsert comment feeling (kind: %s): %w", kind, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *impl) DeleteCommentFeeling(ctx context.Context, userId, commentId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckComment(ctx, commentId); err != nil {
		return fmt.Errorf("failed to check comment: %w", err)
	} else if !ok {
		return ErrCommentNotFound
	}

	n, err := qtx.DeleteFeeling(ctx, queries.DeleteFeelingParams{
		ForId:  commentId,
		UserId: userId,
	})
	if err != nil {
		return fmt.Errorf("failed to delete comment feeling: %w", err)
	} else if n == 0 {
		return ErrFeelingNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

type FeelingCounts struct {
	Likes    int
	Dislikes int
}

func (me *impl) GetFeelingCounts(ctx context.Context, forId uuid.UUID) (*FeelingCounts, error) {
	counts, err := me.queries.GetFeelingCounts(ctx, forId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get feeling counts: %w", err)
	}

	return &FeelingCounts{
		Likes:    int(counts.Likes),
		Dislikes: int(counts.Dislikes),
	}, nil
}

func (me *impl) GetUserFeeling(ctx context.Context, forId, userId uuid.UUID) (FeelingKind, error) {
	kind, err := me.queries.GetUserFeeling(ctx, queries.GetUserFeelingParams{
		ForId:  forId,
		UserId: userId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get user feeling: %w", err)
	}

	return FeelingKind(kind), nil
}

func (me *impl) GetCommentOwner(ctx context.Context, commentId uuid.UUID) (uuid.UUID, error) {
	ownerId, err := me.queries.GetCommentOwner(ctx, commentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrCommentNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get comment owner: %w", err)
	}

	return ownerId, nil
}

func (me *impl) CreateVideoComment(ctx context.Context, userId, videoId uuid.UUID, content string) (uuid.UUID, error) {
	commentId := uuid.Must(uuid.NewV7())
	if err := me.queries.InsertComment(ctx, queries.InsertCommentParams{
		Id:      commentId,
		ForId:   videoId,
		UserId:  userId,
		Content: content,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	return commentId, nil
}

func (me *impl) EditComment(ctx context.Context, userId, commentId uuid.UUID, newContent string) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckCommentForUser(ctx, queries.CheckCommentForUserParams{
		Id:     commentId,
		UserId: userId,
	}); err != nil {
		return fmt.Errorf("failed to check comment: %w", err)
	} else if !ok {
		return ErrCommentNotFound
	}

	if err := qtx.UpdateComment(ctx, queries.UpdateCommentParams{
		Id:      commentId,
		Content: newContent,
	}); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *impl) DeleteComment(ctx context.Context, userId, commentId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckCommentForUser(ctx, queries.CheckCommentForUserParams{
		Id:     commentId,
		UserId: userId,
	}); err != nil {
		return fmt.Errorf("failed to check comment: %w", err)
	} else if !ok {
		return ErrCommentNotFound
	}

	if err := qtx.DeleteComment(ctx, queries.DeleteCommentParams{
		Id:     commentId,
		UserId: userId,
	}); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	if err := qtx.DeleteAllCommentsFor(ctx, commentId); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

type Comment struct {
	Id           uuid.UUID
	UserId       uuid.UUID
	Content      string
	CreatedAt    time.Time
	RepliesCount int
}

func (me *impl) GetVideoCommentsCount(ctx context.Context, videoId uuid.UUID) (int, error) {
	count, err := me.queries.GetCommentsCount(ctx, videoId)
	if err != nil {
		return 0, fmt.Errorf("failed to get comments count: %w", err)
	}
	return int(count), nil
}

func (me *impl) GetVideoComments(ctx context.Context, videoId, lastCommentId uuid.UUID, limit int) ([]Comment, error) {
	dbComments, err := me.queries.GetComments(ctx, queries.GetCommentsParams{
		ForId:         videoId,
		LastCommentId: uuid.NullUUID{UUID: lastCommentId, Valid: lastCommentId != uuid.Nil},
		Limit:         int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get comments on video: %w", err)
	}

	result := make([]Comment, 0, len(dbComments))
	for _, c := range dbComments {
		result = append(result, Comment{
			Id:           c.Id,
			UserId:       c.UserId,
			Content:      c.Content,
			CreatedAt:    c.CreatedAt,
			RepliesCount: int(c.RepliesCount),
		})
	}

	return result, nil
}

func (me *impl) CreateCommentReply(ctx context.Context, userId, commentId uuid.UUID, content string) (uuid.UUID, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckCommentForUser(ctx, queries.CheckCommentForUserParams{
		Id:     commentId,
		UserId: userId,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to check comment: %w", err)
	} else if !ok {
		return uuid.Nil, ErrCommentNotFound
	}

	replyId := uuid.Must(uuid.NewV7())
	if err := qtx.InsertComment(ctx, queries.InsertCommentParams{
		Id:      replyId,
		ForId:   commentId,
		UserId:  userId,
		Content: content,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return replyId, nil
}

func (me *impl) GetCommentReplies(ctx context.Context, commentId uuid.UUID, lastCommentId uuid.UUID, limit int) ([]Comment, error) {
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

	dbComments, err := qtx.GetComments(ctx, queries.GetCommentsParams{
		ForId:         commentId,
		LastCommentId: uuid.NullUUID{UUID: lastCommentId, Valid: lastCommentId != uuid.Nil},
		Limit:         int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get comment replies: %w", err)
	}

	result := make([]Comment, 0, len(dbComments))
	for _, c := range dbComments {
		result = append(result, Comment{
			Id:           c.Id,
			UserId:       c.UserId,
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

				if err := qtx.DeleteAllFeelingsForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all feelings for user: %w", err)
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

				if err := qtx.DeleteAllViewsForVideo(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all views for video: %w", err)
				}

				if err := qtx.DeleteAllFeelingsByForId(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all feelings by for_id: %w", err)
				}

				if err := qtx.DeleteAllCommentsFor(ctx, payload.VideoId); err != nil {
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
