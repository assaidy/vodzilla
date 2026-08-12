package handlers

import (
	"errors"
	"fmt"
	"strings"

	media_service "github.com/assaidy/vodzilla/internals/services/media"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type profileResponse struct {
	Id       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Bio      string    `json:"bio"`
}

func (me *Handler) handleGetProfile(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	user, err := me.userService.GetUserById(c.RequestCtx(), currentUserId)
	if err != nil {
		return err
	}

	return c.JSON(profileResponse{
		Id:       user.Id,
		Name:     user.Name,
		Username: user.Username,
		Email:    user.Email,
		Bio:      user.Bio,
	})
}

type searchCursor struct {
	LastRank float32
	LastId   uuid.UUID
}

// TEST: handleSearchProfiles
func (me *Handler) handleSearchProfiles(c fiber.Ctx) error {
	query := c.Query("query")
	if err := validation.Validate(&query, validation.Required, validation.Length(1, 50)); err != nil {
		return errInvalidData.details(fiber.Map{"query": err})
	}

	pr, err := parsePaginatedRequest[searchCursor](c)
	if err != nil {
		return err
	}

	profiles, err := me.userService.SearchUsers(c.RequestCtx(), query, pr.Cursor.LastRank, pr.Cursor.LastId, pr.Limit)
	if err != nil {
		return err
	}

	items := make([]profileResponse, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, profileResponse{
			Id:       p.Id,
			Name:     p.Name,
			Username: p.Username,
			Email:    p.Email,
			Bio:      p.Bio,
		})
	}

	response := newPaginatedResponse(items, pr.Limit)
	if response.HasMore {
		last := profiles[len(profiles)-1]
		response.Cursor = encodeCursor(searchCursor{
			LastRank: last.Rank,
			LastId:   last.Id,
		})
	}

	return c.JSON(response)
}

func (me *Handler) handleGetProfileByUsername(c fiber.Ctx) error {
	username := c.Params("username")

	user, err := me.userService.GetUserByUsername(c.RequestCtx(), username)
	if err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return errUserNotFound
		}
		return err
	}

	return c.JSON(profileResponse{
		Id:       user.Id,
		Name:     user.Name,
		Username: user.Username,
		Email:    user.Email,
		Bio:      user.Bio,
	})
}

func (me *Handler) handleGetProfileById(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	user, err := me.userService.GetUserById(c.RequestCtx(), userId)
	if err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return errUserNotFound
		}
		return err
	}

	return c.JSON(profileResponse{
		Id:       user.Id,
		Name:     user.Name,
		Username: user.Username,
		Email:    user.Email,
		Bio:      user.Bio,
	})
}

func (me *Handler) handleEditProfile(c fiber.Ctx) error {
	var request struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Bio      string `json:"bio"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Username = strings.TrimSpace(request.Username)
	request.Bio = strings.TrimSpace(request.Bio)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.Name, validation.Required, validation.Length(1, 256)),
		validation.Field(&request.Username, validation.Required, validation.Length(1, 32), usernameLettersRule),
		validation.Field(&request.Bio, validation.Length(0, 500)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.userService.EditProfile(
		c.RequestCtx(),
		currentUserId,
		request.Name,
		request.Username,
		request.Bio,
	); err != nil {
		if errors.Is(err, user_service.ErrUsernameConflict) {
			return errUsernameConflict
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) handleDeleteProfile(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	lock := me.newUserLock(currentUserId)
	if err := lock.SpinWLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.WUnLock(c.RequestCtx())

	if err := me.userService.DeleteUser(c.RequestCtx(), currentUserId); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) handleEditProfileAvatar(c fiber.Ctx) error {
	var request struct {
		ContentType string `json:"contentType"`
		FileSize    int64  `json:"fileSize"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return errInvalidRequestBody.details(err)
	}

	request.ContentType = strings.TrimSpace(request.ContentType)

	if err := validation.ValidateStruct(&request,
		validation.Field(&request.ContentType, validation.Required, validation.By(func(v any) error {
			if !strings.HasPrefix(v.(string), "image/") {
				return fmt.Errorf("must be an image type")
			}
			return nil
		})),
		validation.Field(&request.FileSize, validation.Required, validation.Max(2*utils.MegaByte)),
	); err != nil {
		return errInvalidData.details(err)
	}

	currentUserId := c.Locals("user_id").(uuid.UUID)

	upload, err := me.mediaService.GeneratePresignedAvatarUpload(
		c.RequestCtx(),
		currentUserId,
		request.ContentType,
		request.FileSize,
	)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"uploadUrl": upload.UploadUrl,
		"objectKey": upload.ObjectKey,
	})
}

func (me *Handler) handleConfirmProfileAvatarUpload(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	avatarUrl, err := me.mediaService.ConfirmAvatarUpload(c.RequestCtx(), currentUserId)
	if err != nil {
		if errors.Is(err, media_service.ErrNoPendingAvatarUpload) {
			return errAvatarNotFound
		}
		return err
	}

	return c.JSON(fiber.Map{"avatarUrl": avatarUrl})
}

func (me *Handler) handleDeleteProfileAvatar(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.mediaService.DeleteAvatar(c.RequestCtx(), currentUserId); err != nil {
		if errors.Is(err, media_service.ErrAvatarNotFound) {
			return errAvatarNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) handleGetProfileAvatarUrl(c fiber.Ctx) error {
	userId, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return errUserNotFound
	}

	lock := me.newUserLock(userId)
	if err := lock.SpinRLock(c.RequestCtx(), spinLockTimeout); err != nil {
		return err
	}
	defer lock.RUnLock(c.RequestCtx())

	if ok, err := me.userService.DoesUserExist(c.RequestCtx(), userId); err != nil {
		return err
	} else if !ok {
		return errUserNotFound
	}

	avatarUrl, err := me.mediaService.GetAvatarUrl(c.RequestCtx(), userId)
	if err != nil {
		if errors.Is(err, media_service.ErrAvatarNotFound) {
			return errAvatarNotFound
		}
		return err
	}

	return c.JSON(fiber.Map{"avatarUrl": avatarUrl})
}
