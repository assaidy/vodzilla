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
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
)

const Name = "video"

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
			fmt.Sprintf("%q event consumer", events.VideoUploadedEvent),
			service.videoUploadedEventConsumer,
			workers.WithNRuns(1),
			workers.WithNRetries(0),
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

type VideoStatus string

const (
	// do not put this into db; it's for application to know if video is ready or not
	// this status might change if we added or removed layers of processing to video
	VideoStatusReady = VideoStatusUploaded

	VideoStatusUploading VideoStatus = "uploading"
	VideoStatusUploaded  VideoStatus = "uploaded"
)

type CreateVideoParams struct {
	OwnerId     string
	Title       string
	Description string
}

func (me *Service) CreateVideo(ctx context.Context, params CreateVideoParams) (string, error) {
	videoId := ulid.Make().String()

	if err := me.queries.InsertVideo(ctx, queries.InsertVideoParams{
		Id:          videoId,
		OwnerId:     params.OwnerId,
		Title:       params.Title,
		Description: sql.NullString{Valid: params.Description != "", String: params.Description},
		Status:      string(VideoStatusUploading),
	}); err != nil {
		return "", fmt.Errorf("failed to insert video: %w", err)
	}

	return videoId, nil
}

func (me *Service) videoUploadedEventConsumer(ctx context.Context) error {
	sub := me.redis.Subscribe(ctx, events.VideoUploadedEvent)
	defer sub.Close()
	ch := sub.Channel()

	for {
		select {
		case message := <-ch:
			var payload events.VideoUploadedEventPayload
			if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
				me.logger.Error("failed to unmarshal event payload", "event", events.VideoUploadedEvent, "error", err)
				continue
			}

			if err := me.queries.UpdateVideoStatus(ctx, queries.UpdateVideoStatusParams{
				Id:     payload.VideoId,
				Status: string(VideoStatusUploaded),
			}); err != nil {
				me.logger.Error("failed to update video status", "error", err)
				continue
			}

		case <-ctx.Done():
			return nil
		}
	}
}

type Video struct {
	Id          string
	OwnerId     string
	Timestamp   time.Time
	Title       string
	Description string
}

