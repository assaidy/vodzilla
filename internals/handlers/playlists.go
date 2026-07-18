package handlers

import (
	"errors"
	"strings"

	video_service "github.com/assaidy/vodzilla/internals/services/video"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleCreatePlaylist(c fiber.Ctx) error {
	var request struct {
		Name string `json:"name"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Name = strings.TrimSpace(request.Name)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Name, validation.Required, validation.Length(1, 50)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	playlistId, err := me.videoService.CreatePlaylist(c.RequestCtx(), currentUserId, request.Name)
	if err != nil {
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
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	lock := me.newUserLock(userId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	playlists, err := me.videoService.GetPlaylists(
		c.RequestCtx(),
		userId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items := make([]playlistResponse, 0, len(playlists))
	for _, p := range playlists {
		items = append(items, playlistResponse{
			Id:          p.Id,
			Name:        p.Name,
			VideosCount: p.VideosCount,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(playlists[len(playlists)-1].Id)
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetPlaylistsWithVideoStatus(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	lock := me.newUserLock(userId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	playlists, err := me.videoService.GetPlaylistsWithVideoStatus(
		c.RequestCtx(),
		userId,
		videoId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	}

	// I cannot add HasVideo field with tag omitempty to [playlistResponse] and reused it;
	// omitempty would drop HasVideo=false, but the this response must include it.
	items := make([]fiber.Map, 0, len(playlists))
	for _, p := range playlists {
		items = append(items, fiber.Map{
			"id":          p.Id,
			"name":        p.Name,
			"videosCount": p.VideosCount,
			"hasVideo":    p.HasVideo,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(playlists[len(playlists)-1].Id)
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetPlaylist(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
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

func (me *Handler) HandleRenamePlaylist(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	var request struct {
		Name string `json:"name"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Name = strings.TrimSpace(request.Name)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Name, validation.Required, validation.Length(1, 50)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.RenamePlaylist(
		c.RequestCtx(),
		currentUserId,
		playlistId,
		request.Name,
	); err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeletePlaylist(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	lock := me.newPlaylistLock(playlistId)
	if err := lock.SpinWLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.WUnLock(c.RequestCtx())

	if err := me.videoService.DeletePlaylist(c.RequestCtx(), currentUserId, playlistId); err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleAddVideoToPlaylist(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.AddVideoToPlaylist(
		c.RequestCtx(),
		currentUserId,
		videoId,
		playlistId,
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
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.DeleteVideoFromPlaylist(
		c.RequestCtx(),
		currentUserId,
		videoId,
		playlistId,
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

func (me *Handler) HandleGetPlaylistVideos(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	pr, err := parsePaginatedRequest[int](c)
	if err != nil {
		return err
	}

	videos, err := me.videoService.GetVideosInPlaylist(
		c.RequestCtx(),
		playlistId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	}

	items := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		items = append(items, videoResponse{
			Id:              v.Id,
			OwnerId:         v.OwnerId,
			Timestamp:       v.Timestamp,
			Title:           v.Title,
			Description:     v.Description,
			PlaylistVideoId: v.PlaylistVideoId,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(videos[len(videos)-1].PlaylistVideoId)
	}

	return c.JSON(response)
}
