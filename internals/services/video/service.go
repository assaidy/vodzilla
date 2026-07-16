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

type Service interface {
	services.Service
	CreateVideo(ctx context.Context, params CreateVideoParams) (uuid.UUID, error)
	ActivateVideo(ctx context.Context, videoId uuid.UUID) error
	GetVideoById(ctx context.Context, id uuid.UUID) (*Video, error)
	GetVideosCountForUser(ctx context.Context, userId uuid.UUID) (int, error)
	GetVideosForUser(ctx context.Context, userId, lastVideoId uuid.UUID, limit int) ([]Video, error)
	GetVideosForMultipleUsers(ctx context.Context, userIds []uuid.UUID, lastVideoId uuid.UUID, limit int) ([]Video, error)
	DoesVideoExist(ctx context.Context, id uuid.UUID) (bool, error)
	IsInWatchLater(ctx context.Context, videoId, userId uuid.UUID) (bool, error)
	AddVideoToWatchlater(ctx context.Context, videoId, userId uuid.UUID) error
	DeleteVideoFromWatchlater(ctx context.Context, videoId, userId uuid.UUID) error
	GetVideosInWatchlater(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]WatchlaterVideo, error)
	GetVideoOwner(ctx context.Context, videoId uuid.UUID) (uuid.UUID, error)
	CreatePlaylist(ctx context.Context, userId uuid.UUID, playlistName string) (uuid.UUID, error)
	DeletePlaylist(ctx context.Context, userId, playlistId uuid.UUID) error
	RenamePlaylist(ctx context.Context, userId, playlistId uuid.UUID, name string) error
	AddVideoToPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error
	DeleteVideoFromPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error
	GetPlaylists(ctx context.Context, userId, lastPlaylistId uuid.UUID, limit int) ([]Playlist, error)
	GetPlaylistsWithVideoStatus(ctx context.Context, userId, videoId, lastPlaylistId uuid.UUID, limit int) ([]PlaylistWithVideoStatus, error)
	GetPlaylist(ctx context.Context, playlistId uuid.UUID) (*Playlist, error)
	GetVideosInPlaylist(ctx context.Context, playlistId uuid.UUID, lastId int, limit int) ([]PlaylistVideo, error)
	IsInPlaylist(ctx context.Context, videoId, playlistId uuid.UUID) (bool, error)
	DeleteVideo(ctx context.Context, videoId, userId uuid.UUID) error
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
			"pending videos cleanup",
			service.pendingVideosCleanupJob,
			workers.WithTick(1*time.Hour),
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

type CreateVideoParams struct {
	OwnerId     uuid.UUID
	Title       string
	Description string
}

