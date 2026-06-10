package handlers

import (
	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func HandleDiscoverPage(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.DiscoverPage(currentUser.Username))
}

func HandleDiscoverPageContent(c fiber.Ctx) error {
	currentUser, err := getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.DiscoverPageContent(),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageDiscover,
			}),
		),
	))
}
