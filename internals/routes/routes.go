package routes

import (
	"github.com/assaidy/video_streaming_app/internals/handlers"
	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App) {
	app.Use(handlers.WithLogging)

	app.Get("/health", handlers.HandleCheckHealth)
}
