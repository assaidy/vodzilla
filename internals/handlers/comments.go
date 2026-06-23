package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/assaidy/hyper/v2"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleGetVideoComments(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	var lastCommentId uuid.UUID
	var maxTimestamp time.Time
	if lastCommentIdQuery := c.Query("last_comment_id"); lastCommentIdQuery != "" {
		id, err := uuid.Parse(lastCommentIdQuery)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id format for last_comment_id query param")
		}
		lastCommentId = id
	} else {
		t, err := time.Parse(time.RFC3339, c.Query("max_timestamp"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid timestamp format for max_timestamp query param")
		}
		maxTimestamp = t
	}

	comments, err := me.reactionService.GetVideoComments(c.RequestCtx(), videoId, lastCommentId, maxTimestamp)
	if err != nil {
		return err
	}

	if len(comments) == 0 {
		// swap the loader with empty response (ie. remove it).
		return nil
	}

	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	templateComments, err := me.toTemplateComments(c, comments, videoId, currentUser)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		hyper.Group(templateComments...),
		templates.CommentsLoader(templates.CommentsLoaderParams{
			VideoId:       videoId,
			LastCommentId: comments[len(comments)-1].Id,
		}),
	))
}

func (me *Handler) toTemplateComments(c fiber.Ctx, comments []reaction_service.Comment, videoId uuid.UUID, currentUser *user_service.User) ([]any, error) {
	ownerCache := make(map[uuid.UUID]*user_service.User)
	templateComments := make([]any, 0, len(comments))

	for _, comment := range comments {
		owner, ok := ownerCache[comment.OwnerId]
		if !ok {
			var err error
			owner, err = me.userService.GetUserById(c.RequestCtx(), comment.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return nil, err
			}
			ownerCache[comment.OwnerId] = owner
		}
		templateComments = append(templateComments, templates.Comment(templates.CommentParams{
			Id:            comment.Id,
			VideoId:       videoId,
			OwnerUsername: owner.Username,
			Content:       comment.Content,
			CreatedAt:     comment.CreatedAt,
			IsOwner:       currentUser.Id == comment.OwnerId,
		}))
	}

	return templateComments, nil
}

func (me *Handler) HandleGetCommentReplies(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "comment not found")
	}

	var lastCommentId uuid.UUID
	var maxTimestamp time.Time
	if lastCommentIdQuery := c.Query("last_comment_id"); lastCommentIdQuery != "" {
		id, err := uuid.Parse(lastCommentIdQuery)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id format for last_comment_id query param")
		}
		lastCommentId = id
	} else {
		t, err := time.Parse(time.RFC3339, c.Query("max_timestamp"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid timestamp format for max_timestamp query param")
		}
		maxTimestamp = t
	}

	replies, err := me.reactionService.GetCommentReplies(c.RequestCtx(), commentId, lastCommentId, maxTimestamp)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "comment not found")
		}
		return err
	}

	if len(replies) == 0 {
		// swap the loader with empty response (ie. remove it).
		return nil
	}

	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	templateComments, err := me.toTemplateComments(c, replies, videoId, currentUser)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		hyper.Group(templateComments...),
		templates.RepliesLoader(templates.RepliesLoaderParams{
			VideoId:       videoId,
			CommentId:     commentId,
			LastCommentId: replies[len(replies)-1].Id,
		}),
	))
}

func (me *Handler) HandleCreateComment(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	content := strings.TrimSpace(c.FormValue("comment"))
	if err := validation.Validate(content, validation.Required, validation.Length(1, 500)); err != nil {
		return render(c, templates.CreateCommentForm(templates.CreateCommentFormParams{
			VideoId:         videoId,
			CurrentUsername: currentUser.Username,
			ContentErr:      err,
		}))
	}

	me.videoMutex.RLock(videoId.String())
	defer me.videoMutex.RUnlock(videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	commentId, err := me.reactionService.CreateComment(c.RequestCtx(), videoId, currentUser.Id, content, uuid.Nil)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.CreateCommentForm(templates.CreateCommentFormParams{VideoId: videoId, CurrentUsername: currentUser.Username}),

		hyper.DIV(hyper.AttrId("comments-list"), hyper.Attr("hx-swap-oob", "prepend"))(
			templates.Comment(templates.CommentParams{
				Id:            *commentId,
				VideoId:       videoId,
				OwnerUsername: currentUser.Username,
				Content:       content,
				CreatedAt:     time.Now(),
				IsOwner:       true,
			}),
		),
	))
}

func (me *Handler) HandleCreateReply(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "comment not found")
	}

	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	content := strings.TrimSpace(c.FormValue("comment"))
	if err := validation.Validate(content, validation.Required, validation.Length(1, 500)); err != nil {
		return render(c, templates.CreateReplyForm(templates.CreateReplyFormParams{
			VideoId:    videoId,
			CommentId:  commentId,
			ContentErr: err,
		}))
	}

	me.videoMutex.RLock(videoId.String())
	defer me.videoMutex.RUnlock(videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	replyId, err := me.reactionService.CreateComment(c.RequestCtx(), videoId, currentUser.Id, content, commentId)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.CreateReplyForm(templates.CreateReplyFormParams{VideoId: videoId, CommentId: commentId}),

		hyper.DIV(hyper.AttrId(fmt.Sprintf("replies-%s", commentId)), hyper.Attr("hx-swap-oob", "prepend"))(
			templates.Comment(templates.CommentParams{
				Id:            *replyId,
				VideoId:       videoId,
				OwnerUsername: currentUser.Username,
				Content:       content,
				CreatedAt:     time.Now(),
				IsOwner:       true,
			}),
		),
	))
}

func (me *Handler) HandleDeleteComment(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "comment not found")
	}
	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.DeleteComment(c.RequestCtx(), userId, commentId); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "comment not found")
		}
		return err
	}

	return nil
}
