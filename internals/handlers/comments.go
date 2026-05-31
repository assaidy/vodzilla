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
)

func HandleGetVideoComments(c fiber.Ctx) error {
	videoId := c.Params("video_id")
	currentUserId := c.Locals("user_id").(string)

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	comments, err := reactionService.GetAllVideoComments(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	userCache := make(map[string]*user_service.User)
	templateComments := make([]any, 0, len(comments))
	for _, comment := range comments {
		owner, ok := userCache[comment.OwnerId]
		if !ok {
			owner, err = userService.GetUserById(c.RequestCtx(), comment.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return err
			}
			userCache[comment.OwnerId] = owner
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

func HandleGetCommentReplies(c fiber.Ctx) error {
	videoId := c.Params("video_id")
	commentId := c.Params("comment_id")
	currentUserId := c.Locals("user_id").(string)

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	replies, err := reactionService.GetAllCommentReplies(c.RequestCtx(), commentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userCache := make(map[string]*user_service.User)
	templateComments := make([]any, 0, len(replies))
	for _, reply := range replies {
		owner, ok := userCache[reply.OwnerId]
		if !ok {
			owner, err = userService.GetUserById(c.RequestCtx(), reply.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrUserNotFound) {
					continue
				}
				return err
			}
			userCache[reply.OwnerId] = owner
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

func HandleCreateComment(c fiber.Ctx) error {
	videoId := c.Params("video_id")
	userId := c.Locals("user_id").(string)

	content := strings.TrimSpace(c.FormValue("content"))
	parentId := strings.TrimSpace(c.FormValue("parent_id"))

	contentErr := validation.Validate(content, validation.Required, validation.Length(1, 500))
	if contentErr != nil {
		return render(c, templates.CreateCommentForm(templates.CreateCommentFormParams{
			VideoId:    videoId,
			ParentId:   parentId,
			Content:    content,
			ContentErr: contentErr,
		}))
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	commentId, err := reactionService.CreateComment(c.RequestCtx(), videoId, userId, content, parentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrParentCommentNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	owner, err := userService.GetUserById(c.RequestCtx(), userId)
	if err != nil {
		return err
	}

	oobTargetId := "COMMENTS_LIST"
	oobSwap := "prepend"
	if parentId != "" {
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
				CommentId:     commentId,
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

func HandleEditComment(c fiber.Ctx) error {
	videoId := c.Params("video_id")
	commentId := c.Params("comment_id")
	userId := c.Locals("user_id").(string)

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

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.EditComment(c.RequestCtx(), userId, commentId, content); err != nil {
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

func HandleDeleteComment(c fiber.Ctx) error {
	commentId := c.Params("comment_id")
	userId := c.Locals("user_id").(string)

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	if err := reactionService.DeleteComment(c.RequestCtx(), userId, commentId); err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	return nil
}
