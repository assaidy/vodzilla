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
		return fiber.ErrNotFound
	}
	currentUserId := c.Locals("user_id").(uuid.UUID)

	comments, err := me.reactionService.GetAllVideoComments(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	ownerCache := make(map[uuid.UUID]*user_service.User)
	templateComments := make([]any, 0, len(comments))
	for _, comment := range comments {
		owner, ok := ownerCache[comment.OwnerId]
		if !ok {
			owner, err = me.userService.GetUserById(c.RequestCtx(), comment.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return err
			}
			ownerCache[comment.OwnerId] = owner
		}
		templateComments = append(templateComments, templates.Comment(templates.CommentParams{
			VideoId:       videoId,
			CommentId:     comment.Id,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			Content:       comment.Content,
			CreatedAt:     comment.CreatedAt,
			RepliesCount:  comment.RepliesCount,
			IsOwner:       currentUserId == comment.OwnerId,
		}))
	}

	return render(c, hyper.Group(templateComments...))
}

func (me *Handler) HandleGetCommentReplies(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	currentUserId := c.Locals("user_id").(uuid.UUID)

	replies, err := me.reactionService.GetAllCommentReplies(c.RequestCtx(), commentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	ownerCache := make(map[uuid.UUID]*user_service.User)
	templateComments := make([]any, 0, len(replies))
	for _, reply := range replies {
		owner, ok := ownerCache[reply.OwnerId]
		if !ok {
			owner, err = me.userService.GetUserById(c.RequestCtx(), reply.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return err
			}
			ownerCache[reply.OwnerId] = owner
		}
		templateComments = append(templateComments, templates.Comment(templates.CommentParams{
			VideoId:       videoId,
			CommentId:     reply.Id,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			Content:       reply.Content,
			CreatedAt:     reply.CreatedAt,
			RepliesCount:  reply.RepliesCount,
			IsOwner:       currentUserId == reply.OwnerId,
		}))
	}

	return render(c, hyper.Group(templateComments...))
}

func (me *Handler) HandleCreateComment(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	content := strings.TrimSpace(c.FormValue("content"))
	parentIdStr := strings.TrimSpace(c.FormValue("parent_id"))

	contentErr := validation.Validate(content, validation.Required, validation.Length(1, 500))
	if contentErr != nil {
		var pId uuid.UUID
		if parentIdStr != "" {
			pId, err = uuid.Parse(parentIdStr)
			if err != nil {
				return fiber.ErrNotFound
			}
		}
		return render(c, templates.CreateCommentForm(templates.CreateCommentFormParams{
			VideoId:    videoId,
			ParentId:   pId,
			Content:    content,
			ContentErr: contentErr,
		}))
	}

	var parentId uuid.UUID
	if parentIdStr != "" {
		parentId, err = uuid.Parse(parentIdStr)
		if err != nil {
			return fiber.ErrNotFound
		}
	}

	commentId, err := me.reactionService.CreateComment(c.RequestCtx(), videoId, userId, content, parentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrParentCommentNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	owner, err := me.userService.GetUserById(c.RequestCtx(), userId)
	if err != nil {
		return err
	}

	oobTargetId := "COMMENTS_LIST"
	oobSwap := "prepend"
	if parentId != uuid.Nil {
		oobTargetId = fmt.Sprintf("REPLIES_%s", parentId)
		oobSwap = "append"
	}

	return render(c, hyper.Group(
		templates.CreateCommentForm(templates.CreateCommentFormParams{
			VideoId:  videoId,
			ParentId: parentId,
		}),
		hyper.DIV(hyper.AttrId(oobTargetId), hyper.Attr("hx-swap-oob", oobSwap))(
			templates.Comment(templates.CommentParams{
				VideoId:       videoId,
				CommentId:     *commentId,
				OwnerName:     owner.Name,
				OwnerUsername: owner.Username,
				Content:       content,
				CreatedAt:     time.Now(),
				RepliesCount:  0,
				IsOwner:       true,
			}),
		),
	))
}

func (me *Handler) HandleEditComment(c fiber.Ctx) error {
	videoId, err := uuid.Parse(c.Params("video_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	content := strings.TrimSpace(c.FormValue("content"))

	contentErr := validation.Validate(content, validation.Required, validation.Length(1, 500))
	if contentErr != nil {
		return render(c, templates.EditCommentForm(templates.EditCommentFormParams{
			VideoId:    videoId,
			CommentId:  commentId,
			Content:    content,
			ContentErr: contentErr,
		}))
	}

	if err := me.reactionService.EditComment(c.RequestCtx(), userId, commentId, content); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	return render(c, hyper.Group(
		templates.EditCommentForm(templates.EditCommentFormParams{
			VideoId:   videoId,
			CommentId: commentId,
			Content:   content,
			Hide:      true,
		}),
		hyper.DIV(
			hyper.AttrId(fmt.Sprintf("COMMENT_CONTENT_%s", commentId)),
			hyper.Attr("hx-swap-oob", "outerHTML"),
		)(
			hyper.P(hyper.AttrClass("text-sm"))(content),
		),
	))
}

func (me *Handler) HandleDeleteComment(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.reactionService.DeleteComment(c.RequestCtx(), userId, commentId); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	return nil
}
