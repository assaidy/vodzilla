package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/assaidy/hyper/v2"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandlePostVideo(c fiber.Ctx) error {
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

	pendingVideoId, err := uuid.Parse(c.FormValue("pendingVideoId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid pending video id")
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	videoId, err := me.videoService.CreateVideo(c.RequestCtx(), video_service.CreateVideoParams{
		OwnerId:     currentUserId,
		Title:       title,
		Description: description,
	})
	if err != nil {
		return err
	}

	objectKey := fmt.Sprintf("%s/%s", currentUserId, videoId)

	presignedUpload, err := me.mediaService.GeneratePresignedPutUrls(
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

		hyper.DIV(hyper.AttrId("video-uploaders-container"), templates.AttrHxSwapOob(templates.SwapAppend))(
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

func (me *Handler) HandleCompleteVideoUpload(c fiber.Ctx) error {
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

	if err := me.mediaService.CompleteUpload(c.RequestCtx(), request.VideoId, request.UploadId, parts); err != nil {
		switch {
		case errors.Is(err, media_service.ErrObjectNotFound):
			return fiber.NewError(fiber.StatusNotFound, "object not found")
		case errors.Is(err, media_service.ErrUploadExpired):
			return fiber.NewError(fiber.StatusForbidden, "upload expired")
		case errors.Is(err, media_service.ErrUploadAlreadyCompleted):
			return fiber.NewError(fiber.StatusForbidden, "upload already completed")
		case errors.Is(err, media_service.ErrInvalidCompleteUploadData):
			return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid complete upload data")
		default:
			return err
		}
	}

	return nil
}

func (me *Handler) HandleVideoPage(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	return render(c, templates.VideoPage(currentUser.Username, videoId))
}

func (me *Handler) HandleVideoPageContent(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	video, err := me.videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		}
		return err
	}

	owner, err := me.userService.GetUserById(c.RequestCtx(), video.OwnerId)
	if err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video owner not found")
		}
		return err
	}

	sourceUrl, err := me.mediaService.GeneratePresignedGetUrl(c.RequestCtx(), video.Id)
	if err != nil {
		return err
	}

	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	viewsCount, err := me.reactionService.GetVideoViewsCount(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	reactionCounts, err := me.reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	currentUserReaction, err := me.reactionService.GetVideoReactionForUser(c.RequestCtx(), videoId, currentUser.Id)
	if err != nil {
		return err
	}

	isViewed, err := me.reactionService.IsVideoViewedByUser(c.RequestCtx(), videoId, currentUser.Id)
	if err != nil {
		return err
	}

	isInWatchLater, err := me.videoService.IsInWatchLater(c.RequestCtx(), videoId, currentUser.Id)
	if err != nil {
		return err
	}

	playlists, err := me.videoService.GetAllPlaylistsWithVideoStatus(c.RequestCtx(), currentUser.Id, videoId)
	if err != nil {
		return err
	}

	templatePlaylists := make([]templates.PlaylistCheckboxParams, 0, len(playlists))
	for _, p := range playlists {
		templatePlaylists = append(templatePlaylists, templates.PlaylistCheckboxParams{
			VideoId:    videoId,
			PlaylistId: p.Id,
			Name:       p.Name,
			Checked:    p.HasVideo,
		})
	}

	isFollowed, err := me.socialService.IsFollower(c.RequestCtx(), currentUser.Id, video.OwnerId)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.VideoPageContent(templates.VideoPageContentParams{
			Id:            video.Id,
			OwnerId:       video.OwnerId,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			SourceUrl:     sourceUrl,
			Title:         video.Title,
			Description:   video.Description,
			Timestamp:     video.Timestamp,
			ViewsCount:    viewsCount,
			IsViewed:      isViewed,
			IsFollowed:    isFollowed,
			ReactionsParams: templates.ReactionsWidgetParams{
				VideoId:       videoId,
				LikesCount:    reactionCounts.Likes,
				DislikesCount: reactionCounts.Dislikes,
				IsLiked:       currentUserReaction.IsLike,
				IsDisliked:    currentUserReaction.IsDislike,
			},
			WatchLaterButtonParams: templates.WatchlaterButtonParams{
				VideoId:  videoId,
				IsActive: isInWatchLater,
			},
			AddToPlaylistModalParams: templates.AddToPlaylistModalParams{
				VideoId:   videoId,
				Playlists: templatePlaylists,
			},
			CurrentUserId:   currentUser.Id,
			CurrentUsername: currentUser.Username,
		}),

		hyper.DIV(hyper.AttrId("navbar"), templates.AttrHxSwapOob(templates.SwapOuterHtml))(
			templates.Navbar(templates.NavbarParams{
				Username: currentUser.Username,
			}),
		),
	))
}

func (me *Handler) HandleGetVideoStreamUrl(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	url, err := me.mediaService.GeneratePresignedGetUrl(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, media_service.ErrObjectNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video source not found")
		}
		return err
	}

	return c.JSON(fiber.Map{"url": url})
}

// TODO: add route for this.
func (me *Handler) HandleDeleteVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	currentUserId := c.Locals("user_id").(uuid.UUID)

	me.videoMutex.Lock(videoId.String())
	defer me.videoMutex.Unlock(videoId.String())

	if err := me.videoService.DeleteVideo(c.RequestCtx(), videoId, currentUserId); err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return redirect(c, "/login")
		}
		return err
	}

	return redirect(c, "/login")
}

func (me *Handler) toTemplateVideos(c fiber.Ctx, videos []video_service.Video) ([]templates.VideoCardParams, error) {
	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	ownerCache := make(map[uuid.UUID]*user_service.User)

	for _, v := range videos {
		owner, ok := ownerCache[v.OwnerId]
		if !ok {
			var err error
			owner, err = me.userService.GetUserById(c.RequestCtx(), v.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return nil, err
			}
			ownerCache[v.OwnerId] = owner
		}

		viewsCount, err := me.reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return nil, err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			ViewsCount:    viewsCount,
		})
	}

	return templateVideos, nil
}
