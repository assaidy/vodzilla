package templates

import (
	"fmt"

	. "github.com/assaidy/hyper/v2"
)

// TODO: use hyper.ElementBuilder in all layouts

func FeedPage(profile PageLayoutProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageFeed,
		profile:     profile,
	}, Group(
		// for each app page, put a page content loader here with htmx.
		"Feed Page",
	))
}

func DiscoverPage(profile PageLayoutProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageDiscover,
		profile:     profile,
	}, Group(
		"Discover Page",
	))
}

func WatchLaterPage(profile PageLayoutProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageWatchLater,
		profile:     profile,
	}, Group(
		"Watch Later Page",
	))
}

func PlaylistsPage(profile PageLayoutProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PagePlaylists,
		profile:     profile,
	}, Group(
		"Playlists Page",
	))
}

func NotificationsPage(profile PageLayoutProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageNotifications,
		profile:     profile,
	}, Group(
		"Notifications Page",
	))
}

func ProfilePage(profile PageLayoutProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageProfile,
		profile:     profile,
	}, Group(
		"Profile Page",
	))
}

type appPageLayoutParams struct {
	currentPage AppPageKind
	profile     PageLayoutProfile
}

type PageLayoutProfile struct {
	Username       string
	AvatarImageUrl string
}

func appPageLayout(params appPageLayoutParams, mainContent HyperNode) HyperNode {
	return basicPageLayout("Vidzilla", Group(
		DIV(AttrClass("flex flex-col min-h-screen"))(
			NAV(AttrClass("w-full sticky py-2 flex justify-center"))(
				DIV(AttrClass("card bg-base-100 p-2 flex-row gap-2"))(
					appPageButton(appPageButtonParams{
						tab:      PageFeed,
						link:     "/feed",
						icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-timeline-icon lucide-timeline"><path d="M4 12h.01"/><path d="M4 16h.01"/><path d="M4 20h.01"/><path d="M4 4h.01"/><path d="M4 8h.01"/><path d="M9.414 13.414a2 2 0 0 0 1.414.586H19a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 12z"/><path d="M9.414 21.414a2 2 0 0 0 1.414.586H19a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 20z"/><path d="M9.414 5.414A2 2 0 0 0 10.828 6H19a1 1 0 0 0 1-1V3a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 4z"/></svg>`,
						isActive: params.currentPage == PageFeed,
					}),
					appPageButton(appPageButtonParams{
						tab:      PageDiscover,
						link:     "/discover",
						icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-binoculars-icon lucide-binoculars"><path d="M10 10h4"/><path d="M19 7V4a1 1 0 0 0-1-1h-2a1 1 0 0 0-1 1v3"/><path d="M20 21a2 2 0 0 0 2-2v-3.851c0-1.39-2-2.962-2-4.829V8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v11a2 2 0 0 0 2 2z"/><path d="M 22 16 L 2 16"/><path d="M4 21a2 2 0 0 1-2-2v-3.851c0-1.39 2-2.962 2-4.829V8a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v11a2 2 0 0 1-2 2z"/><path d="M9 7V4a1 1 0 0 0-1-1H6a1 1 0 0 0-1 1v3"/></svg>`,
						isActive: params.currentPage == PageDiscover,
					}),
					DIV(AttrClass("tooltip tooltip-bottom"), Attr("data-tip", "Search"))(
						BUTTON(AttrClass("p-2 btn"))(
							RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`),
						),
					),
					appPageButton(appPageButtonParams{
						tab:      PageWatchLater,
						link:     "/watch_later",
						icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-clock-icon lucide-clock"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>`,
						isActive: params.currentPage == PageWatchLater,
					}),
					appPageButton(appPageButtonParams{
						tab:      PagePlaylists,
						link:     "/playlists",
						icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-list-video-icon lucide-list-video"><path d="M21 5H3"/><path d="M10 12H3"/><path d="M10 19H3"/><path d="M15 12.003a1 1 0 0 1 1.517-.859l4.997 2.997a1 1 0 0 1 0 1.718l-4.997 2.997a1 1 0 0 1-1.517-.86z"/></svg>`,
						isActive: params.currentPage == PagePlaylists,
					}),
					appPageButton(appPageButtonParams{
						tab:      PageNotifications,
						link:     "/notifications",
						icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-bell-icon lucide-bell"><path d="M10.268 21a2 2 0 0 0 3.464 0"/><path d="M3.262 15.326A1 1 0 0 0 4 17h16a1 1 0 0 0 .74-1.673C19.41 13.956 18 12.499 18 8A6 6 0 0 0 6 8c0 4.499-1.411 5.956-2.738 7.326"/></svg>`,
						isActive: params.currentPage == PageNotifications,
					}),
					appPageButton(appPageButtonParams{
						tab:      PageProfile,
						link:     fmt.Sprintf("/%s", params.profile.Username),
						icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user-icon lucide-user"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`,
						isActive: params.currentPage == PageProfile,
					}),
				),
			),
			MAIN(AttrClass("flex-1 p-6 overflow-y-auto"))(
				mainContent,
			),
		),
	))
}

type AppPageKind = string

const (
	PageFeed          AppPageKind = "Feed"
	PageDiscover      AppPageKind = "Discover"
	PageProfile       AppPageKind = "Profile"
	PageNotifications AppPageKind = "Notifications"
	PageWatchLater    AppPageKind = "Watch Later"
	PagePlaylists     AppPageKind = "Playlists"
)

type appPageButtonParams struct {
	tab      AppPageKind
	link     string
	icon     string
	isActive bool
}

func appPageButton(params appPageButtonParams) HyperNode {
	return DIV(AttrClass("tooltip tooltip-bottom"), Attr("data-tip", params.tab))(
		BUTTON(AttrClass("p-2 btn " + IfElse(params.isActive, "btn-primary", "")))(
			A(AttrHref(params.link))(RawText(params.icon)),
		),
	)
}
