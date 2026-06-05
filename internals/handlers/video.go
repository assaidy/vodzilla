package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/assaidy/hyper/v2"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func HandlePostVideo(c fiber.Ctx) error {
	title := strings.TrimSpace(c.FormValue("title"))
	description := strings.TrimSpace(c.FormValue("description"))
	contentType := strings.TrimSpace(c.FormValue("contentType"))
	fileSize, _ := strconv.ParseInt(c.FormValue("fileSize"), 10, 0)

	titleErr := validation.Validate(title, validation.Required, validation.Length(1, 256))
	descriptionErr := validation.Validate(description, validation.Length(0, 500))
	contentTypeErr := validation.Validate(contentType, validation.Required, validation.By(func(value any) error {
		if !strings.HasPrefix(value.(string), "video/") {
			return fmt.Errorf("must be a video file")
		}
		return nil
	}))
	fileSizeErr := validation.Validate(fileSize, validation.Required, validation.Max(32<<30))

	if errors.Join(titleErr, descriptionErr, contentTypeErr, fileSizeErr) != nil {
		return render(c, templates.PostVideoForm(templates.PostVideoFormParams{
			Title:          title,
			TitleErr:       titleErr,
			Description:    description,
			DescriptionErr: descriptionErr,
			VideoErr:       errors.Join(contentTypeErr, fileSizeErr),
		}))
	}

	pendingVideoIdStr := c.FormValue("pendingVideoId")
	pendingVideoId, err := uuid.Parse(pendingVideoIdStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid pending video id")
	}

	currentUser := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videoId, err := videoService.CreateVideo(c.RequestCtx(), video_service.CreateVideoParams{
		OwnerId:     currentUser,
		Title:       title,
		Description: description,
	})
	if err != nil {
		return err
	}

	objectKey := fmt.Sprintf("%s/%s", currentUser, videoId)

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	presignedUpload, err := mediaService.GeneratePresignedPutUrls(
		c.RequestCtx(),
		*videoId,
		objectKey,
		contentType,
		fileSize,
	)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.PostVideoForm(templates.PostVideoFormParams{CloseDialogModal: true}),

		hyper.DIV(hyper.AttrId("VIDEO_UPLOADERS_CONTAINER"), hyper.Attr("hx-swap-oob", "append"))(
			templates.VideoUploader(templates.VideoUploaderParams{
				PendingVideoId: pendingVideoId,
				VideoId:        *videoId,
				UploadId:       presignedUpload.UploadId,
				PartSize:       presignedUpload.PartSize,
				UploadUrls:     presignedUpload.Urls,
				VideoTitle:     title,
			}),
		),
	))
}

func HandleCompleteVideoUpload(c fiber.Ctx) error {
	var request struct {
		VideoId  uuid.UUID `json:"videoId"`
		UploadId string    `json:"uploadId"`
		Parts    []struct {
			ETag       string `json:"etag"`
			PartNumber int    `json:"partNumber"`
		} `json:"parts"`
	}

	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	parts := make([]media_service.CompleteUploadPart, 0, len(request.Parts))
	for _, p := range request.Parts {
		parts = append(parts, media_service.CompleteUploadPart{
			ETag:       p.ETag,
			PartNumber: p.PartNumber,
		})
	}

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	if err := mediaService.CompleteUpload(c.RequestCtx(), request.VideoId, request.UploadId, parts); err != nil {
		switch {
		case errors.Is(err, media_service.ErrObjectNotFound):
			return fiber.NewError(fiber.StatusNotFound, "object not found")
		case errors.Is(err, media_service.ErrUploadExpired):
			return fiber.NewError(fiber.StatusForbidden, "upload expired")
		case errors.Is(err, media_service.ErrUploadAlreadyCompleted):
			return fiber.NewError(fiber.StatusForbidden, "upload already completed")
		case errors.Is(err, media_service.ErrInvalidCompleteUploadData):
			return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid complete upload data")
		}
		return err
	}

	return nil
}

func HandleGetVideoStreamUrl(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	url, err := mediaService.GeneratePresignedGetUrl(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, media_service.ErrObjectNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		}
		return err
	}

	return c.JSON(fiber.Map{"url": url})
}

func HandleViewVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if ok, err := videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.ViewVideo(c.RequestCtx(), videoId, userId); err != nil {
		return err
	}

	return nil
}

const (
	ReactionLike    = "like"
	ReactionDislike = "dislike"
)

func HandleAddVideoReaction(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)
	kind := c.Query("kind")

	if kind != ReactionLike && kind != ReactionDislike {
		return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if ok, err := videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.AddVidoeReaction(c.RequestCtx(), videoId, userId, kind); err != nil {
		return err
	}

	reactinCounts, err := reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	return render(c, templates.ReactionsWidget(templates.ReactionsWidgetParams{
		VideoId:       videoId,
		LikesCount:    reactinCounts.Likes,
		DislikesCount: reactinCounts.Dislikes,
		IsLiked:       kind == ReactionLike,
		IsDisliked:    kind == ReactionDislike,
	}))
}

func HandleDeleteVideoReaction(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)
	kind := c.Query("kind")

	if kind != ReactionLike && kind != ReactionDislike {
		return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if ok, err := videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.DeleteVidoeReaction(c.RequestCtx(), videoId, userId, kind); err != nil {
		return err
	}

	reactinCounts, err := reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	return render(c, templates.ReactionsWidget(templates.ReactionsWidgetParams{
		VideoId:       videoId,
		LikesCount:    reactinCounts.Likes,
		DislikesCount: reactinCounts.Dislikes,
	}))
}

func HandleAddToWatchLater(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.AddVideoToWatchlater(c.RequestCtx(), videoId, userId); err != nil {
		if errors.Is(err, video_service.ErrWatchlaterConflict) {
			return fiber.NewError(fiber.StatusConflict, "already in watch later")
		}
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		}
		return err
	}

	return render(c, templates.WatchLaterButton(templates.WatchLaterButtonParams{
		VideoId:  videoId,
		IsActive: true,
	}))
}

func HandleDeleteFromWatchLater(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.DeleteVideoFromWatchlater(c.RequestCtx(), videoId, userId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "not in watch later")
		}
		return err
	}

	return render(c, templates.WatchLaterButton(templates.WatchLaterButtonParams{
		VideoId:  videoId,
		IsActive: false,
	}))
}

func HandleCreatePlaylist(c fiber.Ctx) error {
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
	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)

	playlistId, err := videoService.CreatePlaylist(c.RequestCtx(), userId, name)
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
			hyper.AttrId("PLAYLIST_CHECKBOX_LIST"),
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

func HandleAddVideoToPlaylist(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.AddVideoToPlaylist(c.RequestCtx(), videoId, userId, playlistId); err != nil {
		switch {
		case errors.Is(err, video_service.ErrPlaylistVideoConflict):
			return fiber.NewError(fiber.StatusConflict, "already in playlist")
		case errors.Is(err, video_service.ErrVideoNotFound):
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		case errors.Is(err, video_service.ErrPlaylistNotFound):
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		}
		return err
	}

	return nil
}

func HandleDeleteVideoFromPlaylist(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.DeleteVideoFromPlaylist(c.RequestCtx(), videoId, userId, playlistId); err != nil {
		switch {
		case errors.Is(err, video_service.ErrPlaylistNotFound):
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		case errors.Is(err, video_service.ErrVideoNotFound):
			return fiber.NewError(fiber.StatusNotFound, "video not in playlist")
		}
		return err
	}

	return nil
}
