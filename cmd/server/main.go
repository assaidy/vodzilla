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
	"github.com/assaidy/vodzilla/internals/registry"
	"github.com/assaidy/vodzilla/internals/routes"
	"github.com/assaidy/vodzilla/internals/services/feed"
	"github.com/assaidy/vodzilla/internals/services/history"
	"github.com/assaidy/vodzilla/internals/services/media"
	"github.com/assaidy/vodzilla/internals/services/moderation"
	"github.com/assaidy/vodzilla/internals/services/reaction"
	"github.com/assaidy/vodzilla/internals/services/search"
	"github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	"github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils"
	"github.com/assaidy/vodzilla/internals/utils/mailer"
	"github.com/charmbracelet/log"
	"github.com/gofiber/fiber/v3"
	_ "github.com/joho/godotenv/autoload"
)

// TODO: put all config in a package?

func main() {
	app := fiber.New(fiber.Config{
		AppName:      "Vodzilla",
		ErrorHandler: handlers.ErrorHandler,
	})
	routes.RegisterRoutes(app)

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

	registry := registry.NewRegistry(logger, app)
	registry.Inject("logger", logger)
	registry.Inject("redis", redisConectionn)
	registry.AddServiceWithInjection(user_service.Name, user_service.New(postgresConnection, redisConectionn, mailer, logger))
	registry.AddServiceWithInjection(video.Name, video.New())
	registry.AddServiceWithInjection(media.Name, media.New())
	registry.AddServiceWithInjection(reaction.Name, reaction.New())
	registry.AddServiceWithInjection(social.Name, social.New())
	registry.AddServiceWithInjection(search.Name, search.New())
	registry.AddServiceWithInjection(feed.Name, feed.New())
	registry.AddServiceWithInjection(history.Name, history.New())
	registry.AddServiceWithInjection(moderation.Name, moderation.New())

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
