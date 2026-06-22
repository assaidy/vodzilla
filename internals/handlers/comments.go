package handlers

import (
	"errors"
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

	currentUserId := c.Locals("user_id").(uuid.UUID)
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
			Id:            comment.Id,
			VideoId:       videoId,
			OwnerUsername: owner.Username,
			Content:       comment.Content,
			CreatedAt:     comment.CreatedAt,
			RepliesCount:  comment.RepliesCount,
			IsOwner:       currentUserId == comment.OwnerId,
		}))
	}

	return render(c, hyper.Group(
		hyper.Group(templateComments...),
		templates.CommentsLoader(templates.CommentsLoaderParams{
			VideoId:       videoId,
			LastCommentId: comments[len(comments)-1].Id,
		}),
	))
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
	currentUserId := c.Locals("user_id").(uuid.UUID)

	replies, err := me.reactionService.GetAllCommentReplies(c.RequestCtx(), commentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "comment not found")
		}
		return err
	}

	// TODO: DRY! getTemplateCommentsFromComments()
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
			Id:            reply.Id,
			VideoId:       videoId,
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
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	content := strings.TrimSpace(c.FormValue("comment"))
	if err := validation.Validate(content, validation.Required, validation.Length(1, 500)); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	me.videoMutex.RLock(videoId.String())
	defer me.videoMutex.RUnlock(videoId.String())

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "video not found")
	}

	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	commentId, err := me.reactionService.CreateComment(c.RequestCtx(), videoId, currentUser.Id, content, uuid.Nil)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.Comment(templates.CommentParams{
			Id:            *commentId,
			VideoId:       videoId,
			OwnerUsername: currentUser.Username,
			Content:       content,
			CreatedAt:     time.Now(),
			IsOwner:       true,
		}),

		// clear the input
		hyper.DIV(hyper.AttrId("#comment-input"), hyper.Attr("hx-swap-oob", "innerHTML")),
	))
}

// func (me *Handler) HandleEditComment(c fiber.Ctx) error {
// 	videoId, err := uuid.Parse(c.Params("video_id"))
// 	if err != nil {
// 		return fiber.NewError(fiber.StatusNotFound, "video not found")
// 	}
// 	commentId, err := uuid.Parse(c.Params("comment_id"))
// 	if err != nil {
// 		return fiber.NewError(fiber.StatusNotFound, "comment not found")
// 	}
// 	userId := c.Locals("user_id").(uuid.UUID)
//
// 	content := strings.TrimSpace(c.FormValue("content"))
//
// 	contentErr := validation.Validate(content, validation.Required, validation.Length(1, 500))
// 	if contentErr != nil {
// 		return render(c, templates.EditCommentForm(templates.EditCommentFormParams{
// 			VideoId:    videoId,
// 			CommentId:  commentId,
// 			Content:    content,
// 			ContentErr: contentErr,
// 		}))
// 	}
//
// 	if err := me.reactionService.EditComment(c.RequestCtx(), userId, commentId, content); err != nil {
// 		if errors.Is(err, reaction_service.ErrCommentNotFound) {
// 			return fiber.NewError(fiber.StatusNotFound, "comment not found")
// 		}
// 		return err
// 	}
//
// 	return render(c, hyper.Group(
// 		templates.EditCommentForm(templates.EditCommentFormParams{
// 			VideoId:   videoId,
// 			CommentId: commentId,
// 			Content:   content,
// 			Hide:      true,
// 		}),
// 		hyper.DIV(
// 			hyper.AttrId(fmt.Sprintf("comment-content-%s", commentId)),
// 			hyper.Attr("hx-swap-oob", "outerHTML"),
// 		)(
// 			hyper.P(hyper.AttrClass("text-sm"))(content),
// 		),
// 	))
// }

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
