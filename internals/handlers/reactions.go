package handlers

import (
	"errors"
	"strings"
	"time"

	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleViewVideo(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.lock.RLock(c.RequestCtx(), "video:"+videoId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "video:"+videoId.String())

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
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	var request struct {
		Content string `json:"content"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Content = strings.TrimSpace(request.Content)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Content, validation.Required, validation.Length(1, 500)),
	); err != nil {
		return errInvalidData.details(err)
	}

	if err := me.lock.RLock(c.RequestCtx(), "video:"+videoId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "video:"+videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	commentId, err := me.reactionService.CreateVideoComment(
		c.RequestCtx(),
		currentUserId,
		videoId,
		request.Content,
	)
	if err != nil {
		return err
	}

	ownerId, err := me.videoService.GetVideoOwner(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	if ownerId != currentUserId {
		if err := me.notify(
			c.RequestCtx(),
			ownerId,
			notification_service.VideoCommentPayload{
				UserId:    currentUserId,
				VideoId:   videoId,
				CommentId: commentId,
			},
		); err != nil {
			return err
		}
	}

	return c.JSON(fiber.Map{"commentId": commentId})
}

type commentResponse struct {
	Id           uuid.UUID `json:"id"`
	OwnerId      uuid.UUID `json:"ownerId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
	RepliesCount int       `json:"repliesCount"`
}

func (me *Handler) HandleGetVideoComments(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	if err := me.lock.RLock(c.RequestCtx(), "video:"+videoId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "video:"+videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	comments, err := me.reactionService.GetVideoComments(
		c.RequestCtx(),
		videoId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items := make([]commentResponse, 0, len(comments))
	for _, c := range comments {
		items = append(items, commentResponse{
			Id:           c.Id,
			OwnerId:      c.UserId,
			Content:      c.Content,
			CreatedAt:    c.CreatedAt,
			RepliesCount: c.RepliesCount,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(comments[len(comments)-1].Id)
	}

	return c.JSON(response)
}

func (me *Handler) HandleCreateCommentReply(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errCommentNotFound
	}

	var request struct {
		Content string `json:"content"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Content = strings.TrimSpace(request.Content)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Content, validation.Required, validation.Length(1, 500)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	replyId, err := me.reactionService.CreateCommentReply(
		c.RequestCtx(),
		currentUserId,
		commentId,
		request.Content,
	)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	ownerId, err := me.reactionService.GetCommentOwner(c.RequestCtx(), commentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}
	if ownerId != currentUserId {
		if err := me.notify(
			c.RequestCtx(),
			ownerId,
			notification_service.CommentReplyPayload{
				UserId:    currentUserId,
				CommentId: commentId,
				ReplyId:   replyId,
			},
		); err != nil {
			return err
		}
	}

	return c.JSON(fiber.Map{"replyId": replyId})
}

func (me *Handler) HandleGetCommentReplies(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errCommentNotFound
	}

	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	replies, err := me.reactionService.GetCommentReplies(
		c.RequestCtx(),
		commentId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	items := make([]commentResponse, 0, len(replies))
	for _, r := range replies {
		items = append(items, commentResponse{
			Id:           r.Id,
			OwnerId:      r.UserId,
			Content:      r.Content,
			CreatedAt:    r.CreatedAt,
			RepliesCount: r.RepliesCount,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(replies[len(replies)-1].Id)
	}

	return c.JSON(response)
}

func (me *Handler) HandleDeleteComment(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errCommentNotFound
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
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errCommentNotFound
	}

	var request struct {
		Content string `json:"content"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Content = strings.TrimSpace(request.Content)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Content, validation.Required, validation.Length(1, 500)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.EditComment(
		c.RequestCtx(),
		currentUserId,
		commentId,
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
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	var request struct {
		Kind reaction_service.FeelingKind `json:"kind"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Kind, validation.In(
			reaction_service.FeelingLike,
			reaction_service.FeelingDislike,
		)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.lock.RLock(c.RequestCtx(), "video:"+videoId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "video:"+videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	if err := me.reactionService.AddVideoFeeling(
		c.RequestCtx(),
		currentUserId,
		videoId,
		request.Kind,
	); err != nil {
		return err
	}

	ownerId, err := me.videoService.GetVideoOwner(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	if ownerId != currentUserId {
		if err := me.notify(
			c.RequestCtx(),
			ownerId,
			notification_service.VideoFeelingPayload{
				UserId:  currentUserId,
				VideoId: videoId,
				Feeling: string(request.Kind),
			},
		); err != nil {
			return err
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteVideoFeeling(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return errVideoNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.lock.RLock(c.RequestCtx(), "video:"+videoId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "video:"+videoId.String())

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
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errCommentNotFound
	}

	var request struct {
		Kind reaction_service.FeelingKind `json:"kind"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Kind, validation.In(
			reaction_service.FeelingLike,
			reaction_service.FeelingDislike,
		)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.AddCommentFeeling(
		c.RequestCtx(),
		currentUserId,
		commentId,
		request.Kind,
	); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	ownerId, err := me.reactionService.GetCommentOwner(c.RequestCtx(), commentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}
	if ownerId != currentUserId {
		if err := me.notify(
			c.RequestCtx(),
			ownerId,
			notification_service.CommentFeelingPayload{
				UserId:    currentUserId,
				CommentId: commentId,
				Feeling:   string(request.Kind),
			},
		); err != nil {
			return err
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteCommentFeeling(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errCommentNotFound
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
