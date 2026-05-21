package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/assaidy/hyper/v2"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/web/templates"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
)

func HandleHomePage(c fiber.Ctx) error {
	return redirect(c, "/feed")
}

func HandleFeedPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.FeedPage(currentUser.Username))
}

func HandleFeedPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.FeedPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageFeed,
			}),
		),
	))
}

func HandleDiscoverPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.DiscoverPage(currentUser.Username))
}

func HandleDiscoverPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.DiscoverPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageDiscover,
			}),
		),
	))
}

func HandleWatchLaterPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videos, err := videoService.GetAllVideosInWatchLater(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)

	ownerCache := make(map[string]*user_service.User)
	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		owner, ok := ownerCache[v.OwnerId]
		if !ok {
			owner, err = userService.GetUserById(c.RequestCtx(), v.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrNotFound) {
					continue
				}
				return err
			}
			ownerCache[v.OwnerId] = owner
		}

		viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			ViewsCount:    viewsCount,
		})
	}

	return render(c, templates.WatchLaterPage(templates.WatchLaterPageContentParams{
		Username: currentUser.Username,
		Videos:   templateVideos,
	}))
}

func HandleWatchLaterPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videos, err := videoService.GetAllVideosInWatchLater(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)

	ownerCache := make(map[string]*user_service.User)
	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		owner, ok := ownerCache[v.OwnerId]
		if !ok {
			owner, err = userService.GetUserById(c.RequestCtx(), v.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrNotFound) {
					continue
				}
				return err
			}
			ownerCache[v.OwnerId] = owner
		}

		viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			ViewsCount:    viewsCount,
		})
	}

	return render(c, hyper.Group(
		templates.WatchLaterPageContent(templates.WatchLaterPageContentParams{
			Username: currentUser.Username,
			Videos:   templateVideos,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageWatchLater,
			}),
		),
	))
}

func HandlePlaylistsPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	playlists, err := videoService.GetAllPlaylists(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	templatePlaylists := make([]templates.PlaylistCardParams, 0, len(playlists))
	for _, p := range playlists {
		templatePlaylists = append(templatePlaylists, templates.PlaylistCardParams{
			Id:          p.Id,
			Name:        p.Name,
			VideosCount: p.VideosCount,
		})
	}

	return render(c, templates.PlaylistsPage(templates.PlaylistsPageContentParams{
		Username:  currentUser.Username,
		Playlists: templatePlaylists,
	}))
}

func HandlePlaylistsPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	playlists, err := videoService.GetAllPlaylists(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	templatePlaylists := make([]templates.PlaylistCardParams, 0, len(playlists))
	for _, p := range playlists {
		templatePlaylists = append(templatePlaylists, templates.PlaylistCardParams{
			Id:          p.Id,
			Name:        p.Name,
			VideosCount: p.VideosCount,
		})
	}

	return render(c, hyper.Group(
		templates.PlaylistsPageContent(templates.PlaylistsPageContentParams{
			Username:  currentUser.Username,
			Playlists: templatePlaylists,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PagePlaylists,
			}),
		),
	))
}

func HandleNotificationsPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.NotificationsPage(currentUser.Username))
}

func HandleNotificationsPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.NotificationsPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageNotifications,
			}),
		),
	))
}

// TODO: we can do a lot of lazy loading and pagination here and in other places
func HandleProfilePage(c fiber.Ctx) error {
	user, currentUser, err := getProfileUserAndCurrentUser(c)
	if err != nil {
		return err
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videos, err := videoService.GetAllUserVideos(c.RequestCtx(), user.Id)
	if err != nil {
		return err
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)

	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     user.Name,
			OwnerUsername: user.Username,
			ViewsCount:    viewsCount,
		})
	}

	return render(c, templates.ProfilePage(templates.ProfilePageContentParams{
		Username: user.Username,
		Name:     user.Name,
		Bio:      user.Bio,
		IsOwner:  user.Username == currentUser.Username,
		Videos:   templateVideos,
	}))
}

