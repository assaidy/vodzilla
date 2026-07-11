package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assaidy/vodzilla/internals/handlers"
	"github.com/assaidy/vodzilla/internals/services"
	history_service "github.com/assaidy/vodzilla/internals/services/history"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	search_service "github.com/assaidy/vodzilla/internals/services/search"
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/charmbracelet/log"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	_ "github.com/joho/godotenv/autoload"
)

// TODO: testing
// TODO: use a proper message queue for events (kafka/rmq)
// TODO: transcoding
// TODO: monitoring

func main() {
	logger := slog.New(log.NewWithOptions(os.Stderr, log.Options{
		Formatter:       log.TextFormatter,
		ReportTimestamp: true,
	}))
	postgres := utils.ConnectToPostgres(utils.MustGetEnv("POSTGRES_URL"))
	redis := utils.ConnectToRedis(utils.MustGetEnv("REDIS_ADDR"))
	s3 := utils.ConnectToS3(
		utils.MustGetEnv("RUSTFS_URL"),
		utils.MustGetEnv("RUSTFS_ACCESS_KEY"),
		utils.MustGetEnv("RUSTFS_SECRET_KEY"),
	)
	mailer := utils.NewMailer(
		utils.MustGetEnv("PAPERCUT_HOST"),
		utils.MustGetEnv("PAPERCUT_PORT"),
		utils.MustGetEnv("PAPERCUT_USERNAME"),
		utils.MustGetEnv("PAPERCUT_PASSWORD"),
	)

	userService := user_service.New(postgres, redis, s3, mailer, logger.WithGroup("user service"))
	videoService := video_service.New(postgres, redis, logger.WithGroup("video service"))
	mediaService := media_service.New(postgres, redis, s3, logger.WithGroup("media service"))
	reactionService := reaction_service.New(postgres, redis, logger.WithGroup("reaction service"))
	socialService := social_service.New(postgres, redis, logger.WithGroup("social service"))
	notificationService := notification_service.New(postgres, redis, logger.WithGroup("notification service"))
	_ = search_service.New()
	_ = history_service.New()

	serviceManager := services.NewManager(logger.WithGroup("service manager"))
	{
		serviceManager.Add("user service", userService)
		serviceManager.Add("video service", videoService)
		serviceManager.Add("media service", mediaService)
		serviceManager.Add("reaction service", reactionService)
		serviceManager.Add("social service", socialService)
		serviceManager.Add("notification service", notificationService)
	}
	serviceManager.StartAll()
	defer serviceManager.StopAll()

	handler := handlers.New(
		logger.WithGroup("handler"),
		redis,
		userService,
		videoService,
		mediaService,
		reactionService,
		socialService,
		notificationService,
	)
	router := fiber.New(fiber.Config{AppName: "Vodzilla"})
	registerRoutes(router, handler)

	quitCtx, quitCtxCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// TODO: Before enabling prefork, make sure to implement Consumer Groups to
		// prevent multiple instances of a service from consuming the same event multiple times.
		port, _ := utils.GetEnv("PORT", "8080")
		if err := router.Listen(fmt.Sprintf(":%s", port)); err != nil {
			logger.Error("failed to start server", "error", err, "pid", os.Getpid())
			os.Exit(1)
		}
	}()

	<-quitCtx.Done()
	quitCtxCancel()
	logger.Warn("gracefully shutting down server. press Ctrl-c to force shutdown.", "pid", os.Getpid())

	if err := router.ShutdownWithTimeout(5 * time.Second); err != nil {
		logger.Error("failed to shutdown server", "error", err, "pid", os.Getpid())
		os.Exit(1)
	}
}

