package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	redis_pubsub "github.com/assaidy/pubsubs/redis"
	"github.com/assaidy/video_streaming_app/internals/handlers"
	"github.com/assaidy/video_streaming_app/internals/routes"
	"github.com/assaidy/video_streaming_app/internals/services"
	"github.com/assaidy/video_streaming_app/internals/services/auth"
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
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		Formatter:       log.TextFormatter,
		ReportTimestamp: true,
	})
	postgresConnection := connectToPostgres(utils.MustGetEnv("POSTGRES_URL"))
	// redisConectionn := connectToRedis(utils.MustGetEnv("REDIS_ADDR"))
	// redisPubsub := redis_pubsub.New(redisConectionn)
	mailer := mailer.New(
		utils.MustGetEnv("PAPERCUT_HOST"),
		utils.MustGetEnv("PAPERCUT_PORT"),
		utils.MustGetEnv("PAPERCUT_USERNAME"),
		utils.MustGetEnv("PAPERCUT_PASSWORD"),
	)

	authService := auth.New(postgresConnection, mailer)
	userService := user.New()
	videoService := video.New()
	mediaService := media.New()
	reactionService := reaction.New()
	socialService := social.New()
	searchService := search.New()
	feedService := feed.New()
	historyService := history.New()
	moderationService := moderation.New()

	registry := services.NewRegistry(logger)
	registry.Add(auth.Name, authService)
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
	app.State().Set(auth.Name, authService)

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		routes.RegisterRoutes(app)

		port, _ := utils.GetEnv("PORT", "8080")
		if err := app.Listen(fmt.Sprintf(":%s", port), fiber.ListenConfig{
			EnablePrefork: true,
		}); err != nil {
			logger.Fatal("failed to start server", "error", err)
		}
	}()

	<-quitChan
	logger.Info("gracefully shutting down server. press Ctrl-c to force shutdown.")

	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		logger.Fatal("failed to shutdown server", "error", err)
	}
}

func connectToPostgres(url string) *sql.DB {
	pool, err := sql.Open("postgres", url)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		panic(err)
	}

	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(5 * time.Minute)
	pool.SetConnMaxIdleTime(1 * time.Minute)

	return pool
}

func connectToRedis(addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	return client
}
