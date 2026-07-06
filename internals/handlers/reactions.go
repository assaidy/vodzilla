package handlers

import (
	"errors"
	"strings"
	"time"

	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleViewVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	me.videoMutex.RLock(videoId.String())
	defer me.videoMutex.RUnlock(videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	if err := me.reactionService.ViewVideo(c.RequestCtx(), videoId, currentUserId); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleCreateVideoComment(c fiber.Ctx) error {
	var request struct {
		VideoId uuid.UUID `uri:"video_id"`
		Content string    `json:"content"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Content = strings.TrimSpace(request.Content)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Content, validation.Required, validation.Length(1, 500)),
	); err != nil {
		return extractValidationError(err)
	}

	me.videoMutex.RLock(request.VideoId.String())
	defer me.videoMutex.RUnlock(request.VideoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), request.VideoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	commentId, err := me.reactionService.CreateVideoComment(
		c.RequestCtx(),
		currentUserId,
		request.VideoId,
		request.Content,
	)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"commentId": *commentId})
}

type commentResponse struct {
	Id           uuid.UUID `json:"id"`
	OwnerId      uuid.UUID `json:"ownerId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
	RepliesCount int       `json:"repliesCount"`
}

func (me *Handler) HandleGetVideoComments(c fiber.Ctx) error {
	var request struct {
		VideoId       uuid.UUID `uri:"video_id"`
		LastCommentId uuid.UUID `query:"last_comment_id"`
		Limit         int       `query:"limit"`
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

	me.videoMutex.RLock(request.VideoId.String())
	defer me.videoMutex.RUnlock(request.VideoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), request.VideoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	comments, err := me.reactionService.GetVideoComments(
		c.RequestCtx(),
		request.VideoId,
		request.LastCommentId,
		request.Limit,
	)
	if err != nil {
		return err
	}

	response := make([]commentResponse, 0, len(comments))
	for _, c := range comments {
		response = append(response, commentResponse{
			Id:           c.Id,
			OwnerId:      c.UserId,
			Content:      c.Content,
			CreatedAt:    c.CreatedAt,
			RepliesCount: c.RepliesCount,
		})
	}

	return c.JSON(response)
}

func (me *Handler) HandleCreateCommentReply(c fiber.Ctx) error {
	var request struct {
		CommentId uuid.UUID `uri:"video_id"`
		Content   string    `json:"content"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Content = strings.TrimSpace(request.Content)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Content, validation.Required, validation.Length(1, 500)),
	); err != nil {
		return extractValidationError(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	replyId, err := me.reactionService.CreateCommentReply(
		c.RequestCtx(),
		currentUserId,
		request.CommentId,
		request.Content,
	)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"replyId": *replyId})
}

func (me *Handler) HandleGetCommentReplies(c fiber.Ctx) error {
	var request struct {
		CommentId     uuid.UUID `uri:"comment_id"`
		LastCommentId uuid.UUID `query:"last_comment_id"`
		Limit         int       `query:"limit"`
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

	replies, err := me.reactionService.GetCommentReplies(
		c.RequestCtx(),
		request.CommentId,
		request.LastCommentId,
		request.Limit,
	)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	resposne := make([]commentResponse, 0, len(replies))
	for _, r := range replies {
		resposne = append(resposne, commentResponse{
			Id:           r.Id,
			OwnerId:      r.UserId,
			Content:      r.Content,
			CreatedAt:    r.CreatedAt,
			RepliesCount: r.RepliesCount,
		})
	}

	return c.JSON(resposne)
}

func (me *Handler) HandleDeleteComment(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.DeleteComment(c.RequestCtx(), currentUserId, commentId); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleEditComment(c fiber.Ctx) error {
	var request struct {
		CommentId uuid.UUID `uri:"comment_id"`
		Content   string    `json:"content"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	request.Content = strings.TrimSpace(request.Content)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Content, validation.Required, validation.Length(1, 500)),
	); err != nil {
		return extractValidationError(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.EditComment(
		c.RequestCtx(),
		currentUserId,
		request.CommentId,
		request.Content,
	); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleAddVideoFeeling(c fiber.Ctx) error {
	var request struct {
		VideoId uuid.UUID                    `uri:"video_id"`
		Kind    reaction_service.FeelingKind `json:"kind"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Kind, validation.In(
			reaction_service.FeelingLike,
			reaction_service.FeelingDislike,
		)),
	); err != nil {
		return extractValidationError(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	me.videoMutex.RLock(request.VideoId.String())
	defer me.videoMutex.RUnlock(request.VideoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), request.VideoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	if err := me.reactionService.AddVideoFeeling(
		c.RequestCtx(),
		currentUserId,
		request.VideoId,
		request.Kind,
	); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteVideoFeeling(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	me.videoMutex.RLock(videoId.String())
	defer me.videoMutex.RUnlock(videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	if err := me.reactionService.DeleteVideoFeeling(c.RequestCtx(), currentUserId, videoId); err != nil {
		if errors.Is(err, reaction_service.ErrFeelingNotFound) {
			return errFeelingNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleAddCommentFeeling(c fiber.Ctx) error {
	var request struct {
		CommentId uuid.UUID                    `uri:"comment_id"`
		Kind      reaction_service.FeelingKind `json:"kind"`
	}
	if err := c.Bind().All(&request); err != nil {
		return errInvalidRequest.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Kind, validation.In(
			reaction_service.FeelingLike,
			reaction_service.FeelingDislike,
		)),
	); err != nil {
		return extractValidationError(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.AddCommentFeeling(
		c.RequestCtx(),
		currentUserId,
		request.CommentId,
		request.Kind,
	); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteCommentFeeling(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errInvalidRequest.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.DeleteCommentFeeling(c.RequestCtx(), currentUserId, commentId); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		if errors.Is(err, reaction_service.ErrFeelingNotFound) {
			return errFeelingNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