func registerRoutes(router *fiber.App, h *handlers.Handler) {
	router.Use(h.WithLogging)
	router.Use(h.WithErrorResolver)

	// Misc.
	router.Get("/health", h.HandleCheckHealth)
	router.Get("/ws", h.WithSession, h.WithWebsocketEssentials, websocket.New(h.HandleWebsocket))

	// Auth
	router.Post("/auth/register", h.HandleRegister)
	router.Post("/auth/login", h.HandleLogin)
	router.Post("/auth/logout", h.WithSession, h.HandleLogout).Name("logout")
	router.Post("/auth/verification_email", h.HandleSendVerificationEmail)
	router.Post("/auth/verification_email/verify", h.HandleVerifyEmail)
	router.Put("/auth/credentials", h.HandleEditCredentials)

	// Profiles
	router.Get("/profiles", h.WithSession, h.HandleGetProfile)
	router.Get("/profiles/usernames/:username", h.WithSession, h.HandleGetProfileByUsername)
	router.Get("/profiles/id/:user_id", h.WithSession, h.HandleGetProfileById)
	router.Put("/profiles", h.WithSession, h.WithCsrfToken, h.HandleEditProfile)
	router.Delete("/profiles", h.WithSession, h.WithCsrfToken, h.HandleDeleteProfile).Name("delete_profile")
	router.Put("/profiles/avatar", h.WithSession, h.WithCsrfToken, h.HandleEditProfileAvatar)
	router.Put("/profiles/avatar/confirm_upload", h.WithSession, h.WithCsrfToken, h.HandleConfirmProfileAvatarUpload)
	router.Delete("/profiles/avatar", h.WithSession, h.WithCsrfToken, h.HandleDeleteProfileAvatar)

	// Social
	router.Post("/follows/:user_id", h.WithSession, h.WithCsrfToken, h.HandleFollow)
	router.Delete("/follows/:user_id", h.WithSession, h.WithCsrfToken, h.HandleUnfollow)
	router.Get("/follows/:user_id/counts", h.WithSession, h.HandleGetFollowCounts)
	router.Get("/follows/:user_id/followers", h.WithSession, h.HandleGetFollowers)
	router.Get("/follows/:user_id/followeds", h.WithSession, h.HandleGetFolloweds)

	// Videos
	router.Post("/videos/upload", h.WithSession, h.WithCsrfToken, h.HandleGenerateVideoUpload)
	router.Put("/videos/upload/confirm", h.WithSession, h.WithCsrfToken, h.HandleConfirmVideoUpload)
	router.Post("/videos", h.WithSession, h.WithCsrfToken, h.HandlePostVideo)
	router.Put("/videos/:video_id/thumbnail", h.WithSession, h.WithCsrfToken, h.HandleEditVideoThumbnail)
	router.Put("/videos/:video_id/thumbnail/confirm_upload", h.WithSession, h.WithCsrfToken, h.HandleConfirmVideoThumbnailUpload)
	router.Delete("/videos/:video_id/thumbnail", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideoThumbnail)
	router.Get("/videos/:video_id", h.WithSession, h.HandleGetVideo)
	router.Get("/videos/:video_id/stream_url", h.WithSession, h.HandleGetVideoStreamUrl)
	router.Delete("/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideo)
	router.Get("/videos/users/:user_id", h.WithSession, h.HandleGetVideosForUser)
	router.Get("/videos/users/:user_id/count", h.WithSession, h.HandleGetVideosCountForUser)

	// Watch Later
	router.Get("/watchlaters", h.WithSession, h.HandleGetWatchlaters)
	router.Post("/watchlaters/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleAddToWatchLaters)
	router.Delete("/watchlaters/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteFromWatchLaters)

	// Playlists
	router.Post("/playlists", h.WithSession, h.WithCsrfToken, h.HandleCreatePlaylist)
	router.Get("/playlists/users/:user_id", h.WithSession, h.HandleGetPlaylists)
	router.Get("/playlists/users/:user_id/videos/:video_id", h.WithSession, h.HandleGetPlaylistsWithVideoStatus)
	router.Get("/playlists/:playlist_id", h.WithSession, h.HandleGetPlaylist)
	router.Get("/playlists/:playlist_id/videos", h.WithSession, h.HandleGetPlaylistVideos)
	router.Delete("/playlists/:playlist_id", h.WithSession, h.WithCsrfToken, h.HandleDeletePlaylist)
	router.Put("/playlists/:playlist_id", h.WithSession, h.WithCsrfToken, h.HandleRenamePlaylist)
	router.Post("/playlists/:playlist_id/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleAddVideoToPlaylist)
	router.Delete("/playlists/:playlist_id/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideoFromPlaylist)

	// Reactions
	router.Post("/reactions/views/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleViewVideo)
	router.Post("/reactions/comments/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleCreateVideoComment)
	router.Get("/reactions/comments/videos/:video_id", h.WithSession, h.HandleGetVideoComments)
	router.Post("/reactions/comments/:comment_id/replies", h.WithSession, h.WithCsrfToken, h.HandleCreateCommentReply)
	router.Get("/reactions/comments/:comment_id/replies", h.WithSession, h.HandleGetCommentReplies)
	router.Put("/reactions/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleEditComment)
	router.Delete("/reactions/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteComment)
	router.Post("/reactions/feelings/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleAddVideoFeeling)
	router.Delete("/reactions/feelings/videos/:video_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideoFeeling)
	router.Post("/reactions/feelings/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleAddCommentFeeling)
	router.Delete("/reactions/feelings/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteCommentFeeling)

	// Feed
	router.Get("/feed", h.WithSession, h.HandleGetFeed)

	// Notifications
	router.Get("/notifications/notifications", h.WithSession, h.HandleGetNotifications)
	router.Get("/notifications/notifications/count", h.WithSession, h.HandleGetUnreadNotificationsCount)
	router.Post("/notifications/:notification_id/mark_read", h.WithSession, h.WithCsrfToken, h.HandleMarkNotificationAsRead)

	// TODO: search, recommendations, history
}
