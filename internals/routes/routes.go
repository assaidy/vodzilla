package routes

import (
	"github.com/assaidy/video_streaming_app/internals/handlers"
	"github.com/assaidy/video_streaming_app/internals/web"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func RegisterRoutes(app *fiber.App) {
	app.Use(handlers.WithLogging)
	app.Use(static.New("public/", static.Config{FS: web.PublicFS, MaxAge: 0}))

	app.Get("/health", handlers.HandleCheckHealth)
	app.Get("/data_sse/:client_id", handlers.WithSseHelperData, handlers.HandleDatastarSse)
	app.RouteChain("/register").
		Get(handlers.HandleRegisterPage).
		Post(handlers.HandleRegister)
	app.RouteChain("/login").
		Get(handlers.HandleLoginPage).
		Post(handlers.HandleLogin)
	// app.Get("/verification_email", handlers.HandleGetVerificationEmail)
	// app.Get("/verification_email/verify", handlers.HandleVerifiyEmail)
}
