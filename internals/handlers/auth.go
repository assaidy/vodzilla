package handlers

import (
	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/video_streaming_app/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func HandleRegisterPage(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, templates.RegisterPage())
}

func HandleRegister(c fiber.Ctx) error {
	println(c.FormValue("name"))
	println(c.FormValue("username"))
	println(c.FormValue("email"))
	println(c.FormValue("password"))
	println(c.FormValue("verifyPassword"))
	return nil
}

func HandleLoginPage(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, templates.LoginPage())
}

func HandleLogin(c fiber.Ctx) error {
	println(c.FormValue("email"))
	println(c.FormValue("password"))
	return nil
}
