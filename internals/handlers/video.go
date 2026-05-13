package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/assaidy/hyper/v2"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
)

func HandlePostVideo(c fiber.Ctx) error {
	title := strings.TrimSpace(c.FormValue("title"))
	description := strings.TrimSpace(c.FormValue("description"))
	contentType := strings.TrimSpace(c.FormValue("contentType"))
	fileSize, _ := strconv.ParseInt(c.FormValue("fileSize"), 10, 0)
	pendingVideoId := c.FormValue("pendingVideoId")

	titleErr := validation.Validate(title, validation.Required, validation.Length(1, 256))
	descriptionErr := validation.Validate(description, validation.Length(0, 500))
	fileSizeErr := validation.Validate(fileSize, validation.Required, validation.Max(32<<30)) // 32 GB max

	if errors.Join(titleErr, descriptionErr, fileSizeErr) != nil {
		return render(c, templates.PostVideoForm(templates.PostVideoFormParams{
			Title:          title,
			TitleErr:       titleErr,
			Description:    description,
			DescriptionErr: descriptionErr,
			VideoErr:       fileSizeErr,
		}))
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	createVideoResult, err := videoService.CreateVideo(c.RequestCtx(), video_service.CreateVideoParams{
		OwnerId:     c.Locals("user_id").(string),
		Title:       title,
		Description: description,
	})
	if err != nil {
		return err
	}

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	presignedUpload, err := mediaService.GeneratePresignedPutUrls(
		c.RequestCtx(),
		createVideoResult.ObjectKey,
		contentType,
		fileSize,
	)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		// reset the form and close dialog modal
		templates.PostVideoForm(templates.PostVideoFormParams{CloseDialogModal: true}),

		// upload the video
		hyper.DIV(hyper.AttrId("VIDEO_UPLOADERS_CONTAINER"), hyper.Attr("hx-swap-oob", "append"))(
			templates.VideoUploader(templates.VideoUploaderParams{
				PendingVideoId: pendingVideoId,
				VideoId:        createVideoResult.VideoId,
				UploadId:       presignedUpload.UploadId,
				UploadUrls:     presignedUpload.Urls,
				VideoTitle:     title,
			}),
		),
	))
}

func HandleCompleteVideoUpload(c fiber.Ctx) error {
	panic("unimplemented")
}
