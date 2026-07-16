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
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Bio       string    `json:"bio"`
	AvatarUrl string    `json:"avatarUrl,omitempty"`
}

func (me *Handler) HandleGetProfile(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	user, err := me.userService.GetUserById(c.RequestCtx(), currentUserId)
	if err != nil {
		return err
	}

	avatarUrl, err := me.mediaService.GetAvatarUrl(c.RequestCtx(), currentUserId)
	if err != nil && !errors.Is(err, media_service.ErrAvatarNotFound) {
		return err
	}

	return c.JSON(profileResponse{
		Id:        user.Id,
		Name:      user.Name,
		Username:  user.Username,
		Email:     user.Email,
		Bio:       user.Bio,
		AvatarUrl: avatarUrl,
	})
}

func (me *Handler) HandleGetProfileByUsername(c fiber.Ctx) error {
	username := c.Params("username")

	user, err := me.userService.GetUserByUsername(c.RequestCtx(), username)
	if err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return errUserNotFound
		}
		return err
	}

	avatarUrl, err := me.mediaService.GetAvatarUrl(c.RequestCtx(), user.Id)
	if err != nil && !errors.Is(err, media_service.ErrAvatarNotFound) {
		return err
	}

	return c.JSON(profileResponse{
		Id:        user.Id,
		Name:      user.Name,
		Username:  user.Username,
		Email:     user.Email,
		Bio:       user.Bio,
		AvatarUrl: avatarUrl,
	})
}

func (me *Handler) HandleGetProfileById(c fiber.Ctx) error {
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

	avatarUrl, err := me.mediaService.GetAvatarUrl(c.RequestCtx(), user.Id)
	if err != nil && !errors.Is(err, media_service.ErrAvatarNotFound) {
		return err
	}

	return c.JSON(profileResponse{
		Id:        user.Id,
		Name:      user.Name,
		Username:  user.Username,
		Email:     user.Email,
		Bio:       user.Bio,
		AvatarUrl: avatarUrl,
	})
}

func (me *Handler) HandleEditProfile(c fiber.Ctx) error {
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

func (me *Handler) HandleDeleteProfile(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	lockKey := "user:" + currentUserId.String()
	if err := me.lock.Lock(c.RequestCtx(), lockKey); err != nil {
		return err
	}
	defer me.lock.Unlock(c.RequestCtx(), lockKey)

	if err := me.userService.DeleteUser(c.RequestCtx(), currentUserId); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (me *Handler) HandleEditProfileAvatar(c fiber.Ctx) error {
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

func (me *Handler) HandleConfirmProfileAvatarUpload(c fiber.Ctx) error {
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

func (me *Handler) HandleDeleteProfileAvatar(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	if err := me.mediaService.DeleteAvatar(c.RequestCtx(), currentUserId); err != nil {
		if errors.Is(err, media_service.ErrAvatarNotFound) {
			return errAvatarNotFound
		}
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
