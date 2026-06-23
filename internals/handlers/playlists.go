package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/assaidy/hyper/v2"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleCreatePlaylist(c fiber.Ctx) error {
	name := strings.TrimSpace(c.FormValue("name"))
	videoIdStr := c.FormValue("videoId")

	var videoId uuid.UUID
	if videoIdStr != "" {
		var err error
		videoId, err = uuid.Parse(videoIdStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid video id")
		}
	}

	nameErr := validation.Validate(name, validation.Required, validation.Length(1, 50))

	if nameErr != nil {
		return render(c, templates.CreatePlaylistForm(templates.CreatePlaylistFormParams{
			VideoId: videoId,
			Name:    name,
			NameErr: nameErr,
		}))
	}

	userId := c.Locals("user_id").(uuid.UUID)

	playlistId, err := me.videoService.CreatePlaylist(c.RequestCtx(), userId, name)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNameConflict) {
			return render(c, templates.CreatePlaylistForm(templates.CreatePlaylistFormParams{
				VideoId: videoId,
				Name:    name,
				NameErr: fmt.Errorf("playlist with this name already exists"),
			}))
		}
		return err
	}

	return render(c, hyper.Group(
		templates.CreatePlaylistForm(templates.CreatePlaylistFormParams{
			VideoId: videoId,
		}),

		hyper.DIV(
			hyper.AttrId("playlist-checkbox-list"),
			hyper.Attr("hx-swap-oob", "prepend"),
		)(
			templates.PlaylistCheckbox(templates.PlaylistCheckboxParams{
				VideoId:    videoId,
				PlaylistId: *playlistId,
				Name:       name,
				Checked:    false,
			}),
		),
	))
}

func (me *Handler) HandleAddVideoToPlaylist(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "playlist not found")
	}

	var request struct {
		PlaylistName string `json:"playlistName"`
	}
	if err := c.Bind().All(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing playlistName field in request body")
	}

	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.AddVideoToPlaylist(c.RequestCtx(), videoId, userId, playlistId); err != nil {
		switch {
		case errors.Is(err, video_service.ErrVideoNotFound):
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		case errors.Is(err, video_service.ErrPlaylistVideoConflict):
			return fiber.NewError(fiber.StatusConflict, "already in playlist")
		case errors.Is(err, video_service.ErrPlaylistNotFound):
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		default:
			return err
		}
	}

	return render(c, templates.PlaylistCheckbox(templates.PlaylistCheckboxParams{
		VideoId:    videoId,
		PlaylistId: playlistId,
		Name:       request.PlaylistName,
		Checked:    true,
	}))
}

func (me *Handler) HandleDeleteVideoFromPlaylist(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "playlist not found")
	}

	var request struct {
		PlaylistName string `json:"playlistName"`
	}
	if err := c.Bind().All(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing playlistName field in request body")
	}

	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.videoService.DeleteVideoFromPlaylist(c.RequestCtx(), videoId, userId, playlistId); err != nil {
		switch {
		case errors.Is(err, video_service.ErrPlaylistNotFound):
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		case errors.Is(err, video_service.ErrVideoNotFound):
			return fiber.NewError(fiber.StatusNotFound, "video not in playlist")
		default:
			return err
		}
	}

	return render(c, templates.PlaylistCheckbox(templates.PlaylistCheckboxParams{
		VideoId:    videoId,
		PlaylistId: playlistId,
		Name:       request.PlaylistName,
		Checked:    false,
	}))
}

func (me *Handler) HandlePlaylistsPage(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.PlaylistsPage(currentUser.Username))
}

func (me *Handler) HandlePlaylistsPageContent(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	playlists, err := me.videoService.GetAllPlaylists(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	templatePlaylists := make([]templates.PlaylistCardParams, 0, len(playlists))
	for _, p := range playlists {
		templatePlaylists = append(templatePlaylists, templates.PlaylistCardParams{
			Id:          p.Id,
			Name:        p.Name,
			VideosCount: p.VideosCount,
		})
	}

	return render(c, hyper.Group(
		templates.PlaylistsPageContent(templates.PlaylistsPageContentParams{
			Username:  currentUser.Username,
			Playlists: templatePlaylists,
		}),

		hyper.DIV(hyper.AttrId("navbar"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PagePlaylists,
			}),
		),
	))
}

func (me *Handler) HandleNotificationsPage(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.NotificationsPage(currentUser.Username))
}

func (me *Handler) HandlePlaylistDetailPage(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "playlist not found")
	}

	return render(c, templates.PlaylistDetailPage(currentUser.Username, playlistId))
}

func (me *Handler) HandlePlaylistDetailPageContent(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "playlist not found")
	}

	playlist, err := me.videoService.GetPlaylist(c.RequestCtx(), currentUser.Id, playlistId)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		}
		return err
	}

	videos, err := me.videoService.GetAllVideosInPlaylist(c.RequestCtx(), currentUser.Id, playlistId)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		}
		return err
	}

	templateVideos, err := me.toTemplateVideos(c, videos)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.PlaylistDetailPageContent(templates.PlaylistDetailPageContentParams{
			Username: currentUser.Username,
			Playlist: templates.PlaylistCardParams{
				Id:          playlist.Id,
				Name:        playlist.Name,
				VideosCount: playlist.VideosCount,
			},
			Videos: templateVideos,
		}),

		hyper.DIV(hyper.AttrId("navbar"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PagePlaylists,
			}),
		),
	))
}