func HandleProfilePageContent(c fiber.Ctx) error {
	user, currentUser, err := getProfileUserAndCurrentUser(c)
	if err != nil {
		return err
	}

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	videos, err := videoService.GetAllUserVideos(c.RequestCtx(), user.Id)
	if err != nil {
		return err
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)

	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     user.Name,
			OwnerUsername: user.Username,
			ViewsCount:    viewsCount,
		})
	}

	return render(c, hyper.Group(
		templates.ProfilePageContent(templates.ProfilePageContentParams{
			Username: user.Username,
			Name:     user.Name,
			Bio:      user.Bio,
			IsOwner:  user.Username == currentUser.Username,
			Videos:   templateVideos,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageProfile,
			}),
		),
	))
}

func getProfileUserAndCurrentUser(c fiber.Ctx) (*user_service.User, *user_service.User, error) {
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	user, err := userService.GetUserByUsername(c.RequestCtx(), c.Params("username"))
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return nil, nil, fiber.ErrNotFound
		}
		return nil, nil, fmt.Errorf("failed to get profile user: %w", err)
	}

	currentUser, err := getCurrentUser(c)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return user, currentUser, nil
}

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]*$`)

func HandleEditProfile(c fiber.Ctx) error {
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

	userId := c.Locals("user_id").(string)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	if err := userService.EditProfile(c.RequestCtx(), userId, name, username, bio); err != nil {
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

func getCurrentUser(c fiber.Ctx) (*user_service.User, error) {
	userId := c.Locals("user_id").(string)
	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)

	user, err := userService.GetUserById(c.RequestCtx(), userId)
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return nil, redirect(c, "/login")
		}
		return nil, err
	}

	return user, nil
}

func HandleVideoPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	videoId := c.Params("video_id")

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	video, err := videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	owner, err := userService.GetUserById(c.RequestCtx(), video.OwnerId)
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	sourceUrl, err := mediaService.GeneratePresignedGetUrl(c.RequestCtx(), video.Id)
	if err != nil {
		return err
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	reactionCounts, err := reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	currentUserReaction, err := reactionService.GetVideoReactionForUser(c.RequestCtx(), videoId, currentUser.Id)
	if err != nil {
		return err
	}

	isInWatchLater, err := videoService.IsInWatchLater(c.RequestCtx(), videoId, currentUser.Id)
	if err != nil {
		return err
	}

	playlists, err := videoService.GetAllPlaylistsWithVideoStatus(c.RequestCtx(), currentUser.Id, videoId)
	if err != nil {
		return err
	}

	templatePlaylists := make([]templates.PlaylistCheckboxParams, 0, len(playlists))
	for _, p := range playlists {
		templatePlaylists = append(templatePlaylists, templates.PlaylistCheckboxParams{
			VideoId:    videoId,
			PlaylistId: p.Id,
			Name:       p.Name,
			Checked:    p.HasVideo,
		})
	}

	return render(c, templates.VideoPage(templates.VideoPageParams{
		Username: currentUser.Username,
		ContentParams: templates.VideoPageContentParams{
			Id:            video.Id,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			SourceUrl:     sourceUrl,
			Title:         video.Title,
			Description:   video.Description,
			Timestamp:     video.Timestamp,
			ViewsCount:    viewsCount,
			ReactionsParams: templates.ReactionsWidgetParams{
				VideoId:       videoId,
				LikesCount:    reactionCounts.Likes,
				DislikesCount: reactionCounts.Dislikes,
				IsLiked:       currentUserReaction.IsLike,
				IsDisliked:    currentUserReaction.IsDislike,
			},
			WatchLaterButtonParams: templates.WatchLaterButtonParams{
				VideoId:  videoId,
				IsActive: isInWatchLater,
			},
			AddToPlaylistModalParams: templates.AddToPlaylistModalParams{
				VideoId:   videoId,
				Playlists: templatePlaylists,
			},
		},
	}))
}

func HandleVideoPageContent(c fiber.Ctx) error {
	videoId := c.Params("video_id")

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	video, err := videoService.GetVideoById(c.RequestCtx(), videoId)
	if err != nil {
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	owner, err := userService.GetUserById(c.RequestCtx(), video.OwnerId)
	if err != nil {
		if errors.Is(err, user_service.ErrNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	mediaService := fiber.MustGetState[*media_service.Service](c.App().State(), media_service.Name)
	sourceUrl, err := mediaService.GeneratePresignedGetUrl(c.RequestCtx(), video.Id)
	if err != nil {
		return err
	}

	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)
	viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	reactionCounts, err := reactionService.GetVideoReactionCounts(c.RequestCtx(), videoId)
	if err != nil {
		return err
	}
	currentUserReaction, err := reactionService.GetVideoReactionForUser(c.RequestCtx(), videoId, currentUser.Id)
	if err != nil {
		return err
	}

	isInWatchLater, err := videoService.IsInWatchLater(c.RequestCtx(), videoId, currentUser.Id)
	if err != nil {
		return err
	}

	playlists, err := videoService.GetAllPlaylistsWithVideoStatus(c.RequestCtx(), currentUser.Id, videoId)
	if err != nil {
		return err
	}

	templatePlaylists := make([]templates.PlaylistCheckboxParams, 0, len(playlists))
	for _, p := range playlists {
		templatePlaylists = append(templatePlaylists, templates.PlaylistCheckboxParams{
			VideoId:    videoId,
			PlaylistId: p.Id,
			Name:       p.Name,
			Checked:    p.HasVideo,
		})
	}

	return render(c, hyper.Group(
		templates.VideoPageContent(templates.VideoPageContentParams{
			Id:            video.Id,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			SourceUrl:     sourceUrl,
			Title:         video.Title,
			Description:   video.Description,
			Timestamp:     video.Timestamp,
			ViewsCount:    viewsCount,
			ReactionsParams: templates.ReactionsWidgetParams{
				VideoId:       videoId,
				LikesCount:    reactionCounts.Likes,
				DislikesCount: reactionCounts.Dislikes,
				IsLiked:       currentUserReaction.IsLike,
				IsDisliked:    currentUserReaction.IsDislike,
			},
			WatchLaterButtonParams: templates.WatchLaterButtonParams{
				VideoId:  videoId,
				IsActive: isInWatchLater,
			},
			AddToPlaylistModalParams: templates.AddToPlaylistModalParams{
				VideoId:   videoId,
				Playlists: templatePlaylists,
			},
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username: currentUser.Username,
			}),
		),
	))
}

func HandlePlaylistDetailPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	playlistId := c.Params("playlist_id")

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	playlist, err := videoService.GetPlaylist(c.RequestCtx(), currentUser.Id, playlistId)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	videos, err := videoService.GetAllVideosInPlaylist(c.RequestCtx(), currentUser.Id, playlistId)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)

	ownerCache := make(map[string]*user_service.User)
	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		owner, ok := ownerCache[v.OwnerId]
		if !ok {
			owner, err = userService.GetUserById(c.RequestCtx(), v.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrNotFound) {
					continue
				}
				return err
			}
			ownerCache[v.OwnerId] = owner
		}

		viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			ViewsCount:    viewsCount,
		})
	}

	return render(c, templates.PlaylistDetailPage(templates.PlaylistDetailPageContentParams{
		Username: currentUser.Username,
		Playlist: templates.PlaylistCardParams{
			Id:          playlist.Id,
			Name:        playlist.Name,
			VideosCount: playlist.VideosCount,
		},
		Videos: templateVideos,
	}))
}

func HandlePlaylistDetailPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	playlistId := c.Params("playlist_id")

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	playlist, err := videoService.GetPlaylist(c.RequestCtx(), currentUser.Id, playlistId)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	videos, err := videoService.GetAllVideosInPlaylist(c.RequestCtx(), currentUser.Id, playlistId)
	if err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.ErrNotFound
		}
		return err
	}

	userService := fiber.MustGetState[*user_service.Service](c.App().State(), user_service.Name)
	reactionService := fiber.MustGetState[*reaction_service.Service](c.App().State(), reaction_service.Name)

	ownerCache := make(map[string]*user_service.User)
	templateVideos := make([]templates.VideoCardParams, 0, len(videos))
	for _, v := range videos {
		owner, ok := ownerCache[v.OwnerId]
		if !ok {
			owner, err = userService.GetUserById(c.RequestCtx(), v.OwnerId)
			if err != nil {
				if errors.Is(err, user_service.ErrNotFound) {
					continue
				}
				return err
			}
			ownerCache[v.OwnerId] = owner
		}

		viewsCount, err := reactionService.GetVideoViewsCount(c.RequestCtx(), v.Id)
		if err != nil {
			return err
		}

		templateVideos = append(templateVideos, templates.VideoCardParams{
			VideoId:       v.Id,
			Title:         v.Title,
			Timestamp:     v.Timestamp,
			OwnerName:     owner.Name,
			OwnerUsername: owner.Username,
			ViewsCount:    viewsCount,
		})
	}

	return render(c, hyper.Group(
		templates.PlaylistDetailPageContent(templates.PlaylistDetailPageContentParams{
			Username: currentUser.Username,
			Playlist: templates.PlaylistCardParams{
				Id:          playlist.Id,
				Name:        playlist.Name,
				VideosCount: playlist.VideosCount,
			},
			Videos: templateVideos,
		}),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PagePlaylists,
			}),
		),
	))
}

func HandleCreatePlaylist(c fiber.Ctx) error {
	name := strings.TrimSpace(c.FormValue("name"))
	videoId := c.FormValue("videoId")

	nameErr := validation.Validate(name, validation.Required, validation.Length(1, 50))

	if nameErr != nil {
		return render(c, templates.CreatePlaylistForm(templates.CreatePlaylistFormParams{
			VideoId: videoId,
			Name:    name,
			NameErr: nameErr,
		}))
	}

	userId := c.Locals("user_id").(string)
	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)

	playlistId, err := videoService.CreatePlaylist(c.RequestCtx(), userId, name)
	if err != nil {
		if errors.Is(err, video_service.ErrConflict) {
			return render(c, templates.CreatePlaylistForm(templates.CreatePlaylistFormParams{
				VideoId: videoId,
				Name:    name,
				NameErr: fmt.Errorf("playlist with this name already exists"),
			}))
		}
		return err
	}

	return render(c, hyper.Group(
		templates.CreatePlaylistForm(templates.CreatePlaylistFormParams{
			VideoId: videoId,
		}),

		hyper.DIV(
			hyper.AttrId("PLAYLIST_CHECKBOX_LIST"),
			hyper.Attr("hx-swap-oob", "beforeend"),
		)(
			templates.PlaylistCheckbox(templates.PlaylistCheckboxParams{
				VideoId:    videoId,
				PlaylistId: playlistId,
				Name:       name,
				Checked:    false,
			}),
		),
	))
}

func HandleAddVideoToPlaylist(c fiber.Ctx) error {
	videoId := c.Params("video_id")
	playlistId := c.Params("playlist_id")
	userId := c.Locals("user_id").(string)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.AddVideoToPlaylist(c.RequestCtx(), videoId, userId, playlistId); err != nil {
		if errors.Is(err, video_service.ErrConflict) {
			return fiber.NewError(fiber.StatusConflict, "already in playlist")
		}
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video not found")
		}
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		}
		return err
	}

	return nil
}

func HandleDeleteVideoFromPlaylist(c fiber.Ctx) error {
	videoId := c.Params("video_id")
	playlistId := c.Params("playlist_id")
	userId := c.Locals("user_id").(string)

	videoService := fiber.MustGetState[*video_service.Service](c.App().State(), video_service.Name)
	if err := videoService.DeleteVideoFromPlaylist(c.RequestCtx(), videoId, userId, playlistId); err != nil {
		if errors.Is(err, video_service.ErrPlaylistNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "playlist not found")
		}
		if errors.Is(err, video_service.ErrVideoNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "video not in playlist")
		}
		return err
	}

	return nil
}
