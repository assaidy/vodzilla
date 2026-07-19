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
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"isPublic"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Name, validation.Required, validation.Length(1, 50)),
		validation.Field(&request.Description, validation.Length(0, 500)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	playlistId, err := me.videoService.CreatePlaylist(
		c.RequestCtx(),
		currentUserId,
		request.Name,
		request.Description,
		request.IsPublic,
	)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"playlistId": playlistId})
}

type playlistResponse struct {
	Id          uuid.UUID `json:"id"`
	UserId      uuid.UUID `json:"userId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"isPublic"`
	VideosCount int       `json:"videosCount"`
}

func (me *Handler) HandleGetUserPlaylists(c fiber.Ctx) error {
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

	currentUserId := c.Locals("user_id").(uuid.UUID)
	includePrivates := userId == currentUserId

	playlists, err := me.videoService.GetUserPlaylists(
		c.RequestCtx(),
		userId,
		pr.Cursor,
		pr.Limit,
		includePrivates,
	)
	if err != nil {
		return err
	}

	items := make([]playlistResponse, 0, len(playlists))
	for _, p := range playlists {
		items = append(items, playlistResponse{
			Id:          p.Id,
			UserId:      p.UserId,
			Name:        p.Name,
			Description: p.Description,
			IsPublic:    p.IsPublic,
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
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)
	includePrivates := true

	playlists, err := me.videoService.GetUserPlaylistsWithVideoStatus(
		c.RequestCtx(),
		currentUserId,
		videoId,
		pr.Cursor,
		pr.Limit,
		includePrivates,
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
			"userId":      p.UserId,
			"name":        p.Name,
			"description": p.Description,
			"isPublic":    p.IsPublic,
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

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if !playlist.IsPublic && playlist.UserId != currentUserId {
		return errPlaylistNotFound
	}

	return c.JSON(playlistResponse{
		Id:          playlist.Id,
		UserId:      playlist.UserId,
		Name:        playlist.Name,
		Description: playlist.Description,
		IsPublic:    playlist.IsPublic,
		VideosCount: playlist.VideosCount,
	})
}

func (me *Handler) HandleEditPlaylist(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"isPublic"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Name, validation.Required, validation.Length(1, 50)),
		validation.Field(&request.Description, validation.Length(0, 500)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.EditPlaylist(
		c.RequestCtx(),
		currentUserId,
		playlistId,
		request.Name,
		request.Description,
		request.IsPublic,
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

	lock := me.newPlaylistLock(playlistId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if playlist, err := me.videoService.GetPlaylist(c.RequestCtx(), playlistId); err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	} else if !playlist.IsPublic && playlist.UserId != currentUserId {
		return errPlaylistNotFound
	}

	videos, err := me.videoService.GetVideosInPlaylist(
		c.RequestCtx(),
		playlistId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		items = append(items, videoResponse{
			Id:              v.Id,
			UserId:          v.UserId,
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
