package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	media_service "github.com/assaidy/vodzilla/internals/services/media"
	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleGenerateVideoUpload(c fiber.Ctx) error {
	var request struct {
		ContentType string `json:"contentType"`
		FileSize    int64  `json:"fileSize"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.ContentType = strings.TrimSpace(request.ContentType)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.ContentType, validation.Required, validation.By(func(value any) error {
			if !strings.HasPrefix(value.(string), "video/") {
				return fmt.Errorf("must be a video file")
			}
			return nil
		})),
		validation.Field(&request.FileSize, validation.Required, validation.Max(32*utils.GigaByte)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	presignedUpload, err := me.mediaService.GenerateVideoPresignedPutUrls(
		c.RequestCtx(),
		currentUserId,
		request.ContentType,
		request.FileSize,
	)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"uploadId":  presignedUpload.UploadId,
		"objectKey": presignedUpload.ObjectKey,
		"chunks":    presignedUpload.Chunks,
	})
}

func (me *Handler) HandleConfirmVideoUpload(c fiber.Ctx) error {
	var request struct {
		ObjectKey string `json:"objectKey"`
		UploadId  string `json:"uploadId"`
		Parts     []struct {
			ETag       string `json:"etag"`
			PartNumber int    `json:"partNumber"`
		} `json:"parts"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.ObjectKey, validation.Required),
		validation.Field(&request.UploadId, validation.Required),
		validation.Field(&request.Parts, validation.Required),
	); err != nil {
		return errInvalidData.details(err)
	}

	parts := make([]media_service.CompleteVideoUploadPart, 0, len(request.Parts))
	for _, p := range request.Parts {
		parts = append(parts, media_service.CompleteVideoUploadPart{
			ETag:       p.ETag,
			PartNumber: p.PartNumber,
		})
	}

	if err := me.mediaService.ConfirmVideoUpload(
		c.RequestCtx(),
		request.ObjectKey,
		request.UploadId,
		parts,
	); err != nil {
		if errors.Is(err, media_service.ErrNoPendingVideoUpload) {
			return errNoPendingVideoUpload
		}
		if errors.Is(err, media_service.ErrInvalidConfirmVideoUploadData) {
			return errInvalidConfirmVideoUploadData
		}
		return err
	}

	return c.JSON(fiber.Map{"objectKey": request.ObjectKey})
}

func (me *Handler) HandlePostVideo(c fiber.Ctx) error {
	var request struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ObjectKey   string `json:"objectKey"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)
	request.ObjectKey = strings.TrimSpace(request.ObjectKey)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Title, validation.Required, validation.Length(1, 256)),
		validation.Field(&request.Description, validation.Length(0, 500)),
		validation.Field(&request.ObjectKey, validation.Required),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	videoId, err := me.videoService.CreateVideo(c.RequestCtx(), video_service.CreateVideoParams{
		OwnerId:     currentUserId,
		Title:       request.Title,
		Description: request.Description,
	})
	if err != nil {
		return err
	}

	if err := me.mediaService.PostVideo(c.RequestCtx(), videoId, request.ObjectKey); err != nil {
		if errors.Is(err, media_service.ErrOrphanUploadNotFound) {
			return errObjectNotFound
		}
		return err
	}

	if err := me.videoService.ActivateVideo(c.RequestCtx(), videoId); err != nil {
		return err
	}

	followerIds, err := me.socialService.GetAllFollowerIds(c.RequestCtx(), currentUserId)
	if err != nil {
		return err
	}
	for _, followerId := range followerIds {
		me.notify(
			c.RequestCtx(),
			followerId,
			notification_service.NewVideoPayload{
				UserId:  currentUserId,
				VideoId: videoId,
			},
		)
	}

	return c.JSON(fiber.Map{"videoId": videoId})
}

func (me *Handler) HandleEditVideoThumbnail(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	var request struct {
		ContentType string `json:"contentType"`
		FileSize    int64  `json:"fileSize"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.ContentType = strings.TrimSpace(request.ContentType)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.ContentType, validation.Required, validation.By(func(v any) error {
			if !strings.HasPrefix(v.(string), "image/") {
				return fmt.Errorf("must be an image type")
			}
			return nil
		})),
		validation.Field(&request.FileSize, validation.Required, validation.Max(5*utils.MegaByte)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	lock := me.newVideoLock(videoId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ownerId, err := me.videoService.GetVideoOwner(c.RequestCtx(), videoId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	} else if ownerId != currentUserId {
		return errVideoNotFound
	}

	upload, err := me.mediaService.GeneratePresignedThumbnailUpload(
		c.RequestCtx(),
		videoId,
		request.ContentType,
		request.FileSize,
	)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"uploadUrl": upload.UploadUrl,
		"objectKey": upload.ObjectKey,
	})
}

