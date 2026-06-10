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
	feed_service "github.com/assaidy/vodzilla/internals/services/feed"
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
// TODO: use github.com/assaidy/lucide pacakge for lucide icons

// TODO: add pagination to all list endpoints (cursor-based with limit/offset fallback).
// TODO: rethink all cleanup workers and deletion of data. we might need data.
func main() {
	app := fiber.New(fiber.Config{
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
	_ = feed_service.New()
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
	registerRoutes(app, handler)

	quitCtx, quitCtxCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// TODO: Before enabling prefork, make sure to implement Consumer Groups to
		// prevent multiple instances of a service from consuming the same event multiple times.
		port, _ := utils.GetEnv("PORT", "8080")
		if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
			logger.Error("failed to start server", "error", err, "pid", os.Getpid())
			os.Exit(1)
		}
	}()

	<-quitCtx.Done()
	quitCtxCancel()
	logger.Warn("gracefully shutting down server. press Ctrl-c to force shutdown.", "pid", os.Getpid())

	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		logger.Error("failed to shutdown server", "error", err, "pid", os.Getpid())
		os.Exit(1)
	}
}

func registerRoutes(app *fiber.App, h *handlers.Handler) {
	app.Use(h.WithLogging)
	app.Use(h.WithErrorResolver)
	app.Use(h.WithPassClientIdToLocals)

	app.Use(static.New("assets/", static.Config{
		FS:       web.AssetsFS,
		Compress: true,
		ModifyResponse: func(c fiber.Ctx) error {
			c.Set(fiber.HeaderCacheControl, "no-cache, no-store")
			return nil
		},
	}))
	app.Get("/health", h.HandleCheckHealth)

	app.Get("/ws/:client_id", h.WithWebsocketEssentials, websocket.New(h.HandleWebsocket))
	app.Get("/register", h.HandleRegisterPage)
	app.Post("/register", h.HandleRegister)
	app.Get("/login", h.HandleLoginPage)
	app.Post("/login", h.HandleLogin)
	// TODO: app.Post("/verification_email", h.HandleGetVerificationEmail)
	app.Get("/verification_email/sent", h.HandleVerificationEmailSentPage)
	app.Get("/verification_email/verify", h.HandleVerifyEmailPage)

	app.Get("/", h.HandleHomePage)
	app.Get("/feed", h.WithSession, h.HandleFeedPage)
	app.Get("/feed/content", h.WithSession, h.HandleFeedPageContent)
	app.Get("/discover", h.WithSession, h.HandleDiscoverPage)
	app.Get("/discover/content", h.WithSession, h.HandleDiscoverPageContent)
	app.Get("/watch_later", h.WithSession, h.HandleWatchLaterPage)
	app.Get("/watch_later/content", h.WithSession, h.HandleWatchLaterPageContent)
	app.Get("/playlists", h.WithSession, h.HandlePlaylistsPage)
	app.Get("/playlists/content", h.WithSession, h.HandlePlaylistsPageContent)
	app.Get("/notifications", h.WithSession, h.HandleNotificationsPage)
	app.Get("/notifications/content", h.WithSession, h.HandleNotificationsPageContent)
	app.Get("/@:username", h.WithSession, h.HandleProfilePage)
	app.Get("/@:username/content", h.WithSession, h.HandleProfilePageContent)
	app.Put("/profiles", h.WithSession, h.WithCsrfToken, h.HandleEditProfile)
	// TODO: edit account: email, password, delete account

	app.Post("/follow/:id", h.WithSession, h.WithCsrfToken, h.HandleFollow)
	app.Delete("/follow/:id", h.WithSession, h.WithCsrfToken, h.HandleUnfollow)

	app.Post("/videos", h.WithSession, h.WithCsrfToken, h.HandlePostVideo)
	app.Post("/videos/complete_upload", h.WithSession, h.HandleCompleteVideoUpload)
	app.Get("/videos/:video_id", h.WithSession, h.HandleVideoPage)
	app.Get("/videos/:video_id/content", h.WithSession, h.HandleVideoPageContent)
	app.Get("/videos/:video_id/stream_url", h.WithSession, h.HandleGetVideoStreamUrl)
	app.Post("/videos/:video_id/views", h.WithSession, h.WithCsrfToken, h.HandleViewVideo)
	app.Post("/videos/:video_id/reactions", h.WithSession, h.WithCsrfToken, h.HandleAddVideoReaction)
	app.Delete("/videos/:video_id/reactions", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideoReaction)
	app.Post("/videos/:video_id/watch_later", h.WithSession, h.WithCsrfToken, h.HandleAddToWatchLater)
	app.Delete("/videos/:video_id/watch_later", h.WithSession, h.WithCsrfToken, h.HandleDeleteFromWatchLater)

	app.Get("/playlists/:playlist_id", h.WithSession, h.HandlePlaylistDetailPage)
	app.Get("/playlists/:playlist_id/content", h.WithSession, h.HandlePlaylistDetailPageContent)
	app.Post("/playlists", h.WithSession, h.WithCsrfToken, h.HandleCreatePlaylist)
	app.Post("/videos/:video_id/playlists/:playlist_id", h.WithSession, h.WithCsrfToken, h.HandleAddVideoToPlaylist)
	app.Delete("/videos/:video_id/playlists/:playlist_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteVideoFromPlaylist)

	app.Get("/videos/:video_id/comments", h.WithSession, h.HandleGetVideoComments)
	app.Get("/videos/:video_id/comments/:comment_id/replies", h.WithSession, h.HandleGetCommentReplies)
	app.Post("/videos/:video_id/comments", h.WithSession, h.WithCsrfToken, h.HandleCreateComment)
	app.Put("/videos/:video_id/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleEditComment)
	app.Delete("/videos/:video_id/comments/:comment_id", h.WithSession, h.WithCsrfToken, h.HandleDeleteComment)
}
