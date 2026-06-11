package templates

import (
	"fmt"
	"time"

	. "github.com/assaidy/hyper/v2"
	"github.com/assaidy/lucide"
	"github.com/google/uuid"
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
			profileVideos(profileVideosParams{
				videoCards: params.Videos,
			}),
		),
	)
}

type PlaylistCardParams struct {
	Id          uuid.UUID
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
			Range(params.Playlists, func(p PlaylistCardParams) any {
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
						RawText(lucide.ListVideo(lucide.Params{Class: "text-base-content/80"})),
					),
					DIV(AttrClass("p-4"))(
						H2(AttrClass("card-title text-lg font-bold"))(p.Name),
						P(AttrClass("text-sm text-base-content/60"))(fmt.Sprintf("%d videos", p.VideosCount)),
					),
				)
			}),
		),
	)
}

type PlaylistDetailPageContentParams struct {
	Username string
	Playlist PlaylistCardParams
	Videos   []VideoCardParams
}

func PlaylistDetailPage(username string, playlistId uuid.UUID) HyperNode {
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
			profileVideos(profileVideosParams{
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
	OwnerId        uuid.UUID
	Name           string
	Username       string
	Bio            string
	IsOwner        bool
	Videos         []VideoCardParams
	FollowersCount uint
	PostsCount     uint
	IsFollowed     bool
}

func ProfilePageContent(params ProfilePageContentParams) HyperNode {
	return Group(
		// profile card ==================================================
		DIV(AttrClass("card bg-base-100 p-2 flex flex-col md:flex-row overflow-hidden"))(
			profileCardAvatarPlaceholder(),
			DIV(AttrClass("p-4 min-w-0"))(
				DIV(AttrClass("flex items-start justify-between gap-2"))(
					DIV(AttrClass("min-w-0 flex-1"))(
						H1(AttrId("PROFILE_CARD_NAME"), AttrClass("text-2xl font-bold truncate"))(params.Name),
						P(AttrId("PROFILE_CARD_USERNAME"), AttrClass("text-sm text-base-content/60"))("@"+params.Username),
					),
					If(!params.IsOwner,
						FollowButton(FollowButtonParams{
							ProfileOwnerId: params.OwnerId,
							IsFollowed:     params.IsFollowed,
						}),
					),
				),
				DIV(AttrClass("mt-2 flex gap-6"))(
					P()(SPAN(AttrClass("font-bold"))(params.FollowersCount), SPAN(AttrClass("text-base-content/60"))(" followers")),
					P()(SPAN(AttrClass("font-bold"))(params.PostsCount), SPAN(AttrClass("text-base-content/60"))(" posts")),
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
					RawText(lucide.UserPen()),
					"edit profile",
				),
				DIALOG(AttrId("EDIT_PROFILE_MODAL"), AttrClass("modal"))(
					DIV(AttrClass("modal-box"))(
						EditProfileForm(EditProfileFormParams{
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
					RawText(lucide.Plus()),
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
			// profile viewer actions ==================================================
			Group(),
		),

		// profile videos ==================================================
		profileVideos(profileVideosParams{
			videoCards: params.Videos,
		}),
	)
}

type FollowButtonParams struct {
	ProfileOwnerId uuid.UUID
	IsFollowed     bool
}

func FollowButton(params FollowButtonParams) HyperNode {
	return DIV(AttrId("FOLLOW_BUTTON"))(
		If(params.IsFollowed,
			BUTTON(
				AttrClass("btn btn-outline btn-accent hover:btn-error group/follow"),
				Attr("hx-delete", fmt.Sprintf("/follow/%s", params.ProfileOwnerId)),
				Attr("hx-target", "#FOLLOW_BUTTON"),
				Attr("hx-swap", "outerHTML"),
				Attr("hx-disable", "this"),
			)(
				SPAN(AttrClass("group-hover/follow:hidden"))("Following"),
				SPAN(AttrClass("hidden group-hover/follow:inline"))("Unfollow"),
			),
		).Else(
			BUTTON(
				AttrClass("btn btn-accent"),
				Attr("hx-post", fmt.Sprintf("/follow/%s", params.ProfileOwnerId)),
				Attr("hx-target", "#FOLLOW_BUTTON"),
				Attr("hx-swap", "outerHTML"),
				Attr("hx-disable", "this"),
			)("Follow"),
		),
	)
}

type EditProfileFormParams struct {
	Name        string
	NameErr     error
	Username    string
	UsernameErr error
	Bio         string
	BioErr      error
}

func EditProfileForm(params EditProfileFormParams) HyperNode {
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
				AttrClass(Classes("input w-full", IfElseZero(params.NameErr != nil, "input-error"))),
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
				AttrClass(Classes("input w-full", IfElseZero(params.UsernameErr != nil, "input-error"))),
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
				AttrClass(Classes("textarea block w-full", IfElseZero(params.BioErr != nil, "textarea-error"))),
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
		return basicPageLayout("Vidzilla")(
			DIV(AttrClass("flex flex-col min-h-screen"))(
				Navbar(params.navbarParams),
				DIV(
					AttrId("PAGE_CONTENT_CONTAINER"),
					AttrClass("flex-1 relative group pt-20"),
				)(
					MAIN(AttrId("APP_PAGE_CONTENT"), AttrClass("w-full p-6"))(
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
	return NAV(AttrId("NAVBAR"), AttrClass("w-full fixed top-0 z-10 py-2 flex justify-center"))(
		DIV(AttrClass("card bg-base-100 p-2 flex-row gap-2"))(
			appPageButton(appPageButtonParams{
				tab:      PageFeed,
				pageLink: "/feed",
				// TODO: missing assaidy/lucide timeline
				icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-timeline-icon lucide-timeline"><path d="M4 12h.01"/><path d="M4 16h.01"/><path d="M4 20h.01"/><path d="M4 4h.01"/><path d="M4 8h.01"/><path d="M9.414 13.414a2 2 0 0 0 1.414.586H19a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 12z"/><path d="M9.414 21.414a2 2 0 0 0 1.414.586H19a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 20z"/><path d="M9.414 5.414A2 2 0 0 0 10.828 6H19a1 1 0 0 0 1-1V3a1 1 0 0 0-1-1h-8.172a2 2 0 0 0-1.414.586L8 4z"/></svg>`,
				isActive: params.CurrentPage == PageFeed,
			}),
			appPageButton(appPageButtonParams{
				tab:      PageDiscover,
				pageLink: "/discover",
				icon:     lucide.Binoculars(),
				isActive: params.CurrentPage == PageDiscover,
			}),
			DIV(AttrClass("tooltip tooltip-bottom"), Attr("data-tip", "Search"))(
				BUTTON(AttrClass("p-2 btn btn-ghost"))(
					RawText(lucide.Search()),
				),
			),
			appPageButton(appPageButtonParams{
				tab:      PageWatchLater,
				pageLink: "/watch_later",
				icon:     lucide.Clock(),
				isActive: params.CurrentPage == PageWatchLater,
			}),
			appPageButton(appPageButtonParams{
				tab:      PagePlaylists,
				pageLink: "/playlists",
				icon:     lucide.ListVideo(),
				isActive: params.CurrentPage == PagePlaylists,
			}),
			appPageButton(appPageButtonParams{
				tab:      PageNotifications,
				pageLink: "/notifications",
				icon:     lucide.Bell(),
				isActive: params.CurrentPage == PageNotifications,
			}),
			appPageButton(appPageButtonParams{
				tab:      PageProfile,
				pageLink: "/@" + params.Username,
				icon:     lucide.User(),
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
			AttrClass(Classes("btn btn-ghost p-2", IfElseZero(params.isActive, "btn-primary"))),
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
		RawText(lucide.User()),
	)
}

func VideoPage(username string, videoId uuid.UUID) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			Username: username,
		},
	})(
		pageContentLoader(fmt.Sprintf("/videos/%s/content", videoId)),
	)
}

type VideoPageContentParams struct {
	Id                       uuid.UUID
	OwnerId                  uuid.UUID
	OwnerName                string
	OwnerUsername            string
	SourceUrl                string
	Title                    string
	Description              string
	Timestamp                time.Time
	ViewsCount               int
	CurrentUserId            uuid.UUID
	IsFollowed               bool
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
					videoCardOwnerAvatarPlaceholder(),
				),
				A(append(visitProfileAttrs, AttrClass("link link-hover font-semibold"))...)(
					params.OwnerName,
				),

				If(params.CurrentUserId != params.OwnerId,
					FollowButton(FollowButtonParams{
						ProfileOwnerId: params.OwnerId,
						IsFollowed:     params.IsFollowed,
					}),
				),

				// like,dislike
				ReactionsWidget(params.ReactionsParams),
				// watch later
				WatchLaterButton(params.WatchLaterButtonParams),
				// add to playlist
				AddToPlaylistButton(),
				AddToPlaylistModal(params.AddToPlaylistModalParams),
			),
			DIV(AttrClass("mt-4 card bg-base-200 p-4 space-y-2"))(
				DIV(AttrClass("text-sm text-base-content/60 flex gap-4"))(
					SPAN()(fmt.Sprintf("%d views", params.ViewsCount)),
					SPAN()(params.Timestamp.Format(time.DateOnly)),
				),
				P(AttrClass("text-base-content/80"))(IfElse(params.Description == "", "---", params.Description)),
			),
		),

		CommentSection(params.Id),

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
	VideoId       uuid.UUID
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
			Attr(IfElse(params.IsLiked, "hx-delete", "hx-post"), fmt.Sprintf("/videos/%s/reactions?kind=like", params.VideoId)),
			Attr("hx-target", "#REACTIONS_WIDGET"),
			Attr("hx-swap", "outerHTML"),
		)(
			RawText(lucide.ThumbsUp(lucide.Params{
				Class: IfElseZero(params.IsLiked, "text-primary"),
			})),
			SPAN(AttrClass("text-sm"))(params.LikesCount),
		),
		BUTTON(
			AttrClass("join-item btn btn-soft btn-sm tooltip tooltip-top"),
			Attr("data-tip", "Dislike"),
			Attr(IfElse(params.IsDisliked, "hx-delete", "hx-post"), fmt.Sprintf("/videos/%s/reactions?kind=dislike", params.VideoId)),
			Attr("hx-target", "#REACTIONS_WIDGET"),
			Attr("hx-swap", "outerHTML"),
		)(
			RawText(lucide.ThumbsDown(lucide.Params{
				Class: IfElseZero(params.IsDisliked, "text-primary"),
			})),
			SPAN(AttrClass("text-sm"))(params.DislikesCount),
		),
	)
}