func (me *Service) GetVideoById(ctx context.Context, id string) (*Video, error) {
	video, err := me.queries.GetVideoById(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || video.Status != string(VideoStatusReady) {
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

func (me *Service) GetAllUserVideos(ctx context.Context, id string) ([]Video, error) {
	videos, err := me.queries.GetVideosForUser(ctx, queries.GetVideosForUserParams{
		OwnerId: id,
		Status:  string(VideoStatusReady),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get videos by user id: %w", err)
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

func (me *Service) DoesVideoExist(ctx context.Context, id string) (bool, error) {
	ok, err := me.queries.CheckVideo(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to check video: %w", err)
	}

	return ok, nil
}

func (me *Service) IsInWatchLater(ctx context.Context, videoId, userId string) (bool, error) {
	return me.queries.CheckWatchLater(ctx, queries.CheckWatchLaterParams{
		VideoId: videoId,
		UserId:  userId,
	})
}

func (me *Service) AddVideoToWatchLater(ctx context.Context, videoId, userId string) error {
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

	if ok, err := qtx.CheckWatchLater(ctx, queries.CheckWatchLaterParams{
		VideoId: videoId,
		UserId:  userId,
	}); err != nil {
		return fmt.Errorf("failed to check watch later: %w", err)
	} else if ok {
		return ErrConflict
	}

	if err := qtx.InsertIntoWatchLater(ctx, queries.InsertIntoWatchLaterParams{
		VideoId: videoId,
		UserId:  userId,
	}); err != nil {
		return fmt.Errorf("failed to insert into watch later: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *Service) DeleteVideoFromWatchLater(ctx context.Context, videoId, userId string) error {
	if n, err := me.queries.DeleteFromWatchLater(ctx, queries.DeleteFromWatchLaterParams{
		VideoId: videoId,
		UserId:  userId,
	}); err != nil {
		return fmt.Errorf("failed to delete from watch later: %w", err)
	} else if n == 0 {
		return ErrVideoNotFound
	}

	return nil
}

func (me *Service) GetAllVideosInWatchLater(ctx context.Context, userId string) ([]Video, error) {
	videos, err := me.queries.GetVideosInWatchLater(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos in watch later: %w", err)
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

func (me *Service) CreatePlaylist(ctx context.Context, userId, playlistName string) (string, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckPlaylistByNameForUser(ctx, queries.CheckPlaylistByNameForUserParams{
		Name:    playlistName,
		OwnerId: userId,
	}); err != nil {
		return "", fmt.Errorf("failed to check playlist by name for user: %w", err)
	} else if ok {
		return "", ErrConflict
	}

	playlistId := ulid.Make().String()
	if err := qtx.InsertPlaylist(ctx, queries.InsertPlaylistParams{
		Id:      playlistId,
		Name:    playlistName,
		OwnerId: userId,
	}); err != nil {
		return "", fmt.Errorf("failed to insert playlist: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit tx: %w", err)
	}

	return playlistId, nil
}

func (me *Service) DeletePlaylist(ctx context.Context, userId, playlistId string) error {
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

func (me *Service) AddVideoToPlaylist(ctx context.Context, videoId, userId, playlistId string) error {
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
		return ErrConflict
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

func (me *Service) DeleteVideoFromPlaylist(ctx context.Context, videoId, userId, playlistId string) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

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
		return ErrVideoNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

type Playlist struct {
	Id          string
	Name        string
	VideosCount int
}

type PlaylistWithVideoStatus struct {
	Id       string
	Name     string
	HasVideo bool
}

func (me *Service) GetAllPlaylists(ctx context.Context, userId string) ([]Playlist, error) {
	playlists, err := me.queries.GetAllPlaylistsForUser(ctx, userId)
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

func (me *Service) GetAllPlaylistsWithVideoStatus(ctx context.Context, userId, videoId string) ([]PlaylistWithVideoStatus, error) {
	playlists, err := me.queries.GetAllPlaylistsForUser(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get all playlists for user: %w", err)
	}

	result := make([]PlaylistWithVideoStatus, 0, len(playlists))
	for _, p := range playlists {
		hasVideo, err := me.queries.CheckVideoInPlaylist(ctx, queries.CheckVideoInPlaylistParams{
			PlaylistId: p.Id,
			VideoId:    videoId,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check video in playlist: %w", err)
		}

		result = append(result, PlaylistWithVideoStatus{
			Id:       p.Id,
			Name:     p.Name,
			HasVideo: hasVideo,
		})
	}

	return result, nil
}

func (me *Service) GetPlaylist(ctx context.Context, userId, playlistId string) (*Playlist, error) {
	playlist, err := me.queries.GetPlaylistForUser(ctx, queries.GetPlaylistForUserParams{
		Id:      playlistId,
		OwnerId: userId,
	})
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

func (me *Service) GetAllVideosInPlaylist(ctx context.Context, userId, playlistId string) ([]Video, error) {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckPlaylistForUser(ctx, queries.CheckPlaylistForUserParams{
		Id:      playlistId,
		OwnerId: userId,
	}); err != nil {
		return nil, fmt.Errorf("failed to check playlist for user: %w", err)
	} else if !ok {
		return nil, ErrPlaylistNotFound
	}

	videos, err := qtx.GetAllVideosInPlaylist(ctx, playlistId)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos in playlist: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
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

func (me *Service) IsInPlaylist(ctx context.Context, videoId, playlistId string) (bool, error) {
	return me.queries.CheckVideoInPlaylist(ctx, queries.CheckVideoInPlaylistParams{
		VideoId:    videoId,
		PlaylistId: playlistId,
	})
}
