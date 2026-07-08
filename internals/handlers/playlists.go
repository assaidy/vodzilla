package handlers

import (
	"errors"

	video_service "github.com/assaidy/vodzilla/internals/services/video"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleCreatePlaylist(c fiber.Ctx) error {
	var request struct {
		Name string `json:"name"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Name, validation.Required, validation.Length(1, 50)),
	); err != nil {
		return extractValidationError(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	playlistId, err := me.videoService.CreatePlaylist(c.RequestCtx(), currentUserId, request.Name)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNameConflict) {
			return errPlaylistNameConflict
		}
		return err
	}

	return c.JSON(fiber.Map{"playlistId": playlistId})
}

type playlistResponse struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	VideosCount int       `json:"videosCount"`
}

func (me *Handler) HandleGetPlaylists(c fiber.Ctx) error {
	var request struct {
		UserId         uuid.UUID `uri:"user_id"`
		LastPlaylistId uuid.UUID `query:"last_playlist_id"`
		Limit          int       `query:"limit"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if request.Limit == 0 {
		request.Limit = 15
	}

	me.userMutex.RLock(request.UserId.String())
	defer me.userMutex.RUnlock(request.UserId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), request.UserId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	playlists, err := me.videoService.GetPlaylists(
		c.RequestCtx(),
		request.UserId,
		request.LastPlaylistId,
		request.Limit,
	)
	if err != nil {
		return err
	}

	response := make([]playlistResponse, 0, len(playlists))
	for _, p := range playlists {
		response = append(response, playlistResponse{
			Id:          p.Id,
			Name:        p.Name,
			VideosCount: p.VideosCount,
		})
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetPlaylistsWithVideoStatus(c fiber.Ctx) error {
	var request struct {
		UserId         uuid.UUID `uri:"userId"`
		VideoId        uuid.UUID `uri:"videoId"`
		LastPlaylistId uuid.UUID `query:"last_playlist_id"`
		Limit          int       `query:"limit"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if request.Limit == 0 {
		request.Limit = 15
	}

	me.userMutex.RLock(request.UserId.String())
	defer me.userMutex.RUnlock(request.UserId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), request.UserId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	playlists, err := me.videoService.GetPlaylistsWithVideoStatus(
		c.RequestCtx(),
		request.UserId,
		request.VideoId,
		request.LastPlaylistId,
		request.Limit,
	)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound //
		}
		return err
	}

	// I cannot add HasVideo field with tag omitempty to [playlistResponse] and reused it;
	// omitempty would drop HasVideo=false, but the this response must include it.
	type playlistWithStatusResponse struct {
		Id          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		VideosCount int       `json:"videosCount"`
		HasVideo    bool      `json:"hasVideo"`
	}

	response := make([]playlistWithStatusResponse, 0, len(playlists))
	for _, p := range playlists {
		response = append(response, playlistWithStatusResponse{
			Id:          p.Id,
			Name:        p.Name,
			VideosCount: p.VideosCount,
			HasVideo:    p.HasVideo,
		})
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetPlaylist(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	playlist, err := me.videoService.GetPlaylist(c.RequestCtx(), playlistId)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	}

	return c.JSON(playlistResponse{
		Id:          playlist.Id,
		Name:        playlist.Name,
		VideosCount: playlist.VideosCount,
	})
}

func (me *Handler) HandleGetPlaylistVideos(c fiber.Ctx) error {
	var request struct {
		PlaylistId uuid.UUID `uri:"playlist_id"`
		LastId     int64     `query:"last_id"`
		Limit      int       `query:"limit"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if request.Limit == 0 {
		request.Limit = 15
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Limit, validation.Min(15), validation.Max(100)),
	); err != nil {
		return extractValidationError(err)
	}

	videos, err := me.videoService.GetVideosInPlaylist(
		c.RequestCtx(),
		request.PlaylistId,
		request.LastId,
		request.Limit,
	)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	}

	resposne := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		resposne = append(resposne, videoResponse{
			Id:              v.Id,
			OwnerId:         v.OwnerId,
			Timestamp:       v.Timestamp,
			Title:           v.Title,
			Description:     v.Description,
			PlaylistVideoId: v.PlaylistVideoId,
		})
	}

	return c.JSON(resposne)
}

func (me *Handler) HandleDeletePlaylist(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.DeletePlaylist(c.RequestCtx(), currentUserId, playlistId); err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleAddVideoToPlaylist(c fiber.Ctx) error {
	var request struct {
		VideoId    uuid.UUID `uri:"video_id"`
		PlaylistId uuid.UUID `uri:"playlist_id"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.AddVideoToPlaylist(
		c.RequestCtx(),
		currentUserId,
		request.VideoId,
		request.PlaylistId,
	); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		if errors.Is(err, video_service.ErrPlaylistVideoConflict) {
			return errPlaylistVideoConflict
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteVideoFromPlaylist(c fiber.Ctx) error {
	var request struct {
		VideoId    uuid.UUID `uri:"video_id"`
		PlaylistId uuid.UUID `uri:"playlist_id"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.DeleteVideoFromPlaylist(
		c.RequestCtx(),
		currentUserId,
		request.VideoId,
		request.PlaylistId,
	); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		if errors.Is(err, video_service.ErrPlaylistVideoNotFound) {
			return errPlaylistVideoNotFound
		}
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
