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
	app.Use(handlers.WithPassClientIdToLocals)

	app.Use(static.New("assets/", static.Config{
		FS:       web.AssetsFS,
		Compress: true,
		ModifyResponse: func(c fiber.Ctx) error {
			c.Set(fiber.HeaderCacheControl, "no-cache, no-store")
			return nil
		},
	}))
	app.Get("/health", handlers.HandleCheckHealth)

	app.Get("/ws/:client_id", handlers.WithWebsocketEssentials, websocket.New(handlers.HandleWebsocket))
	app.Get("/register", handlers.HandleRegisterPage)
	app.Post("/register", handlers.HandleRegister)
	app.Get("/login", handlers.HandleLoginPage)
	app.Post("/login", handlers.HandleLogin)
	// TODO: app.Post("/verification_email", handlers.HandleGetVerificationEmail)
	app.Get("/verification_email/sent", handlers.HandleVerificationEmailSentPage)
	app.Get("/verification_email/verify", handlers.HandleVerifyEmailPage)

	app.Get("/", handlers.HandleHomePage)
	app.Get("/feed", handlers.WithSession, handlers.HandleFeedPage)
	app.Get("/feed/content", handlers.WithSession, handlers.HandleFeedPageContent)
	app.Get("/discover", handlers.WithSession, handlers.HandleDiscoverPage)
	app.Get("/discover/content", handlers.WithSession, handlers.HandleDiscoverPageContent)
	app.Get("/watch_later", handlers.WithSession, handlers.HandleWatchLaterPage)
	app.Get("/watch_later/content", handlers.WithSession, handlers.HandleWatchLaterPageContent)
	app.Get("/playlists", handlers.WithSession, handlers.HandlePlaylistsPage)
	app.Get("/playlists/content", handlers.WithSession, handlers.HandlePlaylistsPageContent)
	app.Get("/notifications", handlers.WithSession, handlers.HandleNotificationsPage)
	app.Get("/notifications/content", handlers.WithSession, handlers.HandleNotificationsPageContent)
	app.Get("/@:username", handlers.WithSession, handlers.HandleProfilePage)
	app.Get("/@:username/content", handlers.WithSession, handlers.HandleProfilePageContent)
	app.Put("/profiles", handlers.WithSession, handlers.WithCsrfToken, handlers.HandleEditProfile)
	// TODO: edit account: email, password, delete account

	app.Post("/videos", handlers.WithSession, handlers.WithCsrfToken, handlers.HandlePostVideo)
	app.Post("/videos/complete_upload", handlers.WithSession, handlers.HandleCompleteVideoUpload)
	app.Get("/videos/:video_id", handlers.WithSession, handlers.HandleVideoPage)
	app.Get("/videos/:video_id/content", handlers.WithSession, handlers.HandleVideoPageContent)
	app.Get("/videos/:video_id/stream_url", handlers.WithSession, handlers.HandleGetVideoStreamUrl)
}
