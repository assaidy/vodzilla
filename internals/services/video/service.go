package video

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
	"github.com/assaidy/vodzilla/internals/services/video/queries"
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
			fmt.Sprintf("%q event consumer", events.VideoIsReadyEvent),
			service.videoIsReadyEventConsumerJob,
			workers.WithRetryDelay(time.Second),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
		),
	)
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
			fmt.Sprintf("%q event consumer", events.UploadExpiredEvent),
			service.uploadExpiredEventConsumerJob,
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

type CreateVideoParams struct {
	OwnerId     uuid.UUID
	Title       string
	Description string
}

func (me *Service) CreateVideo(ctx context.Context, params CreateVideoParams) (uuid.UUID, error) {
	videoId := uuid.Must(uuid.NewV7())

	if err := me.queries.InsertVideo(ctx, queries.InsertVideoParams{
		Id:          videoId,
		OwnerId:     params.OwnerId,
		Title:       params.Title,
		Description: sql.NullString{Valid: params.Description != "", String: params.Description},
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert video: %w", err)
	}

	return videoId, nil
}

func (me *Service) videoIsReadyEventConsumerJob(ctx context.Context) error {
	sub := me.redis.Subscribe(ctx, events.VideoIsReadyEvent)
	defer sub.Close()
	ch := sub.Channel()

	for {
		select {
		case message := <-ch:
			var payload events.VideoIsReadyEventPayload
			if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
				return fmt.Errorf("failed to unmarshal %q event payload: %w", events.VideoIsReadyEvent, err)
			}

			if err := me.queries.MarkVideoAsPublished(ctx, payload.VideoId); err != nil {
				return fmt.Errorf("failed to update video status: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

type Video struct {
	Id          uuid.UUID
	OwnerId     uuid.UUID
	Timestamp   time.Time
	Title       string
	Description string
}

type WatchlaterVideo struct {
	Video
	WatchlaterId int64
}

type PlaylistVideo struct {
	Video
	PlaylistVideoId int64
}

func (me *Service) GetVideoById(ctx context.Context, id uuid.UUID) (*Video, error) {
	video, err := me.queries.GetVideoById(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVideoNotFound
		}
		return nil, fmt.Errorf("failed to get video by id: %w", err)
	}

	if !video.IsPublished {
		return nil, ErrVideoNotFound
	}

	return &Video{
		Id:          video.Id,
		OwnerId:     video.OwnerId,
		Timestamp:   video.CreatedAt,
		Title:       video.Title,
		Description: video.Description.String,
	}, nil
}

func (me *Service) GetVideosCountForUser(ctx context.Context, userId uuid.UUID) (int, error) {
	n, err := me.queries.GetPublishedVideosCountForUser(ctx, userId)
	if err != nil {
		return 0, fmt.Errorf("failed to get videos count for user: %w", err)
	}

	return int(n), nil
}

func (me *Service) GetVideosForUser(ctx context.Context, userId, lastVideoId uuid.UUID, limit int) ([]Video, error) {
	videos, err := me.queries.GetPublishedVideosForUser(ctx, queries.GetPublishedVideosForUserParams{
		UserId:      userId,
		LastVideoId: uuid.NullUUID{UUID: lastVideoId, Valid: lastVideoId != uuid.Nil},
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get published videos for user: %w", err)
	}

	result := make([]Video, 0, len(videos))
	for _, v := range videos {
		result = append(result, Video{
			Id:          v.Id,
			OwnerId:     v.OwnerId,
			Timestamp:   v.CreatedAt,
			Title:       v.Title,
			Description: v.Description.String,
		})
	}

	return result, nil
}

func (me *Service) GetVideosForMultipleUsers(ctx context.Context, userIds []uuid.UUID, lastVideoId uuid.UUID, limit int) ([]Video, error) {
	if len(userIds) == 0 {
		return nil, nil
	}

	videos, err := me.queries.GetPublishedVideosForMultipleUsers(ctx, queries.GetPublishedVideosForMultipleUsersParams{
		UserIds:     userIds,
		LastVideoId: uuid.NullUUID{UUID: lastVideoId, Valid: lastVideoId != uuid.Nil},
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get published videos for multiple users: %w", err)
	}

	result := make([]Video, 0, len(videos))
	for _, v := range videos {
		result = append(result, Video{
			Id:          v.Id,
			OwnerId:     v.OwnerId,
			Timestamp:   v.CreatedAt,
			Title:       v.Title,
			Description: v.Description.String,
		})
	}

	return result, nil
}

func (me *Service) DoesVideoExist(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := me.queries.CheckVideo(ctx, queries.CheckVideoParams{
		Id:          id,
		IsPublished: true,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check video: %w", err)
	}

	return ok, nil
}

func (me *Service) IsInWatchLater(ctx context.Context, videoId, userId uuid.UUID) (bool, error) {
	return me.queries.CheckVideoInWatchlaters(ctx, queries.CheckVideoInWatchlatersParams{
		VideoId: videoId,
		UserId:  userId,
	})
}

func (me *Service) AddVideoToWatchlater(ctx context.Context, videoId, userId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, queries.CheckVideoParams{
		Id:          videoId,
		IsPublished: true,
	}); err != nil {
		return fmt.Errorf("failed to check video: %w", err)
	} else if !ok {
		return ErrVideoNotFound
	}

	if ok, err := qtx.CheckVideoInWatchlaters(ctx, queries.CheckVideoInWatchlatersParams{
		VideoId: videoId,
		UserId:  userId,
	}); err != nil {
		return fmt.Errorf("failed to check watchlater: %w", err)
	} else if ok {
		return ErrWatchlaterConflict
	}

	if err := qtx.InsertIntoWatchlaters(ctx, queries.InsertIntoWatchlatersParams{
		VideoId: videoId,
		UserId:  userId,
	}); err != nil {
		return fmt.Errorf("failed to insert into watchlater: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *Service) DeleteVideoFromWatchlater(ctx context.Context, videoId, userId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, queries.CheckVideoParams{
		Id:          videoId,
		IsPublished: true,
	}); err != nil {
		return fmt.Errorf("failed to check video: %w", err)
	} else if !ok {
		return ErrVideoNotFound
	}

	if n, err := qtx.DeleteFromWatchlaters(ctx, queries.DeleteFromWatchlatersParams{
		VideoId: videoId,
		UserId:  userId,
	}); err != nil {
		return fmt.Errorf("failed to delete from watchlater: %w", err)
	} else if n == 0 {
		return ErrWatchlaterVideoNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *Service) GetVideosInWatchlater(ctx context.Context, userId uuid.UUID, lastId int64, limit int) ([]WatchlaterVideo, error) {
	rows, err := me.queries.GetVideosInWatchlaters(ctx, queries.GetVideosInWatchlatersParams{
		UserId:           userId,
		LastWatchlaterId: sql.NullInt64{Int64: lastId, Valid: lastId != 0},
		Limit:            int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get videos in watchlater: %w", err)
	}

	result := make([]WatchlaterVideo, 0, len(rows))
	for _, row := range rows {
		result = append(result, WatchlaterVideo{
			Video: Video{
				Id:          row.Id,
				OwnerId:     row.OwnerId,
				Timestamp:   row.CreatedAt,
				Title:       row.Title,
				Description: row.Description.String,
			},
			WatchlaterId: row.WatchlaterId,
		})
	}

	return result, nil
}

func (me *Service) GetVideoOwner(ctx context.Context, videoId uuid.UUID) (uuid.UUID, error) {
	ownerId, err := me.queries.GetVideoOwner(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrVideoNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get video owner: %w", err)
	}

	return ownerId, nil
}

func (me *Service) CreatePlaylist(ctx context.Context, userId uuid.UUID, playlistName string) (uuid.UUID, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckPlaylistByNameForUser(ctx, queries.CheckPlaylistByNameForUserParams{
		Name:    playlistName,
		OwnerId: userId,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to check playlist by name for user: %w", err)
	} else if ok {
		return uuid.Nil, ErrPlaylistNameConflict
	}

	playlistId := uuid.Must(uuid.NewV7())
	if err := qtx.InsertPlaylist(ctx, queries.InsertPlaylistParams{
		Id:      playlistId,
		Name:    playlistName,
		OwnerId: userId,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert playlist: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return playlistId, nil
}

func (me *Service) DeletePlaylist(ctx context.Context, userId, playlistId uuid.UUID) error {
	if n, err := me.queries.DeletePlaylist(ctx, queries.DeletePlaylistParams{
		Id:      playlistId,
		OwnerId: userId,
	}); err != nil {
		return fmt.Errorf("failed to delete playlist: %w", err)
	} else if n == 0 {
		return ErrPlaylistNotFound
	}

	return nil
}

func (me *Service) AddVideoToPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, queries.CheckVideoParams{
		Id:          videoId,
		IsPublished: true,
	}); err != nil {
		return fmt.Errorf("failed to check video: %w", err)
	} else if !ok {
		return ErrVideoNotFound
	}

	if ok, err := qtx.CheckPlaylist(ctx, playlistId); err != nil {
		return fmt.Errorf("failed to check playlist: %w", err)
	} else if !ok {
		return ErrPlaylistNotFound
	}

	if ok, err := qtx.CheckVideoInPlaylist(ctx, queries.CheckVideoInPlaylistParams{
		VideoId:    videoId,
		PlaylistId: playlistId,
	}); err != nil {
		return fmt.Errorf("failed to check video in playlist: %w", err)
	} else if ok {
		return ErrPlaylistVideoConflict
	}

	if err := qtx.InsertIntoPlaylist(ctx, queries.InsertIntoPlaylistParams{
		PlaylistId: playlistId,
		VideoId:    videoId,
	}); err != nil {
		return fmt.Errorf("failed to insert into playlist: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *Service) DeleteVideoFromPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, queries.CheckVideoParams{
		Id:          videoId,
		IsPublished: true,
	}); err != nil {
		return fmt.Errorf("failed to check video: %w", err)
	} else if !ok {
		return ErrVideoNotFound
	}

	if ok, err := qtx.CheckPlaylistForUser(ctx, queries.CheckPlaylistForUserParams{
		Id:      playlistId,
		OwnerId: userId,
	}); err != nil {
		return fmt.Errorf("failed to check playlist for user: %w", err)
	} else if !ok {
		return ErrPlaylistNotFound
	}

	if n, err := qtx.DeleteVideoFromPlaylist(ctx, queries.DeleteVideoFromPlaylistParams{
		PlaylistId: playlistId,
		VideoId:    videoId,
	}); err != nil {
		return fmt.Errorf("failed to delete video from playlist: %w", err)
	} else if n == 0 {
		return ErrPlaylistVideoNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

type Playlist struct {
	Id          uuid.UUID
	Name        string
	VideosCount int
}

type PlaylistWithVideoStatus struct {
	Playlist
	HasVideo bool
}

func (me *Service) GetPlaylists(ctx context.Context, userId, lastPlaylistId uuid.UUID, limit int) ([]Playlist, error) {
	playlists, err := me.queries.GetPlaylistsForUser(ctx, queries.GetPlaylistsForUserParams{
		UserId:         userId,
		LastPlaylistId: uuid.NullUUID{UUID: lastPlaylistId, Valid: lastPlaylistId != uuid.Nil},
		Limit:          int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all playlists for user: %w", err)
	}

	result := make([]Playlist, 0, len(playlists))
	for _, p := range playlists {
		result = append(result, Playlist{
			Id:          p.Id,
			Name:        p.Name,
			VideosCount: int(p.VideosCount),
		})
	}

	return result, nil
}

func (me *Service) GetPlaylistsWithVideoStatus(ctx context.Context, userId, videoId, lastPlaylistId uuid.UUID, limit int) ([]PlaylistWithVideoStatus, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, queries.CheckVideoParams{
		Id:          videoId,
		IsPublished: true,
	}); err != nil {
		return nil, fmt.Errorf("failed to check video: %w", err)
	} else if !ok {
		return nil, ErrVideoNotFound
	}

	rows, err := qtx.GetPlaylistsWithVideoStatusForUser(ctx, queries.GetPlaylistsWithVideoStatusForUserParams{
		UserId:         userId,
		VideoId:        videoId,
		LastPlaylistId: uuid.NullUUID{UUID: lastPlaylistId, Valid: lastPlaylistId != uuid.Nil},
		Limit:          int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get playlists with video status: %w", err)
	}

	result := make([]PlaylistWithVideoStatus, 0, len(rows))
	for _, row := range rows {
		result = append(result, PlaylistWithVideoStatus{
			Playlist: Playlist{
				Id:          row.Id,
				Name:        row.Name,
				VideosCount: int(row.VideosCount),
			},
			HasVideo: row.HasVideo,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return result, nil
}

func (me *Service) GetPlaylist(ctx context.Context, playlistId uuid.UUID) (*Playlist, error) {
	playlist, err := me.queries.GetPlaylist(ctx, playlistId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	return &Playlist{
		Id:          playlist.Id,
		Name:        playlist.Name,
		VideosCount: int(playlist.VideosCount),
	}, nil
}

func (me *Service) GetVideosInPlaylist(ctx context.Context, playlistId uuid.UUID, lastId int64, limit int) ([]PlaylistVideo, error) {
	rows, err := me.queries.GetVideosInPlaylist(ctx, queries.GetVideosInPlaylistParams{
		PlaylistId: playlistId,
		LastId:     sql.NullInt64{Int64: lastId, Valid: lastId != 0},
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get videos in playlist: %w", err)
	}

	result := make([]PlaylistVideo, 0, len(rows))
	for _, row := range rows {
		result = append(result, PlaylistVideo{
			Video: Video{
				Id:          row.Id,
				OwnerId:     row.OwnerId,
				Timestamp:   row.CreatedAt,
				Title:       row.Title,
				Description: row.Description.String,
			},
			PlaylistVideoId: row.PlaylistVideoId,
		})
	}

	return result, nil
}

func (me *Service) IsInPlaylist(ctx context.Context, videoId, playlistId uuid.UUID) (bool, error) {
	return me.queries.CheckVideoInPlaylist(ctx, queries.CheckVideoInPlaylistParams{
		VideoId:    videoId,
		PlaylistId: playlistId,
	})
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
				return fmt.Errorf("failed to unmarshal %q event payload: %w", events.VideoIsReadyEvent, err)
			}

			var deletedVideoIds []uuid.UUID

			if err := func() error {
				tx, err := me.db.BeginTx(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to begin tx: %w", err)
				}
				defer tx.Rollback()
				qtx := me.queries.WithTx(tx)

				if deletedVideoIds, err = qtx.DeleteAllVideosForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to deletd all videos for user: %w", err)
				}

				if err := qtx.DeleteAllWatchlatersForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all watchlaters for user: %w", err)
				}

				if err := qtx.DeleteAllPlaylistsForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all playlists for user: %w", err)
				}

				if err := tx.Commit(); err != nil {
					return fmt.Errorf("failed to commit tx: %w", err)
				}

				return nil
			}(); err != nil {
				return err
			}

			for _, id := range deletedVideoIds {
				payload, err := json.Marshal(events.VideoDeletedEventPayload{VideoId: id})
				if err != nil {
					return fmt.Errorf("failed to marshal %q event payload: %w", events.VideoDeletedEvent, err)
				}
				if err := me.redis.Publish(ctx, events.VideoDeletedEvent, payload).Err(); err != nil {
					return fmt.Errorf("failed to publish %q event: %w", events.VideoDeletedEvent, err)
				}
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (me *Service) DeleteVideo(ctx context.Context, videoId, userId uuid.UUID) error {
	if n, err := me.queries.DeleteVideoByIdForUser(ctx, queries.DeleteVideoByIdForUserParams{
		Id:          videoId,
		OwnerId:     userId,
		IsPublished: true, // only pubished videos are visible to users
	}); err != nil {
		return fmt.Errorf("failed to delete video by id for user: %w", err)
	} else if n == 0 {
		return ErrVideoNotFound
	}

	payload, err := json.Marshal(events.VideoDeletedEventPayload{VideoId: videoId})
	if err != nil {
		return fmt.Errorf("failed to marshal %q event payload: %w", events.VideoDeletedEvent, err)
	}
	if err := me.redis.Publish(ctx, events.VideoDeletedEvent, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish %q event: %w", events.VideoDeletedEvent, err)
	}

	return nil
}

func (me *Service) uploadExpiredEventConsumerJob(ctx context.Context) error {
	sub := me.redis.Subscribe(ctx, events.UploadExpiredEvent)
	defer sub.Close()
	ch := sub.Channel()

	for {
		select {
		case message := <-ch:
			var payload events.UploadExpiredEventPayload
			if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
				return fmt.Errorf("failed to unmarshal %q event payload: %w", events.UploadExpiredEvent, err)
			}

			// The video is not in ready status yet, so no other service except media has reference to it.
			// Media service already deleted it references before publishing the event.
			if err := me.queries.DeleteVideoById(ctx, payload.VideoId); err != nil {
				return fmt.Errorf("failed to delete video by id: %w", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
