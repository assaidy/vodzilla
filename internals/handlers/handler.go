package handlers

import (
	"github.com/assaidy/hyper/v2"
	"github.com/gofiber/fiber/v3"
)

func render(c fiber.Ctx, node hyper.HyperNode) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return hyper.Render(c, node)
}

func redirect(c fiber.Ctx, endpoint string) error {
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", endpoint)
	} else if c.Get("HX-Boosted") == "true" {
		c.Set("HX-Location", endpoint)
	} else {
		return c.Redirect().To(endpoint)
	}
	return nil
}
