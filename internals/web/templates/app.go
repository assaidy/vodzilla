package templates

import (
	"fmt"
	"time"

	. "github.com/assaidy/hyper/v2"
)

func FeedPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageFeed,
			Username:    username,
		},
	})(
		pageContentLoader("/feed/content"),
	)
}

func FeedPageContent() HyperNode {
	return Group(
		"Feed Page",
	)
}

func DiscoverPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageDiscover,
			Username:    username,
		},
	})(
		pageContentLoader("/discover/content"),
	)
}

func DiscoverPageContent() HyperNode {
	return Group(
		"Discover Page",
	)
}

func pageContentLoader(contentPath string) HyperNode {
	return DIV(
		Attr("hx-get", contentPath),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-trigger", "load"),
		Attr("hx-indicator", "#PAGE_CONTENT_CONTAINER"),
	)()
}

type WatchLaterPageContentParams struct {
	Username string
	Videos   []VideoCardParams
}

func WatchLaterPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageWatchLater,
			Username:    username,
		},
	})(
		pageContentLoader("/watch_later/content"),
	)
}

func WatchLaterPageContent(params WatchLaterPageContentParams) HyperNode {
	return Group(
		H1(AttrClass("text-2xl font-bold mb-4"))("Watch Later"),
		If(len(params.Videos) == 0,
			P(AttrClass("text-center text-base-content/60 mt-20"))("No videos in your watch later list."),
		).Else(
			profileVideosContainer(profileVideosParams{
				videoCards: params.Videos,
			}),
		),
	)
}

type PlaylistCardParams struct {
	Id          string
	Name        string
	VideosCount int
}

type PlaylistsPageContentParams struct {
	Username  string
	Playlists []PlaylistCardParams
}

func PlaylistsPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PagePlaylists,
			Username:    username,
		},
	})(
		pageContentLoader("/playlists/content"),
	)
}

func PlaylistsPageContent(params PlaylistsPageContentParams) HyperNode {
	return DIV(AttrId("PLAYLISTS_CONTAINER"), AttrClass("space-y-4"))(
		H1(AttrClass("text-2xl font-bold mb-4"))("Playlists"),
		If(len(params.Playlists) == 0,
			P(AttrClass("text-center text-base-content/60 mt-20"))("No playlists yet."),
		).Else(
			Group(
				Range(params.Playlists, func(p PlaylistCardParams) HyperNode {
					playlistLink := fmt.Sprintf("/playlists/%s", p.Id)
					return DIV(
						AttrClass("card bg-base-100 p-0 cursor-pointer transition-shadow duration-200 flex flex-row items-stretch overflow-hidden"),
						Attr("hx-get", fmt.Sprintf("%s/content", playlistLink)),
						Attr("hx-push-url", playlistLink),
						Attr("hx-target", "#APP_PAGE_CONTENT"),
						Attr("hx-swap", "innerHTML"),
						Attr("hx-indicator", "#PAGE_CONTENT_CONTAINER"),
					)(
						DIV(AttrClass("flex items-center justify-center bg-base-200 px-6"))(
							RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-base-content/80"><path d="M21 5H3"/><path d="M10 12H3"/><path d="M10 19H3"/><path d="M15 12.003a1 1 0 0 1 1.517-.859l4.997 2.997a1 1 0 0 1 0 1.718l-4.997 2.997a1 1 0 0 1-1.517-.86z"/></svg>`),
						),
						DIV(AttrClass("p-4"))(
							H2(AttrClass("card-title text-lg font-bold"))(p.Name),
							P(AttrClass("text-sm text-base-content/60"))(fmt.Sprintf("%d videos", p.VideosCount)),
						),
					)
				}),
			),
		),
	)
}

type PlaylistDetailPageContentParams struct {
	Username string
	Playlist PlaylistCardParams
	Videos   []VideoCardParams
}

func PlaylistDetailPage(username string, playlistId string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PagePlaylists,
			Username:    username,
		},
	})(
		pageContentLoader(fmt.Sprintf("/playlists/%s/content", playlistId)),
	)
}

func PlaylistDetailPageContent(params PlaylistDetailPageContentParams) HyperNode {
	return DIV(AttrId("PLAYLIST_DETAIL_CONTAINER"))(
		H1(AttrClass("text-2xl font-bold mb-6"))(params.Playlist.Name),
		If(len(params.Videos) == 0,
			P(AttrClass("text-center text-base-content/60 mt-20"))("This playlist is empty."),
		).Else(
			profileVideosContainer(profileVideosParams{
				videoCards: params.Videos,
			}),
		),
	)
}

func NotificationsPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageNotifications,
			Username:    username,
		},
	})(
		pageContentLoader("/notifications/content"),
	)
}

func NotificationsPageContent() HyperNode {
	return Group(
		"Notifications Page",
	)
}

func ProfilePage(username string, profileUsername string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageProfile,
			Username:    username,
		},
	})(
		pageContentLoader(fmt.Sprintf("/@%s/content", profileUsername)),
	)
}

type ProfilePageContentParams struct {
	Name     string
	Username string
	Bio      string
	IsOwner  bool
	Videos   []VideoCardParams
}

func ProfilePageContent(params ProfilePageContentParams) HyperNode {
	return Group(
		// profile card ==================================================
		DIV(AttrClass("card bg-base-100 p-2 flex flex-col md:flex-row overflow-hidden"))(
			profileCardAvatarPlaceholder(),
			DIV(AttrClass("p-4"))(
				H1(AttrId("PROFILE_CARD_NAME"), AttrClass("text-2xl font-bold"))(params.Name),
				P(AttrId("PROFILE_CARD_USERNAME"), AttrClass("text-sm text-base-content/60"))("@"+params.Username),
				DIV(AttrClass("mt-2 flex gap-6"))(
					P()(SPAN(AttrClass("font-bold"))("0"), SPAN(AttrClass("text-base-content/60"))(" following")),
					P()(SPAN(AttrClass("font-bold"))("0"), SPAN(AttrClass("text-base-content/60"))(" followers")),
				),
				P(AttrId("PROFILE_CARD_BIO"), AttrClass("mt-2"))(IfElse(params.Bio == "", "---", params.Bio)),
			),
		),

		// actions ==================================================
		If(params.IsOwner,
			// profile owner actions ==================================================
			DIV(AttrClass("mt-4 flex justify-center lg:justify-start gap-2"))(
				// edit profile ==================================================
				BUTTON(
					AttrClass("btn btn-soft"),
					AttrOnClick("EDIT_PROFILE_MODAL.show()"),
				)(
					RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user-pen-icon lucide-user-pen"><path d="M11.5 15H7a4 4 0 0 0-4 4v2"/><path d="M21.378 16.626a1 1 0 0 0-3.004-3.004l-4.01 4.012a2 2 0 0 0-.506.854l-.837 2.87a.5.5 0 0 0 .62.62l2.87-.837a2 2 0 0 0 .854-.506z"/><circle cx="10" cy="7" r="4"/></svg>`),
					"edit profile",
				),
				DIALOG(AttrId("EDIT_PROFILE_MODAL"), AttrClass("modal"))(
					DIV(AttrClass("modal-box"))(
						EditProfileFrom(EditProfileFromParams{
							Name:     params.Name,
							Username: params.Username,
							Bio:      params.Bio,
						}),
					),
					FORM(AttrMethod(MethodDialog), AttrClass("modal-backdrop"))(
						BUTTON()("close"),
					),
				),
				// post video ==================================================
				BUTTON(
					AttrClass("btn btn-soft"),
					AttrOnClick("POST_VIDEO_MODAL.show()"),
				)(
					RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-plus-icon lucide-plus"><path d="M5 12h14"/><path d="M12 5v14"/></svg>`),
					"post a video",
				),
				DIALOG(AttrId("POST_VIDEO_MODAL"), AttrClass("modal"))(
					DIV(AttrClass("modal-box"))(
						PostVideoForm(),
					),
					FORM(AttrMethod(MethodDialog), AttrClass("modal-backdrop"))(
						BUTTON()("close"),
					),
				),
			),
		).Else(
			// profiel viewer actions ==================================================
			Group(),
		),

		// profile videos ==================================================
		profileVideosContainer(profileVideosParams{
			videoCards: params.Videos,
		}),
	)
}

type EditProfileFromParams struct {
	Name        string
	NameErr     error
	Username    string
	UsernameErr error
	Bio         string
	BioErr      error
}

