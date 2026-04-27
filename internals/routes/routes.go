package routes

import (
	"github.com/assaidy/video_streaming_app/internals/handlers"
	"github.com/assaidy/video_streaming_app/internals/web"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func RegisterRoutes(app *fiber.App) {
	app.Use(handlers.WithLogging)
	app.Use(static.New("assets/", static.Config{FS: web.AssetsFS, MaxAge: 0, Compress: true}))

	app.Get("/health", handlers.HandleCheckHealth)
	app.Get("/register", handlers.HandleRegisterPage)
	app.Post("/register", handlers.HandleRegister)
	app.Get("/login", handlers.HandleLoginPage)
	app.Post("/login", handlers.HandleLogin)
	// app.Post("/verification_email", handlers.HandleGetVerificationEmail) // TODO:
	app.Get("/verification_email/sent", handlers.HandleVerificationEmailSentPage)
	app.Get("/verification_email/verify", handlers.HandleVerifyEmailPage)

	app.Get("/", handlers.WithSessionToken, handlers.HandleHomePage)
}
