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
	moderation_service "github.com/assaidy/vodzilla/internals/services/moderation"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	search_service "github.com/assaidy/vodzilla/internals/services/search"
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/vodzilla/internals/utils/mailer"
	"github.com/assaidy/vodzilla/internals/web"
	"github.com/charmbracelet/log"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	_ "github.com/joho/godotenv/autoload"
)

// TODO: add profile avatar upload

// TODO: add pagination to all list endpoints (cursor-based with limit/offset fallback).
// TODO: rethink all cleanup workers and deletion of data. we might need data.
func main() {
	router := fiber.New(fiber.Config{
		AppName: "Vodzilla",
		// We handler errors with [handlers.Handler.WithErrorResolver].
		ErrorHandler: nil,
	})

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
	mailer := mailer.New(
		utils.MustGetEnv("PAPERCUT_HOST"),
		utils.MustGetEnv("PAPERCUT_PORT"),
		utils.MustGetEnv("PAPERCUT_USERNAME"),
		utils.MustGetEnv("PAPERCUT_PASSWORD"),
	)

	serviceManager := services.NewManager(logger.WithGroup("service manager"))

	userService := user_service.New(postgres, redis, s3, mailer, logger.WithGroup("user service"))
	serviceManager.Add("user service", userService)
	videoService := video_service.New(postgres, redis, logger.WithGroup("video service"))
	serviceManager.Add("video service", videoService)
	mediaService := media_service.New(postgres, redis, s3, logger.WithGroup("media service"))
	serviceManager.Add("media service", mediaService)
	reactionService := reaction_service.New(postgres, redis, logger.WithGroup("reaction service"))
	serviceManager.Add("reaction service", reactionService)
	socialService := social_service.New(postgres, redis, logger.WithGroup("social service"))
	serviceManager.Add("social service", socialService)
	_ = search_service.New()
	_ = history_service.New()
	_ = moderation_service.New()

	serviceManager.StartAll(context.Background())
	defer serviceManager.StopAll(context.Background())

	handler := handlers.New(
		logger.WithGroup("handler"),
		redis,
		userService,
		videoService,
		mediaService,
		reactionService,
		socialService,
	)
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
	router.Use(h.WithPassClientIdToLocals)

	router.Use(static.New("assets/", static.Config{
		FS:       web.AssetsFS,
		Compress: true,
		ModifyResponse: func(c fiber.Ctx) error {
			c.Set(fiber.HeaderCacheControl, "no-cache, no-store")
			return nil
		},
	}))
	router.Get("/health", h.HandleCheckHealth)

	router.Get("/", h.HandleHomePage)

	router.Get("/register", h.HandleRegisterPage)
	router.Post("/register", h.HandleRegister)
	router.Get("/login", h.HandleLoginPage)
	router.Post("/login", h.HandleLogin)
	// TODO: app.Post("/verification_email", h.HandleGetVerificationEmail)
	router.Get("/verification_email/sent", h.HandleVerificationEmailSentPage)
	router.Get("/verification_email/verify", h.HandleVerifyEmailPage)

	router.Get("/@:username", h.WithSession, h.HandleProfilePage)
	router.Get("/@:username/content", h.WithSession, h.HandleProfilePageContent)
	router.Put("/profiles", h.WithSession, h.WithCsrfToken, h.HandleEditProfile)
	// TODO: edit account: email, password, delete account

	router.Get("/discover", h.WithSession, h.HandleDiscoverPage)
	router.Get("/discover/content", h.WithSession, h.HandleDiscoverPageContent)
	router.Get("/notifications", h.WithSession, h.HandleNotificationsPage)
	router.Get("/notifications/content", h.WithSession, h.HandleNotificationsPageContent)

	router.Get("/feed", h.WithSession, h.HandleFeedPage)
	router.Get("/feed/content", h.WithSession, h.HandleFeedPageContent)
	router.Post("/follow/:id", h.WithSession, h.WithCsrfToken, h.HandleFollow)
	router.Delete("/follow/:id", h.WithSession, h.WithCsrfToken, h.HandleUnfollow)

	router.Post("/videos", h.WithSession, h.WithCsrfToken, h.HandlePostVideo)
	router.Post("/videos/complete_upload", h.WithSession, h.HandleCompleteVideoUpload)
	router.Get("/videos/:video_id", h.WithSession, h.HandleVideoPage)
	router.Get("/videos/:video_id/content", h.WithSession, h.HandleVideoPageContent)
	router.Get("/videos/:video_id/stream_url", h.WithSession, h.HandleGetVideoStreamUrl)
	router.Post("/videos/:video_id/views", h.WithSession, h.WithCsrfToken, h.HandleViewVideo)
	router.Post("/videos/:video_id/reactions", h.WithSession, h.WithCsrfToken, h.HandleAddVideoReaction)
	router.Delete("/videos/:video_id/reactions", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideoReaction)

	router.Get("/watchlater", h.WithSession, h.HandleWatchLaterPage)
	router.Get("/watchlater/content", h.WithSession, h.HandleWatchLaterPageContent)
	router.Post("/videos/:video_id/watchlater", h.WithSession, h.WithCsrfToken, h.HandleAddToWatchLater)
	router.Delete("/videos/:video_id/watchlater", h.WithSession, h.WithCsrfToken, h.HandleDeleteFromWatchLater)

	router.Get("/playlists", h.WithSession, h.HandlePlaylistsPage)
	router.Get("/playlists/content", h.WithSession, h.HandlePlaylistsPageContent)
	router.Get("/playlists/:playlist_id", h.WithSession, h.HandlePlaylistDetailPage)
	router.Get("/playlists/:playlist_id/content", h.WithSession, h.HandlePlaylistDetailPageContent)
	router.Post("/playlists", h.WithSession, h.WithCsrfToken, h.HandleCreatePlaylist)
	router.Post("/videos/:video_id/playlists/:playlist_id", h.WithSession, h.WithCsrfToken, h.HandleAddVideoToPlaylist)
	router.Delete("/videos/:video_id/playlists/:playlist_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideoFromPlaylist)

	router.Get("/videos/:video_id/comments", h.WithSession, h.HandleGetVideoComments)
	router.Get("/videos/:video_id/comments/:comment_id/replies", h.WithSession, h.HandleGetCommentReplies)
	router.Post("/videos/:video_id/comments", h.WithSession, h.WithCsrfToken, h.HandleCreateComment)
	router.Put("/videos/:video_id/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleEditComment)
	router.Delete("/videos/:video_id/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteComment)

	router.Get("/ws/:client_id", h.WithSession, h.WithWebsocketEssentials, websocket.New(h.HandleWebsocket))
}
