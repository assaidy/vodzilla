package video

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/assaidy/vodzilla/internals/events"
	"github.com/assaidy/vodzilla/internals/services"
	"github.com/assaidy/vodzilla/internals/services/video/queries"
	"github.com/assaidy/workers"
	"github.com/assaidy/workers/lock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	services.Service

	// CreateVideo creates a new pending video for the given user and returns the new video id.
	CreateVideo(ctx context.Context, userId uuid.UUID, title, description string) (uuid.UUID, error)

	// ActivateVideo activates the pending video with the given id, making it publicly visible.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no pending video exists with the given id
	ActivateVideo(ctx context.Context, videoId uuid.UUID) error

	// GetVideoById returns the video with the given id.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists with the given id
	GetVideoById(ctx context.Context, id uuid.UUID) (*Video, error)

	// GetVideosCountForUser returns the number of videos of the given user.
	GetVideosCountForUser(ctx context.Context, userId uuid.UUID) (int, error)

	// GetVideosForUser returns the videos of the given user,
	// paginated by the given last video id and limit.
	GetVideosForUser(ctx context.Context, userId, lastVideoId uuid.UUID, limit int) ([]Video, error)

	// GetVideosForMultipleUsers returns the videos of the given users,
	// paginated by the given last video id and limit.
	GetVideosForMultipleUsers(ctx context.Context, userIds []uuid.UUID, lastVideoId uuid.UUID, limit int) ([]Video, error)

	// DoesVideoExist reports whether a video with the given id exists.
	DoesVideoExist(ctx context.Context, id uuid.UUID) (bool, error)

	// IsInWatchLater reports whether the video with the given id is in the given user's watch later.
	IsInWatchLater(ctx context.Context, videoId, userId uuid.UUID) (bool, error)

	// AddVideoToWatchlater adds the video with the given id to the given user's watch later.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists with the given id
	//   - [ErrWatchlaterConflict] - the video is already in the user's watch later
	AddVideoToWatchlater(ctx context.Context, videoId, userId uuid.UUID) error

	// DeleteVideoFromWatchlater removes the video with the given id from the given user's watch later.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists with the given id
	//   - [ErrWatchlaterVideoNotFound] - the video is not in the user's watch later
	DeleteVideoFromWatchlater(ctx context.Context, videoId, userId uuid.UUID) error

	// GetVideosInWatchlater returns the videos in the given user's watch later,
	// paginated by the given last id and limit.
	GetVideosInWatchlater(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]WatchlaterVideo, error)

	// GetVideoOwner returns the id of the owner of the video with the given id.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists with the given id
	GetVideoOwner(ctx context.Context, videoId uuid.UUID) (uuid.UUID, error)

	// CreatePlaylist creates a new playlist for the given user and returns the new playlist id.
	CreatePlaylist(ctx context.Context, userId uuid.UUID, name, description string, isPublic bool) (uuid.UUID, error)

	// DeletePlaylist deletes the playlist with the given id owned by the given user.
	//
	// Errors:
	//   - [ErrPlaylistNotFound] - no playlist exists for the given user with the given id
	DeletePlaylist(ctx context.Context, userId, playlistId uuid.UUID) error

	// EditPlaylist updates the name, description and visibility of the playlist
	// with the given id owned by the given user.
	//
	// Errors:
	//   - [ErrPlaylistNotFound] - no playlist exists for the given user with the given id
	EditPlaylist(ctx context.Context, userId, playlistId uuid.UUID, name, description string, isPublic bool) error

	// AddVideoToPlaylist adds the video with the given id to the playlist
	// with the given id owned by the given user.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists with the given id
	//   - [ErrPlaylistNotFound] - no playlist exists for the given user with the given id
	//   - [ErrPlaylistVideoConflict] - the video is already in the playlist
	AddVideoToPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error

	// DeleteVideoFromPlaylist removes the video with the given id from the playlist
	// with the given id owned by the given user.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists with the given id
	//   - [ErrPlaylistNotFound] - no playlist exists for the given user with the given id
	//   - [ErrPlaylistVideoNotFound] - the video is not in the playlist
	DeleteVideoFromPlaylist(ctx context.Context, userId, videoId, playlistId uuid.UUID) error

	// GetUserPlaylists returns the playlists of the given user,
	// paginated by the given last playlist id and limit.
	GetUserPlaylists(ctx context.Context, userId, lastPlaylistId uuid.UUID, limit int, includePrivates bool) ([]Playlist, error)

	// GetUserPlaylistsWithVideoStatus returns the playlists of the given user with
	// a flag indicating whether the video with the given id is in each playlist,
	// paginated by the given last playlist id and limit.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists with the given id
	GetUserPlaylistsWithVideoStatus(ctx context.Context, userId, videoId, lastPlaylistId uuid.UUID, limit int, includePrivates bool) ([]PlaylistWithVideoStatus, error)

	// GetPlaylist returns the playlist with the given id.
	//
	// Errors:
	//   - [ErrPlaylistNotFound] - no playlist exists with the given id
	GetPlaylist(ctx context.Context, playlistId uuid.UUID) (*Playlist, error)

	// GetVideosInPlaylist returns the videos in the playlist with the given id,
	// paginated by the given last id and limit.
	GetVideosInPlaylist(ctx context.Context, playlistId uuid.UUID, lastId int, limit int) ([]PlaylistVideo, error)

	// SavePlaylist adds the playlist with the given id to the given user's saved playlists.
	//
	// Errors:
	//   - [ErrSavedPlaylistConflict] - the playlist is already saved by the user
	SavePlaylist(ctx context.Context, playlistId, userId uuid.UUID) error

	// UnsavePlaylist removes the playlist with the given id from the given user's saved playlists.
	//
	// Errors:
	//   - [ErrSavedPlaylistNotFound] - the playlist is not saved by the user
	UnsavePlaylist(ctx context.Context, playlistId, userId uuid.UUID) error

	// GetSavedPlaylists returns the playlists saved by the given user,
	// paginated by the given last id and limit.
	GetSavedPlaylists(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]SavedPlaylist, error)

	// DeleteVideo deletes the video with the given id owned by the given user.
	//
	// Errors:
	//   - [ErrVideoNotFound] - no video exists for the given user with the given id
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
			"pending videos cleanup",
			service.pendingVideosCleanupJob,
			workers.WithSchedules(workers.WeeklyAt(time.Friday, 2, 0)),
			workers.WithTimeout(10*time.Minute),
			workers.WithBackoffStrategy(workers.DecorrelatedJitterBackoff(10*time.Minute)),
			workers.WithSingleInstance(),
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
	UserId      uuid.UUID
	Title       string
	Description string
}

