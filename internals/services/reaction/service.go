package reaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/reaction/queries"
	"github.com/assaidy/workers"
	"github.com/assaidy/workers/lock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	services.Service
	AddView(ctx context.Context, userId uuid.UUID, targetId uuid.UUID, targetKind ViewTargetKind) error
	GetViewsCount(ctx context.Context, targetId uuid.UUID) (int, error)
	AddFeeling(ctx context.Context, userId, targetId uuid.UUID, targetKind FeelingTargetKind, kind FeelingKind) error
	DeleteFeeling(ctx context.Context, userId, targetId uuid.UUID) error
	GetFeelingCounts(ctx context.Context, targetId uuid.UUID) (*FeelingCounts, error)
	GetUserFeeling(ctx context.Context, userId, targetId uuid.UUID) (FeelingKind, error)
	DoesCommentExist(ctx context.Context, commentId uuid.UUID) (bool, error)
	GetCommentById(ctx context.Context, commentId uuid.UUID) (*Comment, error)
	CreateComment(ctx context.Context, userId, targetId uuid.UUID, targetKind CommentTargetKind, content string) (uuid.UUID, error)
	EditComment(ctx context.Context, userId, commentId uuid.UUID, newContent string) error
	DeleteComment(ctx context.Context, userId, commentId uuid.UUID) error
	GetCommentsCount(ctx context.Context, targetId uuid.UUID) (int, error)
	GetComments(ctx context.Context, targetId uuid.UUID, lastCommentId uuid.UUID, limit int) ([]Comment, error)
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
		db:      db,
		queries: queries.New(db),
		redis:   redis,
		logger:  logger,
		workerManager: workers.NewWorkerManager(
			workers.WithLogger(logger),
			workers.WithLockGenerator(lock.NewRedisGenerator(redis)),
		),
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

type ViewTargetKind string

const (
	ViewTargetVideo    ViewTargetKind = "video"
	ViewTargetPlaylist ViewTargetKind = "playlist"
)

func (k ViewTargetKind) isValid() bool {
	return k == ViewTargetVideo || k == ViewTargetPlaylist
}

func (me *impl) AddView(ctx context.Context, userId uuid.UUID, targetId uuid.UUID, targetKind ViewTargetKind) error {
	if !targetKind.isValid() {
		return fmt.Errorf("invalid view kind: %q", targetKind)
	}

	if err := me.queries.InsertView(ctx, queries.InsertViewParams{
		TargetId:   targetId,
		UserId:     userId,
		TargetKind: string(targetKind),
	}); err != nil {
		return fmt.Errorf("failed to insert view: %w", err)
	}

	return nil
}

func (me *impl) GetViewsCount(ctx context.Context, targetId uuid.UUID) (int, error) {
	count, err := me.queries.GetViewsCount(ctx, targetId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to get views count: %w", err)
	}

	return int(count), nil
}

type FeelingTargetKind string

const (
	FeelingTargetVideo   FeelingTargetKind = "video"
	FeelingTargetComment FeelingTargetKind = "comment"
)

func (k FeelingTargetKind) isValid() bool {
	return k == FeelingTargetVideo || k == FeelingTargetComment
}

type FeelingKind string

const (
	FeelingLike    FeelingKind = "like"
	FeelingDislike FeelingKind = "dislike"
)

func (k FeelingKind) isValid() bool {
	return k == FeelingLike || k == FeelingDislike
}

func (me *impl) AddFeeling(ctx context.Context, userId, targetId uuid.UUID, targetKind FeelingTargetKind, kind FeelingKind) error {
	if !targetKind.isValid() {
		return fmt.Errorf("invalid feeling target kind: %q", targetKind)
	}
	if !kind.isValid() {
		return fmt.Errorf("invalid feeling kind: %q", kind)
	}

	if err := me.queries.UpsertFeeling(ctx, queries.UpsertFeelingParams{
		TargetId:   targetId,
		UserId:     userId,
		TargetKind: string(targetKind),
		Kind:       string(kind),
	}); err != nil {
		return fmt.Errorf("failed to upsert feeling (target: %s, kind: %s): %w", targetKind, kind, err)
	}

	return nil
}

func (me *impl) DeleteFeeling(ctx context.Context, userId, targetId uuid.UUID) error {
	n, err := me.queries.DeleteFeeling(ctx, queries.DeleteFeelingParams{
		TargetId: targetId,
		UserId:   userId,
	})
	if err != nil {
		return fmt.Errorf("failed to delete feeling: %w", err)
	} else if n == 0 {
		return ErrFeelingNotFound
	}

	return nil
}