func EditProfileFrom(params EditProfileFromParams) HyperNode {
	inputClass := "input w-full"
	erroredInputClass := "input input-error w-full"

	return FORM(
		AttrId("EDIT_PROFILE_FORM"),
		AttrClass("space-y-4"),
		Attr("hx-put", "/profiles"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .submit-button"),
		Attr("hx-disable", "find .submit-button"),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_NAME"))(
				SPAN(AttrClass("label-text"))("Name"),
			),
			INPUT(
				AttrId("FORM_NAME"),
				AttrClass(IfElse(params.NameErr == nil, inputClass, erroredInputClass)),
				AttrType(TypeText),
				AttrName("name"),
				AttrValue(params.Name),
				AttrRequired(true),
			),
			If(params.NameErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(params.NameErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_USERNAME"))(
				SPAN(AttrClass("label-text"))("Username"),
			),
			INPUT(
				AttrId("FORM_USERNAME"),
				AttrClass(IfElse(params.UsernameErr == nil, inputClass, erroredInputClass)),
				AttrType(TypeText),
				AttrName("username"),
				AttrValue(params.Username),
				AttrRequired(true),
			),
			If(params.UsernameErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(params.UsernameErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_BIO"))(
				SPAN(AttrClass("label-text"))("Bio"),
			),
			TEXTAREA(
				AttrId("FORM_BIO"),
				AttrClass("block w-full textarea "+IfElseZero(params.BioErr != nil, "textarea-error")),
				AttrName("bio"),
			)(
				params.Bio,
			),
			If(params.BioErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(params.BioErr),
				),
			),
		),

		DIV(AttrClass("pt-2"))(
			BUTTON(
				AttrClass("btn btn-primary w-full submit-button group"),
				AttrType(TypeSubmit),
			)(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Submit"),
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block"))(),
			),
		),
	)
}

type appPageLayoutParams struct {
	navbarParams NavbarParams
}

func appPageLayout(params appPageLayoutParams) ChildrenInserter {
	return func(children ...any) Element {
		return basicPageLayout(basicLayoutParams{title: "Vidzilla"})(
			DIV(AttrClass("flex flex-col min-h-screen"))(
				Navbar(params.navbarParams),
				DIV(
					AttrId("PAGE_CONTENT_CONTAINER"),
					AttrClass("flex-1 relative group"),
				)(
					MAIN(AttrId("APP_PAGE_CONTENT"), AttrClass("w-full h-full p-6 overflow-y-auto"))(
						Group(children...),
					),
					DIV(AttrClass("hidden group-[.htmx-request]:flex absolute inset-0 items-center justify-center bg-base-300/70 z-10"))(
						SPAN(AttrClass("loading loading-spinner loading-lg"))(),
					),
				),
				videoUploadersContainer(),
				videoUploadIndicator(),
			),
		)
	}
}

type NavbarParams struct {
	Username    string
	CurrentPage AppPageKind
}

