package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assaidy/video_streaming_app/internals/handlers"
	"github.com/assaidy/video_streaming_app/internals/routes"
	"github.com/assaidy/video_streaming_app/internals/services"
	"github.com/assaidy/video_streaming_app/internals/services/feed"
	"github.com/assaidy/video_streaming_app/internals/services/history"
	"github.com/assaidy/video_streaming_app/internals/services/media"
	"github.com/assaidy/video_streaming_app/internals/services/moderation"
	"github.com/assaidy/video_streaming_app/internals/services/reaction"
	"github.com/assaidy/video_streaming_app/internals/services/search"
	"github.com/assaidy/video_streaming_app/internals/services/social"
	"github.com/assaidy/video_streaming_app/internals/services/user"
	"github.com/assaidy/video_streaming_app/internals/services/video"
	"github.com/assaidy/video_streaming_app/internals/utils"
	"github.com/assaidy/video_streaming_app/internals/utils/mailer"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v3"
	_ "github.com/joho/godotenv/autoload"
)

// TODO: put all config in a package?

func main() {
	logger := slog.New(log.NewWithOptions(os.Stderr, log.Options{
		Formatter:       log.TextFormatter,
		ReportTimestamp: true,
	}))
	postgresConnection := utils.ConnectToPostgres(utils.MustGetEnv("POSTGRES_URL"))
	redisConectionn := utils.ConnectToRedis(utils.MustGetEnv("REDIS_ADDR"))
	mailer := mailer.New(
		utils.MustGetEnv("PAPERCUT_HOST"),
		utils.MustGetEnv("PAPERCUT_PORT"),
		utils.MustGetEnv("PAPERCUT_USERNAME"),
		utils.MustGetEnv("PAPERCUT_PASSWORD"),
	)

	userService := user.New(postgresConnection, redisConectionn, mailer, logger)
	videoService := video.New()
	mediaService := media.New()
	reactionService := reaction.New()
	socialService := social.New()
	searchService := search.New()
	feedService := feed.New()
	historyService := history.New()
	moderationService := moderation.New()

	registry := services.NewRegistry(logger)
	registry.Add(user.Name, userService)
	registry.Add(video.Name, videoService)
	registry.Add(media.Name, mediaService)
	registry.Add(reaction.Name, reactionService)
	registry.Add(social.Name, socialService)
	registry.Add(search.Name, searchService)
	registry.Add(feed.Name, feedService)
	registry.Add(history.Name, historyService)
	registry.Add(moderation.Name, moderationService)

	registry.Start(context.Background())
	defer registry.Stop(context.Background())

	app := fiber.New(fiber.Config{
		AppName:      "video_streaming_app",
		ErrorHandler: handlers.ErrorHandler,
	})

	app.State().Set("logger", logger)
	app.State().Set("redis", redisConectionn)
	app.State().Set(user.Name, userService)

	quitCtx, quitCtxCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
		routes.RegisterRoutes(app)

		port, _ := utils.GetEnv("PORT", "8080")
		if err := app.Listen(fmt.Sprintf(":%s", port), fiber.ListenConfig{
			EnablePrefork: true,
		}); err != nil {
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
