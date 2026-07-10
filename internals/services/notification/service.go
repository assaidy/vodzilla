package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/notification/queries"
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

type NotificationKind string

const (
	NotificationFollow  NotificationKind = "follow"
	NotificationFeeling NotificationKind = "feeling"
	NotificationComment NotificationKind = "comment"
)

func (k NotificationKind) isValid() bool {
	return k == NotificationFollow || k == NotificationFeeling || k == NotificationComment
}

type payload interface {
	isPayload()
}

type payloadMaker struct{}

func (me payloadMaker) isPayload() {}

type FollowPayload struct {
	payloadMaker
	UserId uuid.UUID `json:"user_id"`
}

type FeelingPayload struct {
	payloadMaker
	UserId  uuid.UUID `json:"user_id"`
	VideoId uuid.UUID `json:"video_id"`
	Feeling string    `json:"feeling"`
}

type VideoCommentPayload struct {
	payloadMaker
	UserId    uuid.UUID `json:"user_id"`
	VideoId   uuid.UUID `json:"video_id"`
	CommentId uuid.UUID `json:"comment_id"`
}

type CommentReplyPayload struct {
	payloadMaker
	UserId    uuid.UUID `json:"user_id"`
	CommentId uuid.UUID `json:"comment_id"`
	ReplyId   uuid.UUID `json:"reply_id"`
}

type Notification struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Kind      NotificationKind
	Payload   json.RawMessage
	CreatedAt time.Time
	IsRead    bool
}

func (me *Service) AddNotification(ctx context.Context, userId uuid.UUID, kind NotificationKind, payload payload) error {
	if !kind.isValid() {
		return fmt.Errorf("invalid notification kind: %q", kind)
	}

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	if err := me.queries.InsertNotification(ctx, queries.InsertNotificationParams{
		Id:      uuid.Must(uuid.NewV7()),
		UserId:  userId,
		Kind:    string(kind),
		Payload: encodedPayload,
	}); err != nil {
		return fmt.Errorf("failed to insert notification: %w", err)
	}

	return nil
}

func (me *Service) MarkNotificationAsRead(ctx context.Context, userId, notificationId uuid.UUID) error {
	if n, err := me.queries.MarkNotificationAsRead(ctx, queries.MarkNotificationAsReadParams{
		Id:     notificationId,
		UserId: userId,
	}); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	} else if n == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

func (me *Service) GetNotifications(ctx context.Context, userId, lastNotificationId uuid.UUID, limit int) ([]Notification, error) {
	dbNotifications, err := me.queries.GetNotifications(ctx, queries.GetNotificationsParams{
		UserId:             userId,
		LastNotificationId: uuid.NullUUID{UUID: lastNotificationId, Valid: lastNotificationId != uuid.Nil},
		Limit:              int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	result := make([]Notification, 0, len(dbNotifications))
	for _, n := range dbNotifications {
		result = append(result, Notification{
			Id:        n.Id,
			UserId:    n.UserId,
			Kind:      NotificationKind(n.Kind),
			Payload:   json.RawMessage(n.Payload),
			CreatedAt: n.CreatedAt,
			IsRead:    n.IsRead,
		})
	}

	return result, nil
}

func (me *Service) GetUnreadNotificationsCount(ctx context.Context, userId uuid.UUID) (int, error) {
	count, err := me.queries.GetUnreadNotificationsCount(ctx, userId)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread notifications count: %w", err)
	}

	return int(count), nil
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

			if err := me.queries.DeleteAllNotificationsForUser(ctx, payload.UserId); err != nil {
				return fmt.Errorf("failed to delete all notifications for user: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