func Navbar(params NavbarParams) HyperNode {
	return NAV(AttrId("NAVBAR"), AttrClass("w-full sticky top-0 z-10 py-2 flex justify-center"))(
		DIV(AttrClass("card bg-base-100 p-2 flex-row gap-2"))(
			appPageButton(appPageButtonParams{
				tab:      PageFeed,
				pageLink: "/feed",
				icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-timeline-icon lucide-timeline"><path d="M4 12h.01"/><path d="M4 16h.01"/><path d="M4 20h.01"/><path d="M4 4h.01"/><path d="M4 8h.01"/><path d="M9.414 13.414a2 2 0 0 0 1.414.586H19a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 12z"/><path d="M9.414 21.414a2 2 0 0 0 1.414.586H19a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 20z"/><path d="M9.414 5.414A2 2 0 0 0 10.828 6H19a1 1 0 0 0 1-1V3a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 4z"/></svg>`,
				isActive: params.CurrentPage == PageFeed,
			}),
			appPageButton(appPageButtonParams{
				tab:      PageDiscover,
				pageLink: "/discover",
				icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-binoculars-icon lucide-binoculars"><path d="M10 10h4"/><path d="M19 7V4a1 1 0 0 0-1-1h-2a1 1 0 0 0-1 1v3"/><path d="M20 21a2 2 0 0 0 2-2v-3.851c0-1.39-2-2.962-2-4.829V8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v11a2 2 0 0 0 2 2z"/><path d="M 22 16 L 2 16"/><path d="M4 21a2 2 0 0 1-2-2v-3.851c0-1.39 2-2.962 2-4.829V8a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v11a2 2 0 0 1-2 2z"/><path d="M9 7V4a1 1 0 0 0-1-1H6a1 1 0 0 0-1 1v3"/></svg>`,
				isActive: params.CurrentPage == PageDiscover,
			}),
			DIV(AttrClass("tooltip tooltip-bottom"), Attr("data-tip", "Search"))(
				BUTTON(AttrClass("p-2 btn btn-ghost"))(
					RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-search"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`),
				),
			),
			appPageButton(appPageButtonParams{
				tab:      PageWatchLater,
				pageLink: "/watch_later",
				icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-clock-icon lucide-clock"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>`,
				isActive: params.CurrentPage == PageWatchLater,
			}),
			appPageButton(appPageButtonParams{
				tab:      PagePlaylists,
				pageLink: "/playlists",
				icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-list-video-icon lucide-list-video"><path d="M21 5H3"/><path d="M10 12H3"/><path d="M10 19H3"/><path d="M15 12.003a1 1 0 0 1 1.517-.859l4.997 2.997a1 1 0 0 1 0 1.718l-4.997 2.997a1 1 0 0 1-1.517-.86z"/></svg>`,
				isActive: params.CurrentPage == PagePlaylists,
			}),
			appPageButton(appPageButtonParams{
				tab:      PageNotifications,
				pageLink: "/notifications",
				icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-bell-icon lucide-bell"><path d="M10.268 21a2 2 0 0 0 3.464 0"/><path d="M3.262 15.326A1 1 0 0 0 4 17h16a1 1 0 0 0 .74-1.673C19.41 13.956 18 12.499 18 8A6 6 0 0 0 6 8c0 4.499-1.411 5.956-2.738 7.326"/></svg>`,
				isActive: params.CurrentPage == PageNotifications,
			}),
			appPageButton(appPageButtonParams{
				tab:      PageProfile,
				pageLink: "/@" + params.Username,
				icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user-icon lucide-user"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`,
				isActive: params.CurrentPage == PageProfile,
			}),
		),
	)
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
	pageLink string
	icon     string
	isActive bool
}

func appPageButton(params appPageButtonParams) HyperNode {
	return DIV(AttrClass("tooltip tooltip-bottom"), Attr("data-tip", params.tab))(
		BUTTON(
			AttrClass("p-2 btn btn-ghost "+IfElseZero(params.isActive, "btn-primary")),
			Attr("hx-get", fmt.Sprintf("%s/content", params.pageLink)),
			Attr("hx-push-url", params.pageLink),
			Attr("hx-target", "#APP_PAGE_CONTENT"),
			Attr("hx-swap", "innerHTML"),
			Attr("hx-indicator", "#PAGE_CONTENT_CONTAINER"),
		)(
			RawText(params.icon),
		),
	)
}

