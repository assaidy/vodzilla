package handlers

import (
	"github.com/assaidy/video_streaming_app/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func HandleHomePage(c fiber.Ctx) error {
	return render(c, templates.HomePage())
}