type FeelingCounts struct {
	Likes    int
	Dislikes int
}

func (me *impl) GetFeelingCounts(ctx context.Context, targetId uuid.UUID) (*FeelingCounts, error) {
	counts, err := me.queries.GetFeelingCounts(ctx, targetId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get feeling counts: %w", err)
	}

	return &FeelingCounts{
		Likes:    int(counts.Likes),
		Dislikes: int(counts.Dislikes),
	}, nil
}

func (me *impl) GetUserFeeling(ctx context.Context, userId, targetId uuid.UUID) (FeelingKind, error) {
	kind, err := me.queries.GetUserFeeling(ctx, queries.GetUserFeelingParams{
		TargetId: targetId,
		UserId:   userId,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("failed to get user feeling: %w", err)
	}

	return FeelingKind(kind), nil
}

type CommentTargetKind string

const (
	CommentTargetVideo   CommentTargetKind = "video"
	CommentTargetComment CommentTargetKind = "comment"
)

func (me CommentTargetKind) isValid() bool {
	return me == CommentTargetVideo || me == CommentTargetComment
}

func (me *impl) DoesCommentExist(ctx context.Context, commentId uuid.UUID) (bool, error) {
	return me.queries.CheckComment(ctx, commentId)
}

func (me *impl) GetCommentById(ctx context.Context, commentId uuid.UUID) (*Comment, error) {
	dbComment, err := me.queries.GetCommentById(ctx, commentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}
	return &Comment{
		Id:        dbComment.Id,
		UserId:    dbComment.UserId,
		Content:   dbComment.Content,
		CreatedAt: dbComment.CreatedAt,
	}, nil
}

func (me *impl) CreateComment(ctx context.Context, userId, targetId uuid.UUID, targetKind CommentTargetKind, content string) (uuid.UUID, error) {
	if !targetKind.isValid() {
		return uuid.Nil, fmt.Errorf("invalid comment target kind: %q", targetKind)
	}

	commentId := uuid.Must(uuid.NewV7())
	if err := me.queries.InsertComment(ctx, queries.InsertCommentParams{
		Id:         commentId,
		TargetId:   targetId,
		TargetKind: string(targetKind),
		UserId:     userId,
		Content:    content,
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

	if err := qtx.DeleteAllCommentsForTarget(ctx, commentId); err != nil {
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

func (me *impl) GetCommentsCount(ctx context.Context, targetId uuid.UUID) (int, error) {
	count, err := me.queries.GetCommentsCount(ctx, targetId)
	if err != nil {
		return 0, fmt.Errorf("failed to get comments count: %w", err)
	}
	return int(count), nil
}

func (me *impl) GetComments(ctx context.Context, targetId, lastCommentId uuid.UUID, limit int) ([]Comment, error) {
	dbComments, err := me.queries.GetComments(ctx, queries.GetCommentsParams{
		TargetId:      targetId,
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

func (me *impl) userDeletedEventConsumerJob(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var payload events.UserDeletedEventPayload
			if ok, err := events.Consume(ctx, me.redis, events.UserDeletedEvent, &payload); err != nil {
				return fmt.Errorf("failed to consume %q event: %w", events.UserDeletedEvent, err)
			} else if !ok {
				continue
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
		}
	}
}

func (me *impl) videoDeletedEventConsumerJob(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var payload events.VideoDeletedEventPayload
			if ok, err := events.Consume(ctx, me.redis, events.VideoDeletedEvent, &payload); err != nil {
				return fmt.Errorf("failed to consume %q event: %w", events.VideoDeletedEvent, err)
			} else if !ok {
				continue
			}

			if err := func() error {
				tx, err := me.db.BeginTx(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to begin tx: %w", err)
				}
				defer tx.Rollback()
				qtx := me.queries.WithTx(tx)

				if err := qtx.DeleteAllViewsForTarget(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all views for video: %w", err)
				}

				if err := qtx.DeleteAllFeelingsForTarget(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all feelings for target: %w", err)
				}

				if err := qtx.DeleteAllCommentsForTarget(ctx, payload.VideoId); err != nil {
					return fmt.Errorf("failed to delete all comments for video: %w", err)
				}

				if err := tx.Commit(); err != nil {
					return fmt.Errorf("failed to commit tx: %w", err)
				}

				return nil
			}(); err != nil {
				return err
			}
		}
	}
}
