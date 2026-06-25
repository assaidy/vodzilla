package handlers

import (
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func (me *Handler) HandlePlaygroundPage(c fiber.Ctx) error {
	return render(c, templates.PlaygroundPage())
}
