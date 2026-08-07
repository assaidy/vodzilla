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
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/vodzilla/internals/utils/mailer"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v3"
	_ "github.com/joho/godotenv/autoload"
)

// TODO: transcoding

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
	mailer := mailer.New(
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
	historyService := history_service.New(postgres, redis, logger.WithGroup("history service"))

	serviceManager := services.NewManager(logger.WithGroup("service manager"))
	{
		serviceManager.Add("user service", userService)
		serviceManager.Add("video service", videoService)
		serviceManager.Add("media service", mediaService)
		serviceManager.Add("reaction service", reactionService)
		serviceManager.Add("social service", socialService)
		serviceManager.Add("notification service", notificationService)
		serviceManager.Add("history service", historyService)
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
		historyService,
	)
	router := fiber.New(fiber.Config{AppName: "Vodzilla"})
	handler.RegisterRoutes(router)

	quitCtx, quitCtxCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
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
