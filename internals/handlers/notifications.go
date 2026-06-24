package handlers

import (
	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func (me *Handler) HandleNotificationsPageContent(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.NotificationsPageContent(),

		hyper.DIV(hyper.AttrId("navbar"), templates.AttrHxSwapOob(templates.SwapOuterHtml))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageNotifications,
			}),
		),
	))
}
