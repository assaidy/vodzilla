package handlers

import (
	"github.com/assaidy/hyper/v2"
	"github.com/assaidy/vodzilla/internals/web/templates"
	"github.com/gofiber/fiber/v3"
)

func (me *Handler) HandleHomePage(c fiber.Ctx) error {
	return redirect(c, "/feed")
}

func (me *Handler) HandleFeedPage(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	return render(c, templates.FeedPage(currentUser.Username))
}

func (me *Handler) HandleFeedPageContent(c fiber.Ctx) error {
	currentUser, err := me.getCurrentUser(c)
	if err != nil {
		return err
	}

	followedIds, err := me.socialService.GetFollowedUserIdsForUser(c.RequestCtx(), currentUser.Id)
	if err != nil {
		return err
	}

	videos, err := me.videoService.GetAllVideosForMultipleUsers(c.RequestCtx(), followedIds)
	if err != nil {
		return err
	}

	templateVideos, err := me.getTemplateVideosFromVideos(c, videos)
	if err != nil {
		return err
	}

	return render(c, hyper.Group(
		templates.FeedPageContent(templateVideos),

		hyper.DIV(hyper.AttrId("NAVBAR"), hyper.Attr("hx-swap-oob", "outerHTML"))(
			templates.Navbar(templates.NavbarParams{
				Username:    currentUser.Username,
				CurrentPage: templates.PageFeed,
			}),
		),
	))
}
