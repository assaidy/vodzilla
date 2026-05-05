package routes

import (
	"github.com/assaidy/vodzilla/internals/handlers"
	"github.com/assaidy/vodzilla/internals/web"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func RegisterRoutes(app *fiber.App) {
	app.Use(handlers.WithLogging)
	app.Use(static.New("assets/", static.Config{FS: web.AssetsFS, Compress: true}))
	app.Use(handlers.WithPassClientIDToLocals)

	app.Get("/health", handlers.HandleCheckHealth)
	app.Get("/ws/:client_id", handlers.WithWebsocketEssentials, websocket.New(handlers.HandleWebsocket))
	app.Get("/register", handlers.HandleRegisterPage)
	app.Post("/register", handlers.HandleRegister)
	app.Get("/login", handlers.HandleLoginPage)
	app.Post("/login", handlers.HandleLogin)
	// app.Post("/verification_email", handlers.HandleGetVerificationEmail)
	app.Get("/verification_email/sent", handlers.HandleVerificationEmailSentPage)
	app.Get("/verification_email/verify", handlers.HandleVerifyEmailPage)

	app.Get("/", handlers.WithSession, handlers.HandleHomePage)
}
