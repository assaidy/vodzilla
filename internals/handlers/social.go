package handlers

import (
	"errors"

	media_service "github.com/assaidy/vodzilla/internals/services/media"
	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleFollow(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	if err := me.lock.RLock(c.RequestCtx(), "user:"+userId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "user:"+userId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.socialService.Follow(c.RequestCtx(), currentUserId, userId); err != nil {
		if errors.Is(err, social_service.ErrSelfFollowNotAllowed) {
			return errSelfFollowNotAllowed
		}
		if errors.Is(err, social_service.ErrAlreadyFollowing) {
			return errAlreadyFollowing
		}
		return err
	}

	if err := me.notify(
		c.RequestCtx(),
		userId,
		notification_service.FollowPayload{
			UserId: currentUserId,
		},
	); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleUnfollow(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	if err := me.lock.RLock(c.RequestCtx(), "user:"+userId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "user:"+userId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.socialService.Unfollow(c.RequestCtx(), currentUserId, userId); err != nil {
		if errors.Is(err, social_service.ErrNotFollowing) {
			return errNotFollowing
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleIsFollowing(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	if err := me.lock.RLock(c.RequestCtx(), "user:"+userId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "user:"+userId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	ok, err := me.socialService.IsFollower(c.RequestCtx(), currentUserId, userId)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"isFollowing": ok})
}

func (me *Handler) HandleGetFollowCounts(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	if err := me.lock.RLock(c.RequestCtx(), "user:"+userId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "user:"+userId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	counts, err := me.socialService.GetFollowCounts(c.RequestCtx(), userId)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"followers": counts.Followers,
		"followeds": counts.Followeds,
	})
}

func (me *Handler) HandleGetFollowers(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	if err := me.lock.RLock(c.RequestCtx(), "user:"+userId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "user:"+userId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	ids, err := me.socialService.GetFollowerIds(
		c.RequestCtx(),
		userId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items, err := me.getProfilesByIds(c, ids)
	if err != nil {
		return err
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(ids[len(ids)-1])
	}

	return c.JSON(response)
}

func (me *Handler) HandleGetFolloweds(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	pr, err := parsePaginatedRequest[uuid.UUID](c)
	if err != nil {
		return err
	}

	if err := me.lock.RLock(c.RequestCtx(), "user:"+userId.String()); err != nil {
		return err
	}
	defer me.lock.RUnlock(c.RequestCtx(), "user:"+userId.String())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	ids, err := me.socialService.GetFollowedIds(
		c.RequestCtx(),
		userId,
		pr.Cursor,
		pr.Limit,
	)
	if err != nil {
		return err
	}

	items, err := me.getProfilesByIds(c, ids)
	if err != nil {
		return err
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		response.Cursor = encodeCursor(ids[len(ids)-1])
	}

	return c.JSON(response)
}

func (me *Handler) getProfilesByIds(c fiber.Ctx, ids []uuid.UUID) ([]profileResponse, error) {
	profiles := make([]profileResponse, 0, len(ids))

	for _, id := range ids {
		user, err := me.userService.GetUserById(c.RequestCtx(), id)
		if err != nil {
			if errors.Is(err, user_service.ErrUserNotFound) {
				continue
			}
			return nil, err
		}

		avatarUrl, err := me.mediaService.GetAvatarUrl(c.RequestCtx(), user.Id)
		if err != nil && !errors.Is(err, media_service.ErrAvatarNotFound) {
			return nil, err
		}

		profiles = append(profiles, profileResponse{
			Id:        user.Id,
			Name:      user.Name,
			Username:  user.Username,
			Email:     user.Email,
			Bio:       user.Bio,
			AvatarUrl: avatarUrl,
		})
	}

	return profiles, nil
}
