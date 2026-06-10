package handlers

import (
	"errors"

	social_service "github.com/assaidy/vodzilla/internals/services/social"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// TODO: implement notifications for new comments/follows.
// also, update video comment/follow counts for all user's online clients.

func (me *Handler) HandleFollow(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	me.userLock(userId)
	defer me.userUnlock(userId)

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.socialService.Follow(c.RequestCtx(), currentUserId, userId); err != nil {
		switch {
		case errors.Is(err, social_service.ErrSelfFollowNotAllowed):
			return fiber.NewError(fiber.StatusForbidden, "self follow is not allowed")
		case errors.Is(err, social_service.ErrAlreadyFollowing):
			return fiber.NewError(fiber.StatusConflict, "already following")
		default:
			return err
		}
	}

	return render(c, templates.FollowButton(templates.FollowButtonParams{
		ProfileOwnerId: userId,
		IsFollowed:     true,
	}))
}

func (me *Handler) HandleUnfollow(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.socialService.Unfollow(c.RequestCtx(), currentUserId, userId); err != nil {
		if errors.Is(err, social_service.ErrNotFollowing) {
			return fiber.NewError(fiber.StatusNotFound, "not following")
		}
		return err
	}

	return render(c, templates.FollowButton(templates.FollowButtonParams{
		ProfileOwnerId: userId,
		IsFollowed:     false,
	}))
}
