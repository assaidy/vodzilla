package handlers

import (
	"errors"
	"strings"
	"time"

	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
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

	if err := me.reactionService.AddView(
		c.RequestCtx(),
		currentUserId,
		videoId,
		reaction_service.ViewTargetVideo,
	); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleGetVideoViewsCount(c fiber.Ctx) error {
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

	count, err := me.reactionService.GetViewsCount(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"count": count})
}

func (me *Handler) HandleGetPlaylistViewsCount(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	lock := me.newPlaylistLock(playlistId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if playlist, err := me.videoService.GetPlaylist(c.RequestCtx(), playlistId); err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	} else if !playlist.IsPublic && playlist.UserId != currentUserId {
		return errPlaylistNotFound
	}

	count, err := me.reactionService.GetViewsCount(c.RequestCtx(), playlistId)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"count": count})
}

func (me *Handler) HandleViewPlaylist(c fiber.Ctx) error {
	playlistId, err := uuid.Parse(c.Params("playlist_id"))
	if err != nil {
		return errPlaylistNotFound
	}

	lock := me.newPlaylistLock(playlistId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if playlist, err := me.videoService.GetPlaylist(c.RequestCtx(), playlistId); err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return errPlaylistNotFound
		}
		return err
	} else if !playlist.IsPublic && playlist.UserId != currentUserId {
		return errPlaylistNotFound
	}

	if err := me.reactionService.AddView(
		c.RequestCtx(),
		currentUserId,
		playlistId,
		reaction_service.ViewTargetPlaylist,
	); err != nil {
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

	lock := me.newVideoLock(videoId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	video, err := me.videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	commentId, err := me.reactionService.CreateComment(
		c.RequestCtx(),
		currentUserId,
		videoId,
		reaction_service.CommentTargetVideo,
		request.Content,
	)
	if err != nil {
		return err
	}

	if video.UserId != currentUserId {
		me.notify(
			c.RequestCtx(),
			video.UserId,
			notification_service.VideoCommentPayload{
				UserId:    currentUserId,
				VideoId:   videoId,
				CommentId: commentId,
			},
		)
	}

	return c.JSON(fiber.Map{"commentId": commentId})
}

type commentResponse struct {
	Id           uuid.UUID `json:"id"`
	UserId       uuid.UUID `json:"userId"`
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

	comments, err := me.reactionService.GetComments(
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
			UserId:       c.UserId,
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

	lock := me.newCommentLock(commentId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	comment, err := me.reactionService.GetCommentById(c.RequestCtx(), commentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	replyId, err := me.reactionService.CreateComment(
		c.RequestCtx(),
		currentUserId,
		commentId,
		reaction_service.CommentTargetComment,
		request.Content,
	)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	if comment.UserId != currentUserId {
		me.notify(
			c.RequestCtx(),
			comment.UserId,
			notification_service.CommentReplyPayload{
				UserId:    currentUserId,
				CommentId: commentId,
				ReplyId:   replyId,
			},
		)
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

	lock := me.newCommentLock(commentId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ok, err := me.reactionService.DoesCommentExist(c.RequestCtx(), commentId); err != nil {
		return err
	} else if !ok {
		return errCommentNotFound
	}

	replies, err := me.reactionService.GetComments(
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
			UserId:       r.UserId,
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

	lock := me.newCommentLock(commentId)
	if err := lock.SpinWLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.WUnLock(c.RequestCtx())

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

	lock := me.newVideoLock(videoId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	video, err := me.videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return errVideoNotFound
		}
		return err
	}

	if err := me.reactionService.AddFeeling(
		c.RequestCtx(),
		currentUserId,
		videoId,
		reaction_service.FeelingTargetVideo,
		request.Kind,
	); err != nil {
		return err
	}

	if video.UserId != currentUserId {
		me.notify(c.RequestCtx(),
			video.UserId,
			notification_service.VideoFeelingPayload{
				UserId:  currentUserId,
				VideoId: videoId,
				Feeling: string(request.Kind),
			},
		)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteVideoFeeling(c fiber.Ctx) error {
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

	if ok, err := me.videoService.DoesVideoExist(c.RequestCtx(), videoId); err != nil {
		return err
	} else if !ok {
		return errVideoNotFound
	}

	if err := me.reactionService.DeleteFeeling(
		c.RequestCtx(),
		currentUserId,
		videoId,
	); err != nil {
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

	lock := me.newCommentLock(commentId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	comment, err := me.reactionService.GetCommentById(c.RequestCtx(), commentId)
	if err != nil {
		if errors.Is(err, reaction_service.ErrCommentNotFound) {
			return errCommentNotFound
		}
		return err
	}

	if err := me.reactionService.AddFeeling(
		c.RequestCtx(),
		currentUserId,
		commentId,
		reaction_service.FeelingTargetComment,
		request.Kind,
	); err != nil {
		return err
	}

	if comment.UserId != currentUserId {
		me.notify(
			c.RequestCtx(),
			comment.UserId,
			notification_service.CommentFeelingPayload{
				UserId:    currentUserId,
				CommentId: commentId,
				Feeling:   string(request.Kind),
			},
		)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleDeleteCommentFeeling(c fiber.Ctx) error {
	commentId, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return errCommentNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	lock := me.newCommentLock(commentId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ok, err := me.reactionService.DoesCommentExist(c.RequestCtx(), commentId); err != nil {
		return err
	} else if !ok {
		return errCommentNotFound
	}

	if err := me.reactionService.DeleteFeeling(
		c.RequestCtx(),
		currentUserId,
		commentId,
	); err != nil {
		if errors.Is(err, reaction_service.ErrFeelingNotFound) {
			return errFeelingNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
