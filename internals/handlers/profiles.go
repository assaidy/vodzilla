package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/assaidy/hyper/v2"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func (me *Handler) HandleProfilePage(c fiber.Ctx) error {
	profileUser, currentUser, err := me.getProfileUserAndCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.ProfilePage(currentUser.Username, profileUser.Username))
}

// TODO: we can do a lot of lazy loading and pagination here and in other places
func (me *Handler) HandleProfilePageContent(c fiber.Ctx) error {
	profileUser, currentUser, err := me.getProfileUserAndCurrentUser(c)
	if err != nil {
		return err
	}

	videosCount, err := me.videoService.GetVideosCountForUser(c.RequestCtx(), profileUser.Id)
	if err != nil {
		return err
	}

	videos, err := me.videoService.GetAllVideosForUser(c.RequestCtx(), profileUser.Id)
	if err != nil {
		return err
	}

	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		viewsCount, err := me.reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     profileUser.Name,
			OwnerUsername: profileUser.Username,
			ViewsCount:    viewsCount,
		})
	}

	followersCount, err := me.socialService.GetFollowersCount(c.RequestCtx(), profileUser.Id)
	if err != nil {
		return err
	}

	isFollowed, err := me.socialService.IsFollower(c.RequestCtx(), currentUser.Id, profileUser.Id)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.ProfilePageContent(templates.ProfilePageContentParams{
			OwnerId:        profileUser.Id,
			Username:       profileUser.Username,
			Name:           profileUser.Name,
			Bio:            profileUser.Bio,
			IsOwner:        profileUser.Username == currentUser.Username,
			Videos:         templateVideos,
			FollowersCount: followersCount,
			PostsCount:     videosCount,
			IsFollowed:     isFollowed,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageProfile,
			}),
		),
	))
}

func (me *Handler) getCurrentUser(c fiber.Ctx) (*user_service.User, error) {
	userId := c.Locals("user_id").(uuid.UUID)

	user, err := me.userService.GetUserById(c.RequestCtx(), userId)
	if err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return nil, redirect(c, "/login")
		}
		return nil, err
	}

	return user, nil
}

func (me *Handler) getProfileUserAndCurrentUser(c fiber.Ctx) (*user_service.User, *user_service.User, error) {
	profileUser, err := me.userService.GetUserByUsername(c.RequestCtx(), c.Params("username"))
	if err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return nil, nil, fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return nil, nil, fmt.Errorf("failed to get profile user: %w", err)
	}

	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return profileUser, currentUser, nil
}

func (me *Handler) HandleEditProfile(c fiber.Ctx) error {
	name := strings.TrimSpace(c.FormValue("name"))
	username := strings.TrimSpace(c.FormValue("username"))
	bio := strings.TrimSpace(c.FormValue("bio"))

	nameErr := validation.Validate(&name, validation.Required, validation.Length(1, 256))
	usernameErr := validation.Validate(&username, validation.Required, validation.Length(1, 32),
		validation.Match(usernameRegex).Error("can only cotain letters, digits or _"))
	bioErr := validation.Validate(&bio, validation.Length(0, 500))

	if errors.Join(nameErr, usernameErr, bioErr) != nil {
		return render(c, templates.EditProfileFrom(templates.EditProfileFromParams{
			Name:        name,
			NameErr:     nameErr,
			Username:    username,
			UsernameErr: usernameErr,
			Bio:         bio,
			BioErr:      bioErr,
		}))
	}

	userId := c.Locals("user_id").(uuid.UUID)

	if err := me.userService.EditProfile(c.RequestCtx(), userId, name, username, bio); err != nil {
		switch {
		case errors.Is(err, fiber.ErrNotFound):
			return redirect(c, "/login")
		case errors.Is(err, user_service.ErrUsernameConflict):
			return render(c, templates.EditProfileFrom(templates.EditProfileFromParams{
				Name:        name,
				Username:    username,
				UsernameErr: fmt.Errorf("username already exists"),
				Bio:         bio,
			}))
		default:
			return err
		}
	}

	c.Set("HX-Replace-Url", fmt.Sprintf("/@%s", username))
	return render(c, hyper.Group(
		templates.EditProfileFrom(templates.EditProfileFromParams{
			Name:     name,
			Username: username,
			Bio:      bio,
		}),

		hyper.H1(hyper.AttrId("PROFILE_CARD_NAME"), hyper.Attr("hx-swap-oob", "innerHTML"))(name),
		hyper.P(hyper.AttrId("PROFILE_CARD_USERNAME"), hyper.Attr("hx-swap-oob", "innerHTML"))("@"+username),
		hyper.P(hyper.AttrId("PROFILE_CARD_BIO"), hyper.Attr("hx-swap-oob", "innerHTML"))(bio),
		templates.Alert(templates.AlertInfo, "Profile was updated successfully."),
	))
}

// TODO: add a route for this
func (me *Handler) HandleDeleteProfile(c fiber.Ctx) error {
	currentUserId := c.Locals("user_id").(uuid.UUID)

	me.userLock(currentUserId)
	defer me.userUnlock(currentUserId)

	if err := me.userService.DeleteUser(c.RequestCtx(), currentUserId); err != nil {
		if errors.Is(err, user_service.ErrUserNotFound) {
			return redirect(c, "/login")
		}
		return err
	}

	return redirect(c, "/login")
}