func (me *impl) CreateVideo(ctx context.Context, params CreateVideoParams) (uuid.UUID, error) {
	videoId := uuid.Must(uuid.NewV7())

	if err := me.queries.InsertPendingVideo(ctx, queries.InsertPendingVideoParams{
		Id:          videoId,
		OwnerId:     params.OwnerId,
		Title:       params.Title,
		Description: sql.NullString{Valid: params.Description != "", String: params.Description},
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert pending video: %w", err)
	}

	return videoId, nil
}

func (me *impl) ActivateVideo(ctx context.Context, videoId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	pending, err := qtx.GetPendingVideoById(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrVideoNotFound
		}
		return fmt.Errorf("failed to get pending video: %w", err)
	}

	if _, err := qtx.DeletePendingVideoById(ctx, videoId); err != nil {
		return fmt.Errorf("failed to delete pending video: %w", err)
	}

	if err := qtx.InsertVideo(ctx, queries.InsertVideoParams{
		Id:          videoId,
		OwnerId:     pending.OwnerId,
		Title:       pending.Title,
		Description: pending.Description,
	}); err != nil {
		return fmt.Errorf("failed to insert video: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
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
	WatchlaterVideoId int
}

type PlaylistVideo struct {
	Video
	PlaylistVideoId int
}

func (me *impl) GetVideoById(ctx context.Context, id uuid.UUID) (*Video, error) {
	video, err := me.queries.GetVideoById(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVideoNotFound
		}
		return nil, fmt.Errorf("failed to get video by id: %w", err)
	}

	return &Video{
		Id:          video.Id,
		OwnerId:     video.OwnerId,
		Timestamp:   video.CreatedAt,
		Title:       video.Title,
		Description: video.Description.String,
	}, nil
}

func (me *impl) GetVideosCountForUser(ctx context.Context, userId uuid.UUID) (int, error) {
	n, err := me.queries.GetVideosCountForUser(ctx, userId)
	if err != nil {
		return 0, fmt.Errorf("failed to get videos count for user: %w", err)
	}

	return int(n), nil
}

func (me *impl) GetVideosForUser(ctx context.Context, userId, lastVideoId uuid.UUID, limit int) ([]Video, error) {
	videos, err := me.queries.GetVideosForUser(ctx, queries.GetVideosForUserParams{
		UserId:      userId,
		LastVideoId: uuid.NullUUID{UUID: lastVideoId, Valid: lastVideoId != uuid.Nil},
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get videos for user: %w", err)
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

func (me *impl) GetVideosForMultipleUsers(ctx context.Context, userIds []uuid.UUID, lastVideoId uuid.UUID, limit int) ([]Video, error) {
	if len(userIds) == 0 {
		return nil, nil
	}

	videos, err := me.queries.GetVideosForMultipleUsers(ctx, queries.GetVideosForMultipleUsersParams{
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

func (me *impl) DoesVideoExist(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := me.queries.CheckVideo(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to check video: %w", err)
	}

	return ok, nil
}

func (me *impl) IsInWatchLater(ctx context.Context, videoId, userId uuid.UUID) (bool, error) {
	return me.queries.CheckVideoInWatchlaters(ctx, queries.CheckVideoInWatchlatersParams{
		VideoId: videoId,
		UserId:  userId,
	})
}

func (me *impl) AddVideoToWatchlater(ctx context.Context, videoId, userId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, videoId); err != nil {
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

func (me *impl) DeleteVideoFromWatchlater(ctx context.Context, videoId, userId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, videoId); err != nil {
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

func (me *impl) GetVideosInWatchlater(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]WatchlaterVideo, error) {
	rows, err := me.queries.GetVideosInWatchlaters(ctx, queries.GetVideosInWatchlatersParams{
		UserId:           userId,
		LastWatchlaterId: sql.NullInt64{Int64: int64(lastId), Valid: lastId != 0},
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
			WatchlaterVideoId: int(row.WatchlaterId),
		})
	}

	return result, nil
}

func (me *impl) GetVideoOwner(ctx context.Context, videoId uuid.UUID) (uuid.UUID, error) {
	ownerId, err := me.queries.GetVideoOwner(ctx, videoId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrVideoNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get video owner: %w", err)
	}

	return ownerId, nil
}

func (me *impl) CreatePlaylist(ctx context.Context, userId uuid.UUID, playlistName string) (uuid.UUID, error) {
	playlistId := uuid.Must(uuid.NewV7())
	if err := me.queries.InsertPlaylist(ctx, queries.InsertPlaylistParams{
		Id:      playlistId,
		Name:    playlistName,
		OwnerId: userId,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert playlist: %w", err)
	}

	return playlistId, nil
}

func (me *impl) DeletePlaylist(ctx context.Context, userId, playlistId uuid.UUID) error {
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

func (me *impl) RenamePlaylist(ctx context.Context, userId, playlistId uuid.UUID, name string) error {
	if n, err := me.queries.UpdatePlaylistName(ctx, queries.UpdatePlaylistNameParams{
		Id:      playlistId,
		Name:    name,
		OwnerId: userId,
	}); err != nil {
		return fmt.Errorf("failed to rename playlist: %w", err)
	} else if n == 0 {
		return ErrPlaylistNotFound
	}

	return nil
}

func (me *impl) AddVideoToPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, videoId); err != nil {
		return fmt.Errorf("failed to check video: %w", err)
	} else if !ok {
		return ErrVideoNotFound
	}

	if ok, err := qtx.CheckPlaylistForUser(ctx, queries.CheckPlaylistForUserParams{
		Id:      playlistId,
		OwnerId: userId,
	}); err != nil {
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

func (me *impl) DeleteVideoFromPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, videoId); err != nil {
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

func (me *impl) GetPlaylists(ctx context.Context, userId, lastPlaylistId uuid.UUID, limit int) ([]Playlist, error) {
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

func (me *impl) GetPlaylistsWithVideoStatus(ctx context.Context, userId, videoId, lastPlaylistId uuid.UUID, limit int) ([]PlaylistWithVideoStatus, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckVideo(ctx, videoId); err != nil {
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

func (me *impl) GetPlaylist(ctx context.Context, playlistId uuid.UUID) (*Playlist, error) {
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

func (me *impl) GetVideosInPlaylist(ctx context.Context, playlistId uuid.UUID, lastId int, limit int) ([]PlaylistVideo, error) {
	if ok, err := me.queries.CheckPlaylist(ctx, playlistId); err != nil {
		return nil, fmt.Errorf("failed to check playlist: %w", err)
	} else if !ok {
		return nil, ErrPlaylistNotFound
	}

	rows, err := me.queries.GetVideosInPlaylist(ctx, queries.GetVideosInPlaylistParams{
		PlaylistId: playlistId,
		LastId:     sql.NullInt64{Int64: int64(lastId), Valid: lastId != 0},
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
			PlaylistVideoId: int(row.PlaylistVideoId),
		})
	}

	return result, nil
}

func (me *impl) IsInPlaylist(ctx context.Context, videoId, playlistId uuid.UUID) (bool, error) {
	return me.queries.CheckVideoInPlaylist(ctx, queries.CheckVideoInPlaylistParams{
		VideoId:    videoId,
		PlaylistId: playlistId,
	})
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

			var deletedVideoIds []uuid.UUID

			if err := func() error {
				tx, err := me.db.BeginTx(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to begin tx: %w", err)
				}
				defer tx.Rollback()
				qtx := me.queries.WithTx(tx)

				if deletedVideoIds, err = qtx.DeleteAllVideosForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all videos for user: %w", err)
				}

				if _, err := qtx.DeleteAllPendingVideosForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all pending videos for user: %w", err)
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

func (me *impl) pendingVideosCleanupJob(ctx context.Context) error {
	me.logger.Info("cleaning up expired pending videos")
	if err := me.queries.DeleteExpiredPendingVideos(ctx); err != nil {
		return fmt.Errorf("failed to delete expired pending videos: %w", err)
	}
	return nil
}

func (me *impl) DeleteVideo(ctx context.Context, videoId, userId uuid.UUID) error {
	if n, err := me.queries.DeleteVideoByIdForUser(ctx, queries.DeleteVideoByIdForUserParams{
		Id:      videoId,
		OwnerId: userId,
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