func profileCardAvatarPlaceholder() HyperNode {
	return DIV(AttrClass("w-full aspect-square md:w-48 md:h-48 md:aspect-auto lg:w-64 lg:h-64 rounded-box shrink-0 flex items-center justify-center bg-neutral text-neutral-content"))(
		RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`),
	)
}

func VideoPage(username string, videoId string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			Username: username,
		},
	})(
		pageContentLoader(fmt.Sprintf("/videos/%s/content", videoId)),
	)
}

type VideoPageContentParams struct {
	Id                       string
	OwnerName                string
	OwnerUsername            string
	SourceUrl                string
	Title                    string
	Description              string
	Timestamp                time.Time
	ViewsCount               int
	ReactionsParams          ReactionsWidgetParams
	WatchLaterButtonParams   WatchLaterButtonParams
	AddToPlaylistModalParams AddToPlaylistModalParams
}

func VideoPageContent(params VideoPageContentParams) HyperNode {
	ownerProfileLink := fmt.Sprintf("/@%s", params.OwnerUsername)
	visitProfileAttrs := []Attribute{
		Attr("hx-get", fmt.Sprintf("%s/content", ownerProfileLink)),
		Attr("hx-push-url", ownerProfileLink),
		Attr("hx-target", "#APP_PAGE_CONTENT"),
		Attr("hx-swap", "innerHTML"),
		Attr("hx-trigger", "click consume"),
		Attr("hx-indicator", "#PAGE_CONTENT_CONTAINER"),
	}

	return DIV(AttrClass("max-w-6xl mx-auto"))(
		VIDEO(
			AttrId("VIDEO_PLAYER"),
			AttrClass("w-full aspect-video bg-black rounded-box"),
			AttrSrc(params.SourceUrl),
			AttrControls(true),
			AttrPlaysInline(true),
			Attr("hx-post", fmt.Sprintf("/videos/%s/views", params.Id)),
			Attr("hx-trigger", "load"),
		)(),
		DIV(AttrClass("mt-4"))(
			H1(AttrClass("text-2xl font-bold"))(params.Title),
			DIV(AttrClass("mt-4 flex items-center gap-3"))(
				DIV(append(visitProfileAttrs, AttrClass("shrink-0 cursor-pointer"))...)(
					videoCardAvatarPlaceholder(),
				),
				A(append(visitProfileAttrs, AttrClass("link link-hover font-semibold"))...)(
					params.OwnerName,
				),

				// like,dislike
				ReactionsWidget(params.ReactionsParams),
				// watch later
				WatchLaterButton(params.WatchLaterButtonParams),
				// add to playlist
				AddToPlaylistButton(params.AddToPlaylistModalParams.VideoId),
			),
			DIV(AttrClass("mt-4 card bg-base-200 p-4 space-y-2"))(
				DIV(AttrClass("text-sm text-base-content/60 flex gap-4"))(
					SPAN()(fmt.Sprintf("%d views", params.ViewsCount)),
					SPAN()(params.Timestamp.Format(time.DateOnly)),
				),
				P(AttrClass("text-base-content/80"))(IfElse(params.Description == "", "---", params.Description)),
			),
		),

		AddToPlaylistModal(params.AddToPlaylistModalParams),

		SCRIPT()(RawText(fmt.Sprintf(`
			(() => {
				const v = VIDEO_PLAYER;
				let attempts = 0;
				v.addEventListener('error', async () => {
					if (v.error && v.error.code !== v.error.MEDIA_ERR_NETWORK) return;
					if (++attempts > 3) return;
					try {
						const r = await fetch('/videos/%s/stream_url');
						const d = await r.json();
						const t = v.currentTime;
						const p = !v.paused;
						v.src = d.url;
						v.currentTime = t;
						if (p) await v.play();
					} catch(e) {
						console.error(e);
					}
				});
				v.addEventListener('playing', () => { attempts = 0; });
			})();
		`, params.Id))),
	)
}

type ReactionsWidgetParams struct {
	VideoId       string
	LikesCount    int
	DislikesCount int
	IsLiked       bool
	IsDisliked    bool
}

func ReactionsWidget(params ReactionsWidgetParams) HyperNode {
	return DIV(AttrId("REACTIONS_WIDGET"), AttrClass("join ml-auto"))(
		BUTTON(
			AttrClass("join-item btn btn-soft btn-sm tooltip tooltip-top"),
			Attr("data-tip", "Like"),
			IfElse(!params.IsLiked,
				Attr("hx-post", fmt.Sprintf("/videos/%s/reactions?kind=like", params.VideoId)),
				Attr("hx-delete", fmt.Sprintf("/videos/%s/reactions?kind=like", params.VideoId)),
			),
			Attr("hx-target", "#REACTIONS_WIDGET"),
			Attr("hx-swap", "outerHTML"),
		)(
			RawText(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="%s"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"/></svg>`,
				IfElse(params.IsLiked, "text-primary", ""),
			)),
			SPAN(AttrClass("text-sm"))(fmt.Sprint(params.LikesCount)),
		),
		BUTTON(
			AttrClass("join-item btn btn-soft btn-sm tooltip tooltip-top"),
			Attr("data-tip", "Dislike"),
			IfElse(!params.IsDisliked,
				Attr("hx-post", fmt.Sprintf("/videos/%s/reactions?kind=dislike", params.VideoId)),
				Attr("hx-delete", fmt.Sprintf("/videos/%s/reactions?kind=dislike", params.VideoId)),
			),
			Attr("hx-target", "#REACTIONS_WIDGET"),
			Attr("hx-swap", "outerHTML"),
		)(
			RawText(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="%s"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22a3.13 3.13 0 0 1-3-3.88Z"/></svg>`,
				IfElse(params.IsDisliked, "text-primary", ""),
			)),
			SPAN(AttrClass("text-sm"))(fmt.Sprint(params.DislikesCount)),
		),
	)
}
