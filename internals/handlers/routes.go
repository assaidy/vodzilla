package handlers

import (
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

func (me *Handler) RegisterRoutes(router *fiber.App) {
	router.Use(me.WithLogging)
	router.Use(me.WithErrorResolver)

	// Misc.
	router.Get("/health", me.HandleCheckHealth)
	router.Get("/ws", me.WithSession, me.WithWebsocketEssentials, websocket.New(me.HandleWebsocket))

	// Auth
	router.Post("/auth/register", me.HandleRegister)
	router.Post("/auth/login", me.HandleLogin)
	router.Post("/auth/logout", me.WithSession, me.HandleLogout).Name("logout")
	router.Post("/auth/verification_email", me.HandleSendVerificationEmail)
	router.Get("/auth/verification_email/verify", me.HandleVerifyEmail)
	router.Put("/auth/credentials", me.WithSession, me.WithCsrfToken, me.HandleEditCredentials)

	// Profiles
	router.Get("/profiles", me.WithSession, me.HandleGetProfile)
	router.Get("/profiles/usernames/:username", me.WithSession, me.HandleGetProfileByUsername)
	router.Get("/profiles/id/:user_id", me.WithSession, me.HandleGetProfileById)
	router.Put("/profiles", me.WithSession, me.WithCsrfToken, me.HandleEditProfile)
	router.Delete("/profiles", me.WithSession, me.WithCsrfToken, me.HandleDeleteProfile).Name("delete_profile")
	router.Put("/profiles/avatar", me.WithSession, me.WithCsrfToken, me.HandleEditProfileAvatar)
	router.Put("/profiles/avatar/confirm_upload", me.WithSession, me.WithCsrfToken, me.HandleConfirmProfileAvatarUpload)
	router.Delete("/profiles/avatar", me.WithSession, me.WithCsrfToken, me.HandleDeleteProfileAvatar)
	router.Get("/profiles/:user_id/avatar", me.WithSession, me.HandleGetProfileAvatarUrl)

	// Social
	router.Post("/follows/:user_id", me.WithSession, me.WithCsrfToken, me.HandleFollow)
	router.Delete("/follows/:user_id", me.WithSession, me.WithCsrfToken, me.HandleUnfollow)
	router.Get("/follows/:user_id/counts", me.WithSession, me.HandleGetFollowCounts)
	router.Get("/follows/:user_id/is_following", me.WithSession, me.HandleIsFollowing)
	router.Get("/follows/:user_id/followers", me.WithSession, me.HandleGetFollowers)
	router.Get("/follows/:user_id/followeds", me.WithSession, me.HandleGetFolloweds)

	// Videos
	router.Post("/videos/upload", me.WithSession, me.WithCsrfToken, me.HandleGenerateVideoUpload)
	router.Put("/videos/upload/confirm", me.WithSession, me.WithCsrfToken, me.HandleConfirmVideoUpload)
	router.Post("/videos", me.WithSession, me.WithCsrfToken, me.HandlePostVideo)
	router.Put("/videos/:video_id/thumbnail", me.WithSession, me.WithCsrfToken, me.HandleEditVideoThumbnail)
	router.Put("/videos/:video_id/thumbnail/confirm_upload", me.WithSession, me.WithCsrfToken, me.HandleConfirmVideoThumbnailUpload)
	router.Delete("/videos/:video_id/thumbnail", me.WithSession, me.WithCsrfToken, me.HandleDeleteVideoThumbnail)
	router.Get("/videos/:video_id/thumbnail", me.WithSession, me.HandleGetVideoThumbnailUrl)
	router.Get("/videos/:video_id", me.WithSession, me.HandleGetVideo)
	router.Get("/videos/:video_id/stream_url", me.WithSession, me.HandleGetVideoStreamUrl)
	router.Delete("/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleDeleteVideo)
	router.Get("/videos/users/:user_id", me.WithSession, me.HandleGetVideosForUser)
	router.Get("/videos/users/:user_id/count", me.WithSession, me.HandleGetVideosCountForUser)

	// Watch Later
	router.Get("/watchlaters", me.WithSession, me.HandleGetWatchlaters)
	router.Post("/watchlaters/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleAddToWatchLaters)
	router.Delete("/watchlaters/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleDeleteFromWatchLaters)

	// Playlists
	router.Post("/playlists", me.WithSession, me.WithCsrfToken, me.HandleCreatePlaylist)
	router.Get("/playlists/users/:user_id", me.WithSession, me.HandleGetPlaylists)
	router.Get("/playlists/users/:user_id/videos/:video_id", me.WithSession, me.HandleGetPlaylistsWithVideoStatus)
	router.Get("/playlists/:playlist_id", me.WithSession, me.HandleGetPlaylist)
	router.Get("/playlists/:playlist_id/videos", me.WithSession, me.HandleGetPlaylistVideos)
	router.Delete("/playlists/:playlist_id", me.WithSession, me.WithCsrfToken, me.HandleDeletePlaylist)
	router.Put("/playlists/:playlist_id", me.WithSession, me.WithCsrfToken, me.HandleRenamePlaylist)
	router.Post("/playlists/:playlist_id/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleAddVideoToPlaylist)
	router.Delete("/playlists/:playlist_id/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleDeleteVideoFromPlaylist)
	// TODO: add playlist description
	// TODO: add playlist bookmarking(saving)
	// TODO: add playlist visibility (is_public)

	// Reactions
	router.Post("/reactions/views/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleViewVideo)
	router.Get("/reactions/views/videos/:video_id/count", me.WithSession, me.HandleGetVideoViewsCount)
	router.Post("/reactions/views/playlists/:playlist_id", me.WithSession, me.WithCsrfToken, me.HandleViewPlaylist)
	router.Get("/reactions/views/playlists/:playlist_id/count", me.WithSession, me.HandleGetPlaylistViewsCount)
	router.Post("/reactions/comments/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleCreateVideoComment)
	router.Get("/reactions/comments/videos/:video_id", me.WithSession, me.HandleGetVideoComments)
	router.Post("/reactions/comments/:comment_id/replies", me.WithSession, me.WithCsrfToken, me.HandleCreateCommentReply)
	router.Get("/reactions/comments/:comment_id/replies", me.WithSession, me.HandleGetCommentReplies)
	router.Put("/reactions/comments/:comment_id", me.WithSession, me.WithCsrfToken, me.HandleEditComment)
	router.Delete("/reactions/comments/:comment_id", me.WithSession, me.WithCsrfToken, me.HandleDeleteComment)
	router.Post("/reactions/feelings/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleAddVideoFeeling)
	router.Delete("/reactions/feelings/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleDeleteVideoFeeling)
	router.Post("/reactions/feelings/comments/:comment_id", me.WithSession, me.WithCsrfToken, me.HandleAddCommentFeeling)
	router.Delete("/reactions/feelings/comments/:comment_id", me.WithSession, me.WithCsrfToken, me.HandleDeleteCommentFeeling)

	// Feed
	router.Get("/feed", me.WithSession, me.HandleGetFeed)

	// Notifications
	router.Get("/notifications/notifications", me.WithSession, me.HandleGetNotifications)
	router.Get("/notifications/notifications/count", me.WithSession, me.HandleGetUnreadNotificationsCount)
	router.Post("/notifications/:notification_id/mark_read", me.WithSession, me.WithCsrfToken, me.HandleMarkNotificationAsRead)

	// History
	router.Get("/history", me.WithSession, me.HandleGetWatchHistory)
	router.Post("/history/videos/:video_id", me.WithSession, me.WithCsrfToken, me.HandleAddToWatchHistory)
	router.Delete("/history/:entry_id", me.WithSession, me.WithCsrfToken, me.HandleDeleteWatchHistoryEntry)
	router.Delete("/history", me.WithSession, me.WithCsrfToken, me.HandleClearWatchHistory)

	// TODO: search, recommendations
}
