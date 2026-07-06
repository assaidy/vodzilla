package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	media_service "github.com/assaidy/vodzilla/internals/services/media"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandlePostVideo(c fiber.Ctx) error {
	var request struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ContentType string `json:"contentType"`
		FileSize    int64  `json:"fileSize"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)
	request.ContentType = strings.TrimSpace(request.ContentType)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Title, validation.Required, validation.Length(1, 256)),
		validation.Field(&request.Description, validation.Length(0, 500)),
		validation.Field(&request.ContentType, validation.Required, validation.By(func(value any) error {
			if !strings.HasPrefix(value.(string), "video/") {
				return fmt.Errorf("must be a video file")
			}
			return nil
		})),
		validation.Field(&request.FileSize, validation.Required, validation.Max(32*utils.GigaByte)),
	); err != nil {
		return extractValidationError(err)
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

	// FIX: if the upload is expired, we don't have a way to tell video service to remove the orphan video metadata row.
	// solution:
	// 	- separate the post video (metadata) from video uploading.
	// 	- upload video first, then send the post request with the object key with it.
	//	- store object key in video service. that way metadata and raw video are linked.
	// 	- set an expiration for uploaded videos to be linked with a video in video servcie.
	objectKey := fmt.Sprintf("%s/%s", currentUserId, videoId)
	presignedUpload, err := me.mediaService.GenerateVideoPresignedPutUrls(
		c.RequestCtx(),
		*videoId,
		objectKey,
		request.ContentType,
		request.FileSize,
	)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"videoId":  videoId,
		"uploadId": presignedUpload.UploadId,
		"partSize": presignedUpload.PartSize,
		"urls":     presignedUpload.Urls,
	})
}

func (me *Handler) HandleConfirmVideoUpload(c fiber.Ctx) error {
	var request struct {
		VideoId  uuid.UUID `uri:"video_id"`
		UploadId string    `json:"uploadId"`
		Parts    []struct {
			ETag       string `json:"etag"`
			PartNumber int    `json:"partNumber"`
		} `json:"parts"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.UploadId, validation.Required),
		validation.Field(&request.Parts, validation.Required),
	); err != nil {
		return extractValidationError(err)
	}

	parts := make([]media_service.CompleteVideoUploadPart, 0, len(request.Parts))
	for _, p := range request.Parts {
		parts = append(parts, media_service.CompleteVideoUploadPart{
			ETag:       p.ETag,
			PartNumber: p.PartNumber,
		})
	}

	if err := me.mediaService.ConfirmVideoUpload(c.RequestCtx(), request.VideoId, request.UploadId, parts); err != nil {
		if errors.Is(err, media_service.ErrNoPendingVideoUpload) {
			return fiber.NewError(fiber.StatusBadRequest, "no pending video upload")
		}
		if errors.Is(err, media_service.ErrInvalidConfirmVideoUploadData) {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid complete upload data")
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

// TODO: implement edit video thumbnail
func (me *Handler) HandleEditVideoThumbnail(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// TODO: implement complete video thumbnail upload
func (me *Handler) HandleConfirmVideoThumbnailUpload(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

// TODO: implement delete video thumbnail
func (me *Handler) HandleDeleteVideoThumbnail(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}

type videoResponse struct {
	Id                uuid.UUID `json:"id"`
	OwnerId           uuid.UUID `json:"ownerId"`
	Timestamp         time.Time `json:"timestamp"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	WatchlaterVideoId int64     `json:"watchlaterVideoId,omitempty"`
	PlaylistVideoId   int64     `json:"playlistVideoId,omitempty"`
}

func (me *Handler) HandleGetVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errInvalidRequest.details(err)
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
		return errInvalidRequest.details(err)
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
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	me.videoMutex.Lock(videoId.String())
	defer me.videoMutex.Unlock(videoId.String())

	if err := me.videoService.DeleteVideo(c.RequestCtx(), videoId, currentUserId); err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleGetVideosForUser(c fiber.Ctx) error {
	var request struct {
		UserId      uuid.UUID `uri:"user_id"`
		LastVideoId uuid.UUID `query:"last_video_id"`
		Limit       int       `query:"limit"`
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

	me.userMutex.RLock(request.UserId.String())
	defer me.userMutex.RUnlock(request.UserId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), request.UserId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	videos, err := me.videoService.GetAllVideosForUser(
		c.RequestCtx(),
		request.UserId,
		request.LastVideoId,
		request.Limit,
	)
	if err != nil {
		return err
	}

	response := make([]videoResponse, 0, len(videos))
	for _, v := range videos {
		response = append(response, videoResponse{
			Id:          v.Id,
			OwnerId:     v.OwnerId,
			Timestamp:   v.Timestamp,
			Title:       v.Title,
			Description: v.Description,
		})
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetVideosCountForUser(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	me.userMutex.RLock(userId.String())
	defer me.userMutex.RUnlock(userId.String())

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