func (me *impl) CreateVideo(ctx context.Context, userId uuid.UUID, title, description string) (uuid.UUID, error) {
	videoId := uuid.Must(uuid.NewV7())

	if err := me.queries.InsertPendingVideo(ctx, queries.InsertPendingVideoParams{
		Id:          videoId,
		UserId:      userId,
		Title:       title,
		Description: sql.NullString{Valid: description != "", String: description},
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
		UserId:      pending.UserId,
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
	UserId      uuid.UUID
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
		UserId:      video.UserId,
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
			UserId:      v.UserId,
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
			UserId:      v.UserId,
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
				UserId:      row.UserId,
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

func (me *impl) CreatePlaylist(ctx context.Context, userId uuid.UUID, name, description string, isPublic bool) (uuid.UUID, error) {
	playlistId := uuid.Must(uuid.NewV7())
	if err := me.queries.InsertPlaylist(ctx, queries.InsertPlaylistParams{
		Id:          playlistId,
		Name:        name,
		UserId:      userId,
		Description: sql.NullString{String: description, Valid: description != ""},
		IsPublic:    isPublic,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert playlist: %w", err)
	}

	return playlistId, nil
}

func (me *impl) DeletePlaylist(ctx context.Context, userId, playlistId uuid.UUID) error {
	if n, err := me.queries.DeletePlaylist(ctx, queries.DeletePlaylistParams{
		Id:     playlistId,
		UserId: userId,
	}); err != nil {
		return fmt.Errorf("failed to delete playlist: %w", err)
	} else if n == 0 {
		return ErrPlaylistNotFound
	}

	return nil
}

func (me *impl) EditPlaylist(ctx context.Context, userId, playlistId uuid.UUID, name, description string, isPublic bool) error {
	if n, err := me.queries.UpdatePlaylist(ctx, queries.UpdatePlaylistParams{
		Id:          playlistId,
		UserId:      userId,
		Name:        name,
		Description: sql.NullString{String: description, Valid: description != ""},
		IsPublic:    isPublic,
	}); err != nil {
		return fmt.Errorf("failed to update playlist: %w", err)
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
		Id:     playlistId,
		UserId: userId,
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
		Id:     playlistId,
		UserId: userId,
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
	UserId      uuid.UUID
	Name        string
	Description string
	IsPublic    bool
	VideosCount int
}

type PlaylistWithVideoStatus struct {
	Playlist
	HasVideo bool
}

type SavedPlaylist struct {
	Playlist
	SavedPlaylistId int
}

func (me *impl) GetUserPlaylists(ctx context.Context, userId, lastPlaylistId uuid.UUID, limit int, includePrivates bool) ([]Playlist, error) {
	playlists, err := me.queries.GetPlaylistsForUser(ctx, queries.GetPlaylistsForUserParams{
		UserId:          userId,
		LastPlaylistId:  uuid.NullUUID{UUID: lastPlaylistId, Valid: lastPlaylistId != uuid.Nil},
		Limit:           int32(limit),
		IncludePrivates: includePrivates,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all playlists for user: %w", err)
	}

	result := make([]Playlist, 0, len(playlists))
	for _, p := range playlists {
		result = append(result, Playlist{
			Id:          p.Id,
			UserId:      p.UserId,
			Name:        p.Name,
			Description: p.Description.String,
			IsPublic:    p.IsPublic,
			VideosCount: int(p.VideosCount),
		})
	}

	return result, nil
}

func (me *impl) GetUserPlaylistsWithVideoStatus(ctx context.Context, userId, videoId, lastPlaylistId uuid.UUID, limit int, includePrivates bool) ([]PlaylistWithVideoStatus, error) {
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
		UserId:          userId,
		VideoId:         videoId,
		LastPlaylistId:  uuid.NullUUID{UUID: lastPlaylistId, Valid: lastPlaylistId != uuid.Nil},
		Limit:           int32(limit),
		IncludePrivates: includePrivates,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get playlists with video status: %w", err)
	}

	result := make([]PlaylistWithVideoStatus, 0, len(rows))
	for _, row := range rows {
		result = append(result, PlaylistWithVideoStatus{
			Playlist: Playlist{
				Id:          row.Id,
				UserId:      row.UserId,
				Name:        row.Name,
				Description: row.Description.String,
				IsPublic:    row.IsPublic,
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
		UserId:      playlist.UserId,
		Name:        playlist.Name,
		Description: playlist.Description.String,
		IsPublic:    playlist.IsPublic,
		VideosCount: int(playlist.VideosCount),
	}, nil
}

func (me *impl) GetVideosInPlaylist(ctx context.Context, playlistId uuid.UUID, lastId int, limit int) ([]PlaylistVideo, error) {
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
				UserId:      row.UserId,
				Timestamp:   row.CreatedAt,
				Title:       row.Title,
				Description: row.Description.String,
			},
			PlaylistVideoId: int(row.PlaylistVideoId),
		})
	}

	return result, nil
}

func (me *impl) SavePlaylist(ctx context.Context, playlistId, userId uuid.UUID) error {
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()
	qtx := me.queries.WithTx(tx)

	if ok, err := qtx.CheckSavedPlaylist(ctx, queries.CheckSavedPlaylistParams{
		PlaylistId: playlistId,
		UserId:     userId,
	}); err != nil {
		return fmt.Errorf("failed to check saved playlist: %w", err)
	} else if ok {
		return ErrSavedPlaylistConflict
	}

	if err := qtx.InsertIntoSavedPlaylists(ctx, queries.InsertIntoSavedPlaylistsParams{
		PlaylistId: playlistId,
		UserId:     userId,
	}); err != nil {
		return fmt.Errorf("failed to insert into saved playlists: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

func (me *impl) UnsavePlaylist(ctx context.Context, playlistId, userId uuid.UUID) error {
	if n, err := me.queries.DeleteFromSavedPlaylists(ctx, queries.DeleteFromSavedPlaylistsParams{
		PlaylistId: playlistId,
		UserId:     userId,
	}); err != nil {
		return fmt.Errorf("failed to delete from saved playlists: %w", err)
	} else if n == 0 {
		return ErrSavedPlaylistNotFound
	}

	return nil
}

func (me *impl) GetSavedPlaylists(ctx context.Context, userId uuid.UUID, lastId int, limit int) ([]SavedPlaylist, error) {
	rows, err := me.queries.GetSavedPlaylistsForUser(ctx, queries.GetSavedPlaylistsForUserParams{
		UserId:              userId,
		LastSavedPlaylistId: sql.NullInt64{Int64: int64(lastId), Valid: lastId != 0},
		Limit:               int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get saved playlists: %w", err)
	}

	result := make([]SavedPlaylist, 0, len(rows))
	for _, row := range rows {
		result = append(result, SavedPlaylist{
			Playlist: Playlist{
				Id:          row.Id,
				UserId:      row.UserId,
				Name:        row.Name,
				Description: row.Description.String,
				IsPublic:    row.IsPublic,
				VideosCount: int(row.VideosCount),
			},
			SavedPlaylistId: int(row.SavedPlaylistId),
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
			if ok, err := events.Dequeue(ctx, me.redis, events.UserDeletedEvent, &payload); err != nil {
				return fmt.Errorf("failed to dequeue %q event: %w", events.UserDeletedEvent, err)
			} else if !ok {
				continue
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

				if err := qtx.DeleteAllSavedPlaylistsForUser(ctx, payload.UserId); err != nil {
					return fmt.Errorf("failed to delete all saved playlists for user: %w", err)
				}

				if err := tx.Commit(); err != nil {
					return fmt.Errorf("failed to commit tx: %w", err)
				}

				return nil
			}(); err != nil {
				return err
			}

			for _, id := range deletedVideoIds {
				if err := events.Enqueu(
					ctx,
					me.redis,
					events.VideoDeletedEvent,
					events.VideoDeletedEventPayload{VideoId: id},
				); err != nil {
					return err
				}
			}
		}
	}
}

func (me *impl) pendingVideosCleanupJob(ctx context.Context) error {
	if err := me.queries.DeleteExpiredPendingVideos(ctx); err != nil {
		return fmt.Errorf("failed to delete expired pending videos: %w", err)
	}
	return nil
}

func (me *impl) DeleteVideo(ctx context.Context, videoId, userId uuid.UUID) error {
	if n, err := me.queries.DeleteVideoByIdForUser(ctx, queries.DeleteVideoByIdForUserParams{
		Id:     videoId,
		UserId: userId,
	}); err != nil {
		return fmt.Errorf("failed to delete video by id for user: %w", err)
	} else if n == 0 {
		return ErrVideoNotFound
	}

	if err := events.Enqueu(
		ctx,
		me.redis,
		events.VideoDeletedEvent,
		events.VideoDeletedEventPayload{VideoId: videoId},
	); err != nil {
		return err
	}

	return nil
}