func (me *Handler) HandleConfirmVideoThumbnailUpload(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	lock := me.newVideoLock(videoId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ownerId, err := me.videoService.GetVideoOwner(c.RequestCtx(), videoId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	} else if ownerId != currentUserId {
		return errVideoNotFound
	}

	thumbnailUrl, err := me.mediaService.ConfirmThumbnailUpload(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, media_service.ErrNoPendingThumbnailUpload) {
			return errThumbnailNotFound
		}
		return err
	}

	return c.JSON(fiber.Map{"thumbnailUrl": thumbnailUrl})
}

func (me *Handler) HandleDeleteVideoThumbnail(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	lock := me.newVideoLock(videoId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ownerId, err := me.videoService.GetVideoOwner(c.RequestCtx(), videoId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	} else if ownerId != currentUserId {
		return errVideoNotFound
	}

	if err := me.mediaService.DeleteThumbnail(c.RequestCtx(), videoId); err != nil {
		if errors.Is(err, media_service.ErrThumbnailNotFound) {
			return errThumbnailNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

type videoResponse struct {
	Id          uuid.UUID `json:"id"`
	OwnerId     uuid.UUID `json:"ownerId"`
	Timestamp   time.Time `json:"timestamp"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	// TODO: these are used for pagination; they will be replaced by index (reordering feature) in future.
	WatchlaterVideoId int `json:"watchlaterVideoId,omitempty"`
	PlaylistVideoId   int `json:"playlistVideoId,omitempty"`
}

func (me *Handler) HandleGetVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	video, err := me.videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	}

	return c.JSON(videoResponse{
		Id:          video.Id,
		OwnerId:     video.OwnerId,
		Timestamp:   video.Timestamp,
		Title:       video.Title,
		Description: video.Description,
	})
}

func (me *Handler) HandleGetVideoStreamUrl(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	url, err := me.mediaService.GenerateVideoPresignedGetUrl(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, media_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	}

	return c.JSON(fiber.Map{"url": url})
}

func (me *Handler) HandleDeleteVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	lock := me.newVideoLock(videoId)
	if err := lock.SpinWLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.WUnLock(c.RequestCtx())

	if err := me.videoService.DeleteVideo(c.RequestCtx(), videoId, currentUserId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleGetVideosForUser(c fiber.Ctx) error {
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

	videos, err := me.videoService.GetVideosForUser(
		c.RequestCtx(),
		userId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		items = append(items, videoResponse{
			Id:          v.Id,
			OwnerId:     v.OwnerId,
			Timestamp:   v.Timestamp,
			Title:       v.Title,
			Description: v.Description,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(videos[len(videos)-1].Id)
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetVideosCountForUser(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
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

	count, err := me.videoService.GetVideosCountForUser(c.RequestCtx(), userId)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"count": count})
}

func (me *Handler) HandleGetVideoThumbnailUrl(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	lock := me.newVideoLock(videoId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	thumbnailUrl, err := me.mediaService.GetThumbnailUrl(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, media_service.ErrThumbnailNotFound) {
			return errThumbnailNotFound
		}
		return err
	}

	return c.JSON(fiber.Map{"thumbnailUrl": thumbnailUrl})
}
