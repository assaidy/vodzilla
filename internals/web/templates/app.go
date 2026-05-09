package templates

import (
	. "github.com/assaidy/hyper/v2"
)

// TODO: use hyper.ElementBuilder in all layouts

func FeedPage(profile NavbarProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageFeed,
		profile:     profile,
	})(
		"Feed Page",
	)
}

func DiscoverPage(profile NavbarProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageDiscover,
		profile:     profile,
	})(
		"Discover Page",
	)
}

func WatchLaterPage(profile NavbarProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageWatchLater,
		profile:     profile,
	})(
		"Watch Later Page",
	)
}

func PlaylistsPage(profile NavbarProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PagePlaylists,
		profile:     profile,
	})(
		"Playlists Page",
	)
}

func NotificationsPage(profile NavbarProfile) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageNotifications,
		profile:     profile,
	})(
		"Notifications Page",
	)
}

type ProfilePageParams struct {
	NavbarProfile NavbarProfile
	Name          string
	Username      string
	Bio           string
	IsOwner       bool
}

func ProfilePage(params ProfilePageParams) HyperNode {
	return appPageLayout(appPageLayoutParams{
		currentPage: PageProfile,
		profile:     params.NavbarProfile,
	})(
		// profile card ==================================================
		DIV(AttrClass("card bg-base-100 p-2 flex flex-col lg:flex-row overflow-hidden"))(
			DIV(AttrClass("w-full aspect-square lg:w-64 lg:h-64 lg:aspect-auto rounded-box bg-gradient-to-r from-info to-error shrink-0"))(),
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
					AttrOnClick("UPLOAD_VIDEO_MODAL.show()"),
				)(
					RawText(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#ffffff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-plus-icon lucide-plus"><path d="M5 12h14"/><path d="M12 5v14"/></svg>`),
					"post a video",
				),
				DIALOG(AttrId("UPLOAD_VIDEO_MODAL"), AttrClass("modal"))(
					DIV(AttrClass("modal-box"))(
						UploadVideoForm(),
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

		// profile posts (videos) ==================================================
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
				spinner(),
			),
		),
	)
}

type UploadVideoFormParams struct {
	Title          string
	TitleErr       error
	Description    string
	DescriptionErr error
	ThumbnailErr   error
	VideoErr       error
}

func UploadVideoForm(params ...UploadVideoFormParams) HyperNode {
	var p UploadVideoFormParams
	if len(params) > 0 {
		p = params[0]
	}

	return FORM(
		AttrClass("space-y-4"),
		Attr("hx-post", "/videos"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .submit-button"),
		Attr("hx-disable", "find .submit-button"),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_TITLE"))(
				SPAN(AttrClass("label-text"))("Title"),
			),
			INPUT(
				AttrId("FORM_TITLE"),
				AttrClass(IfElse(p.TitleErr == nil, "input w-full", "input input-error w-full")),
				AttrType(TypeText),
				AttrName("title"),
				AttrValue(p.Title),
				AttrRequired(true),
			),
			If(p.TitleErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.TitleErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_DESCRIPTION"))(
				SPAN(AttrClass("label-text"))("Description"),
			),
			TEXTAREA(
				AttrId("FORM_DESCRIPTION"),
				AttrClass("block w-full textarea"+IfElseZero(p.DescriptionErr != nil, " textarea-error")),
				AttrName("description"),
			)(
				p.Description,
			),
			If(p.DescriptionErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.DescriptionErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_VIDEO"))(
				SPAN(AttrClass("label-text"))("Video"),
			),
			INPUT(
				AttrId("FORM_VIDEO"),
				AttrClass("file-input w-full "+IfElseZero(p.VideoErr != nil, "file-input-error")),
				AttrType(TypeFile),
				AttrName("video"),
				AttrAccept("video/*"),
				AttrRequired(true),
			),
			If(p.VideoErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.VideoErr),
				),
			),
		),

		DIV(AttrClass("pt-2"))(
			BUTTON(
				AttrClass("btn btn-primary w-full submit-button group"),
				AttrType(TypeSubmit),
			)(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Upload"),
				spinner(),
			),
		),
	)
}

type appPageLayoutParams struct {
	currentPage AppPageKind
	profile     NavbarProfile
}

type NavbarProfile struct {
	Username string
}

func appPageLayout(params appPageLayoutParams) ElementBuilder {
	return func(children ...any) Element {
		return basicPageLayout(basicLayoutParams{title: "Vidzilla"})(
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
							link:     "/@" + params.profile.Username,
							icon:     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user-icon lucide-user"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`,
							isActive: params.currentPage == PageProfile,
						}),
					),
				),
				MAIN(AttrClass("flex-1 p-6 overflow-y-auto"))(
					Group(children...),
				),
			),
		)
	}
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
		BUTTON(AttrClass("p-2 btn " + IfElseZero(params.isActive, "btn-primary")))(
			A(AttrHref(params.link))(RawText(params.icon)),
		),
	)
}
