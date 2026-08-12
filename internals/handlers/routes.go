package handlers

import (
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

func (me *Handler) RegisterRoutes(router *fiber.App) {
	router.Use(me.withLogging)
	router.Use(me.withErrorResolver)

	// Misc.
	router.Get("/health", me.handleCheckHealth)
	router.Get("/ws", me.withSession, me.withWebsocketEssentials, websocket.New(me.handleWebsocket))

	// Auth
	router.Post("/auth/register", me.handleRegister)
	router.Post("/auth/login", me.handleLogin)
	router.Post("/auth/logout", me.withSession, me.handleLogout).Name("logout")
	router.Post("/auth/verification_email", me.handleSendVerificationEmail)
	router.Get("/auth/verification_email/verify", me.handleVerifyEmail)
	router.Put("/auth/password", me.withSession, me.withCsrfToken, me.handleEditPassword)

	// Profiles
	router.Get("/profiles/me", me.withSession, me.handleGetProfile)
	router.Get("/profiles", me.withSession, me.handleSearchProfiles)
	router.Get("/profiles/usernames/:username", me.withSession, me.handleGetProfileByUsername)
	router.Get("/profiles/id/:user_id", me.withSession, me.handleGetProfileById)
	router.Put("/profiles", me.withSession, me.withCsrfToken, me.handleEditProfile)
	router.Delete("/profiles", me.withSession, me.withCsrfToken, me.handleDeleteProfile).Name("delete_profile")
	router.Put("/profiles/avatar", me.withSession, me.withCsrfToken, me.handleEditProfileAvatar)
	router.Put("/profiles/avatar/confirm_upload", me.withSession, me.withCsrfToken, me.handleConfirmProfileAvatarUpload)
	router.Delete("/profiles/avatar", me.withSession, me.withCsrfToken, me.handleDeleteProfileAvatar)
	router.Get("/profiles/:user_id/avatar", me.withSession, me.handleGetProfileAvatarUrl)

	// Social
	router.Post("/follows/:user_id", me.withSession, me.withCsrfToken, me.handleFollow)
	router.Delete("/follows/:user_id", me.withSession, me.withCsrfToken, me.handleUnfollow)
	router.Get("/follows/:user_id/counts", me.withSession, me.handleGetFollowCounts)
	router.Get("/follows/:user_id/is_following", me.withSession, me.handleIsFollowing)
	router.Get("/follows/:user_id/followers", me.withSession, me.handleGetFollowers)
	router.Get("/follows/:user_id/followeds", me.withSession, me.handleGetFolloweds)

	// Videos
	router.Post("/videos/upload", me.withSession, me.withCsrfToken, me.handleGenerateVideoUpload)
	router.Put("/videos/upload/confirm", me.withSession, me.withCsrfToken, me.handleConfirmVideoUpload)
	router.Post("/videos", me.withSession, me.withCsrfToken, me.handlePostVideo)
	router.Put("/videos/:video_id/thumbnail", me.withSession, me.withCsrfToken, me.handleEditVideoThumbnail)
	router.Put("/videos/:video_id/thumbnail/confirm_upload", me.withSession, me.withCsrfToken, me.handleConfirmVideoThumbnailUpload)
	router.Delete("/videos/:video_id/thumbnail", me.withSession, me.withCsrfToken, me.handleDeleteVideoThumbnail)
	router.Get("/videos/:video_id/thumbnail", me.withSession, me.handleGetVideoThumbnailUrl)
	router.Get("/videos/:video_id", me.withSession, me.handleGetVideo)
	router.Get("/videos/:video_id/stream_url", me.withSession, me.handleGetVideoStreamUrl)
	router.Delete("/videos/:video_id", me.withSession, me.withCsrfToken, me.handleDeleteVideo)
	router.Get("/videos/users/:user_id", me.withSession, me.handleGetVideosForUser)
	router.Get("/videos/users/:user_id/count", me.withSession, me.handleGetVideosCountForUser)
	router.Get("/videos/", me.withSession, me.handleSearchVideos)

	// Watch Later
	router.Get("/watchlaters", me.withSession, me.handleGetWatchlaters)
	router.Post("/watchlaters/videos/:video_id", me.withSession, me.withCsrfToken, me.handleAddToWatchLaters)
	router.Delete("/watchlaters/videos/:video_id", me.withSession, me.withCsrfToken, me.handleDeleteFromWatchLaters)

	// Playlists
	router.Post("/playlists", me.withSession, me.withCsrfToken, me.handleCreatePlaylist)
	router.Get("/playlists/users/:user_id", me.withSession, me.handleGetUserPlaylists)
	router.Get("/playlists/videos/:video_id", me.withSession, me.handleGetPlaylistsWithVideoStatus)
	router.Get("/playlists/:playlist_id", me.withSession, me.handleGetPlaylist)
	router.Get("/playlists/:playlist_id/videos", me.withSession, me.handleGetPlaylistVideos)
	router.Delete("/playlists/:playlist_id", me.withSession, me.withCsrfToken, me.handleDeletePlaylist)
	router.Put("/playlists/:playlist_id", me.withSession, me.withCsrfToken, me.handleEditPlaylist)
	router.Post("/playlists/:playlist_id/videos/:video_id", me.withSession, me.withCsrfToken, me.handleAddVideoToPlaylist)
	router.Delete("/playlists/:playlist_id/videos/:video_id", me.withSession, me.withCsrfToken, me.handleDeleteVideoFromPlaylist)
	// NOTE: added postfix /list to avoid conflict with GET /playlists/:playlist_id
	router.Get("/playlists/saved/list", me.withSession, me.handleGetSavedPlaylists)
	router.Post("/playlists/saved/:playlist_id", me.withSession, me.withCsrfToken, me.handleAddToSavedPlaylists)
	router.Delete("/playlists/saved/:playlist_id", me.withSession, me.withCsrfToken, me.handleDeleteFromSavedPlaylists)

	// Reactions
	router.Post("/reactions/views/videos/:video_id", me.withSession, me.withCsrfToken, me.handleViewVideo)
	router.Get("/reactions/views/videos/:video_id/count", me.withSession, me.handleGetVideoViewsCount)
	router.Post("/reactions/views/playlists/:playlist_id", me.withSession, me.withCsrfToken, me.handleViewPlaylist)
	router.Get("/reactions/views/playlists/:playlist_id/count", me.withSession, me.handleGetPlaylistViewsCount)
	router.Post("/reactions/comments/videos/:video_id", me.withSession, me.withCsrfToken, me.handleCreateVideoComment)
	router.Get("/reactions/comments/videos/:video_id", me.withSession, me.handleGetVideoComments)
	router.Post("/reactions/comments/:comment_id/replies", me.withSession, me.withCsrfToken, me.handleCreateCommentReply)
	router.Get("/reactions/comments/:comment_id/replies", me.withSession, me.handleGetCommentReplies)
	router.Put("/reactions/comments/:comment_id", me.withSession, me.withCsrfToken, me.handleEditComment)
	router.Delete("/reactions/comments/:comment_id", me.withSession, me.withCsrfToken, me.handleDeleteComment)
	router.Post("/reactions/feelings/videos/:video_id", me.withSession, me.withCsrfToken, me.handleAddVideoFeeling)
	router.Delete("/reactions/feelings/videos/:video_id", me.withSession, me.withCsrfToken, me.handleDeleteVideoFeeling)
	router.Get("/reactions/feelings/videos/:video_id/counts", me.withSession, me.handleGetVideoFeelingCounts)
	router.Get("/reactions/feelings/videos/:video_id/user", me.withSession, me.handleGetVideoFeelingForCurrentUser)
	router.Post("/reactions/feelings/comments/:comment_id", me.withSession, me.withCsrfToken, me.handleAddCommentFeeling)
	router.Delete("/reactions/feelings/comments/:comment_id", me.withSession, me.withCsrfToken, me.handleDeleteCommentFeeling)
	router.Get("/reactions/feelings/comments/:comment_id/counts", me.withSession, me.handleGetCommentFeelingCounts)
	router.Get("/reactions/feelings/comments/:comment_id/user", me.withSession, me.handleGetCommentFeelingForCurrentUser)

	// Feed
	router.Get("/feed", me.withSession, me.handleGetFeed)

	// Notifications
	router.Get("/notifications/notifications", me.withSession, me.handleGetNotifications)
	router.Get("/notifications/notifications/count", me.withSession, me.handleGetUnreadNotificationsCount)
	router.Post("/notifications/:notification_id/mark_read", me.withSession, me.withCsrfToken, me.handleMarkNotificationAsRead)

	// History
	router.Get("/history", me.withSession, me.handleGetWatchHistory)
	router.Post("/history/videos/:video_id", me.withSession, me.withCsrfToken, me.handleAddToWatchHistory)
	router.Delete("/history/:entry_id", me.withSession, me.withCsrfToken, me.handleDeleteWatchHistoryEntry)
	router.Delete("/history", me.withSession, me.withCsrfToken, me.handleClearWatchHistory)

	// TODO: recommendations
}
