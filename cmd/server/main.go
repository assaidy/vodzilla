package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assaidy/vodzilla/internals/registry"
	"github.com/assaidy/vodzilla/internals/routes"
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
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v3"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName:      "Vodzilla",
		ErrorHandler: nil, // overriden in logger middelware
	})
	routes.RegisterRoutes(app)

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

	registry := registry.NewRegistry(logger.WithGroup("registry"), app)
	registry.Inject("logger", logger.WithGroup("handler"))
	registry.Inject("redis", redis)
	registry.AddServiceWithInjection(
		user_service.Name,
		user_service.New(postgres, redis, s3, mailer, logger.WithGroup("user service")),
	)
	registry.AddServiceWithInjection(
		video_service.Name,
		video_service.New(postgres, redis, logger.WithGroup("video service")),
	)
	registry.AddServiceWithInjection(
		media_service.Name,
		media_service.New(postgres, s3, redis, logger.WithGroup("media service")),
	)
	registry.AddServiceWithInjection(
		reaction_service.Name,
		reaction_service.New(postgres, logger.WithGroup("reaction service")),
	)
	registry.AddServiceWithInjection(social_service.Name, social_service.New())
	registry.AddServiceWithInjection(search_service.Name, search_service.New())
	registry.AddServiceWithInjection(feed_service.Name, feed_service.New())
	registry.AddServiceWithInjection(history_service.Name, history_service.New())
	registry.AddServiceWithInjection(moderation_service.Name, moderation_service.New())

	registry.Start(context.Background())
	defer registry.Stop(context.Background())

	quitCtx, quitCtxCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
		port, _ := utils.GetEnv("PORT", "8080")
		if err := app.Listen(fmt.Sprintf(":%s", port), fiber.ListenConfig{EnablePrefork: true}); err != nil {
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
