package templates

import (
	"fmt"
	"net/url"
	"time"

	. "github.com/assaidy/hyper/v2"
	"github.com/assaidy/icons"
	"github.com/assaidy/icons/lucide"
	"github.com/google/uuid"
)

var strf = fmt.Sprintf

func basicPageLayout(title string) ChildrenInserter {
	clientId := uuid.New()

	return func(children ...any) Element {
		return Group(
			DOCTYPE(),
			HTML(AttrLang("en"))(
				HEAD()(
					META(AttrCharset("UTF-8")),
					META(AttrName("viewport"), AttrContent("width=device-width, initial-scale=1.0")),
					TITLE()(title),
					LINK(AttrRel("preconnect"), AttrHref("https://fonts.googleapis.com")),
					LINK(AttrRel("preconnect"), AttrHref("https://fonts.gstatic.com"), AttrCrossOrigin(CrossOriginAnonymous)),
					LINK(AttrRel("stylesheet"), AttrHref("https://fonts.googleapis.com/css2?family=Roboto:wght@400;500;700&display=swap")),
					LINK(AttrRel("stylesheet"), AttrHref("/assets/css/style.css")),
					SCRIPT(AttrSrc("/assets/js/lib/htmx@4.0.0_beta2.js")),
					SCRIPT(AttrSrc("/assets/js/main.js")),
				),
				BODY(
					AttrClass("min-h-screen bg-base-300"),
					Attr("hx-status:5xx:inherited", "swap:none"),
					Attr("data-client-id", clientId.String()),
				)(
					DIV(AttrId("alert-toast"), AttrClass("toast toast-top w-md z-[1000000]")),
					Group(children...),
				),
			),
		)
	}
}

type AlertLevel string

const (
	AlertInfo    AlertLevel = "info"
	AlertSuccess AlertLevel = "success"
	AlertWarning AlertLevel = "warning"
	AlertError   AlertLevel = "error"
)

func Alert(level AlertLevel, message string, timeout ...time.Duration) HyperNode {
	var icon string
	switch level {
	case AlertInfo:
		icon = lucide.Info()
	case AlertSuccess:
		icon = lucide.CircleCheck()
	case AlertWarning:
		icon = lucide.TriangleAlert()
	case AlertError:
		icon = lucide.CircleX()
	}

	t := 5 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}

	return DIV(AttrId("alert-toast"), AttrHxSwapOob(SwapPrepend))(
		DIV(
			AttrRole("alert"),
			AttrClass(strf("alert alert-%s", level)),
			AttrHxOn(EventHtmxAfterProcess, strf("setTimeout(() => this.remove(), %d)", t.Milliseconds())),
		)(
			RawText(icon), SPAN()(message),
		),
	)
}

func RegisterPage() HyperNode {
	return Once(func() HyperNode {
		return basicPageLayout("Register")(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body"))(
						H1(AttrClass("card-title text-3xl justify-center mb-4"))("Create Your Account"),
						RegisterForm(),
						P(AttrClass("text-center text-sm text-base-content/70 mt-4"))(
							"Already have an account? ",
							A(AttrClass("link link-primary"), AttrHref("/login"))("Login"),
						),
					),
				),
			),
		)
	})
}

type RegisterFormParams struct {
	Name        string
	NameErr     error
	Username    string
	UsernameErr error
	Email       string
	EmailErr    error
	Password    string
	PasswordErr error
}

func RegisterForm(params ...RegisterFormParams) HyperNode {
	var p RegisterFormParams
	if len(params) > 0 {
		p = params[0]
	}

	return FORM(
		AttrId("register-form"),
		AttrClass("space-y-4"),
		AttrHxPost("/register"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator("find .register-button"),
		AttrHxDisable("find button"),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-name"))(
				SPAN(AttrClass("label-text"))("Name"),
			),
			INPUT(
				AttrId("form-name"),
				AttrClass(Classes("input w-full", IfElseZero(p.NameErr != nil, "input-error"))),
				AttrType(TypeText),
				AttrName("name"),
				AttrValue(p.Name),
				AttrRequired(true),
			),
			If(p.NameErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.NameErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-username"))(
				SPAN(AttrClass("label-text"))("Username"),
			),
			INPUT(
				AttrId("form-username"),
				AttrClass(Classes("input w-full", IfElseZero(p.UsernameErr != nil, "input-error"))),
				AttrType(TypeText),
				AttrName("username"),
				AttrValue(p.Username),
				AttrRequired(true),
			),
			If(p.UsernameErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.UsernameErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-email"))(
				SPAN(AttrClass("label-text"))("Email"),
			),
			INPUT(
				AttrId("form-email"),
				AttrClass(Classes("input w-full", IfElseZero(p.EmailErr != nil, "input-error"))),
				AttrType(TypeEmail),
				AttrName("email"),
				AttrValue(p.Email),
				AttrRequired(true),
			),
			If(p.EmailErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.EmailErr),
				),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-password"))(
				SPAN(AttrClass("label-text"))("Password"),
			),
			INPUT(
				AttrId("form-password"),
				AttrClass(Classes("input w-full", IfElseZero(p.PasswordErr != nil, "input-error"))),
				AttrType(TypePassword),
				AttrName("password"),
				AttrValue(p.Password),
				AttrRequired(true),
			),
			If(p.PasswordErr != nil,
				LABEL(AttrClass("label"))(
					SPAN(AttrClass("label-text-alt text-error"))(p.PasswordErr),
				),
			),
		),

		DIV(AttrClass("pt-2"))(
			BUTTON(
				AttrClass("btn btn-primary w-full register-button group"),
				AttrType(TypeSubmit),
			)(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Register"),
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block")),
			),
		),
	)
}

func VerificationEmailSentPage() HyperNode {
	return Once(func() HyperNode {
		return basicPageLayout("Verification Email Sent")(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body text-center"))(
						DIV(AttrClass("text-warning mb-4"))(
							RawText(lucide.Mail(icons.Params{Class: "w-16 h-16 mx-auto"})),
						),
						H1(AttrClass("text-2xl font-bold"))("Verification Email Sent"),
						P(AttrClass("text-base-content/70"))("We have sent you a verification email. Please check your inbox."),
					),
				),
			),
		)
	})
}

func EmailVerifiedPage() HyperNode {
	return Once(func() HyperNode {
		return basicPageLayout("Email Verified")(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body text-center"))(
						DIV(AttrClass("text-success mb-4"))(
							RawText(lucide.MailCheck(icons.Params{Class: "w-16 h-16 mx-auto"})),
						),
						H1(AttrClass("text-2xl font-bold"))("Email Verified"),
						P(AttrClass("text-base-content/70"))("Your email has been verified successfully. You can now log in to your account."),
						DIV(AttrClass("card-actions justify-center mt-4"))(
							A(AttrClass("btn btn-primary"), AttrHref("/login"))("Go to Login"),
						),
					),
				),
			),
		)
	})
}

func InvalidVerificationLinkPage() HyperNode {
	return Once(func() HyperNode {
		return basicPageLayout("Invalid Verification Link")(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body text-center"))(
						DIV(AttrClass("text-error mb-4"))(
							RawText(lucide.MailWarning(icons.Params{Class: "w-16 h-16 mx-auto"})),
						),
						H1(AttrClass("text-2xl font-bold"))("Invalid Link"),
						P(AttrClass("text-base-content/70"))("This verification link is invalid or has expired."),
					),
				),
			),
		)
	})
}

func LoginPage() HyperNode {
	return Once(func() HyperNode {
		return basicPageLayout("Login")(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body"))(
						H1(AttrClass("card-title text-3xl justify-center mb-4"))("Login"),
						LoginForm(),
						P(AttrClass("text-center text-sm text-base-content/70 mt-4"))(
							"Don't have an account? ", A(AttrClass("link link-primary"), AttrHref("/register"))("Register"),
						),
					),
				),
			),
		)
	})
}

type LoginFormParams struct {
	Email    string
	Password string
	Err      LoginError
}

type LoginError int

const (
	_ LoginError = iota
	ErrInvalidCredentials
	ErrEmailNotVerified
)

func LoginForm(params ...LoginFormParams) HyperNode {
	var p LoginFormParams
	if len(params) > 0 {
		p = params[0]
	}

	return FORM(
		AttrId("login-form"),
		AttrClass("space-y-4"),
		AttrHxPost("/login"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator("find .login-button"),
		AttrHxDisable("find button"),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-email"))(
				SPAN(AttrClass("label-text"))("Email"),
			),
			INPUT(
				AttrId("form-email"),
				AttrClass(Classes("input w-full", IfElseZero(p.Err == ErrInvalidCredentials, "input-error"))),
				AttrType(TypeEmail),
				AttrName("email"),
				AttrValue(p.Email),
				AttrRequired(true),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-password"))(
				SPAN(AttrClass("label-text"))("Password"),
			),
			INPUT(
				AttrId("form-password"),
				AttrClass(Classes("input w-full", IfElseZero(p.Err == ErrInvalidCredentials, "input-error"))),
				AttrType(TypePassword),
				AttrName("password"),
				AttrValue(p.Password),
				AttrRequired(true),
			),
		),

		DIV(AttrClass("pt-2"))(
			BUTTON(AttrClass("btn btn-primary w-full login-button group"), AttrType(TypeSubmit))(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Login"),
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block")),
			),
		),

		If(p.Err == ErrInvalidCredentials,
			DIV(AttrClass("text-center text-error mt-2"))("Invalid email or password."),
		).ElseIf(p.Err == ErrEmailNotVerified,
			DIV(AttrClass("text-center text-error mt-2"))("Email not verified. Please check your inbox for verification email."),
		),
	)
}

func appPageContentLoader(contentPath string) HyperNode {
	return DIV(
		AttrHxGet(contentPath),
		AttrHxSwap(SwapOuterHtml),
		AttrHxTrigger("load"),
		AttrHxIndicator("#page-content-indicator"),
	)
}

type appPageLayoutParams struct {
	navbarParams NavbarParams
}

func appPageLayout(params appPageLayoutParams) ChildrenInserter {
	return func(children ...any) Element {
		return basicPageLayout("Vodzilla")(
			DIV(AttrClass("flex flex-col min-h-screen"))(
				Navbar(params.navbarParams),
				DIV(AttrClass("flex-1 relative pt-20"))(
					MAIN(AttrId("app-page-content"), AttrClass("w-full p-6"))(
						Group(children...),
					),
					DIV(
						AttrId("page-content-indicator"),
						AttrClass("hidden [.htmx-request]:flex absolute inset-0 items-center justify-center bg-base-300/70 z-10"),
					)(
						SPAN(AttrClass("loading loading-spinner loading-lg")),
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
	return NAV(AttrId("navbar"), AttrClass("w-full fixed top-0 z-10 py-2 flex justify-center"))(
		DIV(AttrClass("card bg-base-100 p-2 flex-row gap-2"))(
			appPageButton(appPageButtonParams{
				tab:      PageFeed,
				pageLink: "/feed",
				icon:     lucide.Timeline(),
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
				pageLink: "/watchlater",
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
			AttrHxGet(strf("%s/content", params.pageLink)),
			AttrHxPushUrl(params.pageLink),
			AttrHxTarget("#app-page-content"),
			AttrHxSwap(SwapInnerHtml),
			AttrHxIndicator("#page-content-indicator"),
		)(
			RawText(params.icon),
		),
	)
}

func ProfilePage(username string, profileUsername string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageProfile,
			Username:    username,
		},
	})(
		appPageContentLoader(strf("/@%s/content", profileUsername)),
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
		DIV(AttrClass("card bg-base-100 p-2 flex flex-col md:flex-row overflow-hidden"))(
			profileCardAvatarPlaceholder(),
			DIV(AttrClass("p-4 min-w-0"))(
				DIV(AttrClass("flex items-start justify-between gap-2"))(
					DIV(AttrClass("min-w-0 flex-1"))(
						H1(AttrId("profile-card-name"), AttrClass("text-2xl font-bold truncate"))(params.Name),
						P(AttrId("profile-card-username"), AttrClass("text-sm text-base-content/60"))("@"+params.Username),
					),
					If(!params.IsOwner,
						FollowButton(FollowButtonParams{ProfileOwnerId: params.OwnerId, IsFollowed: params.IsFollowed}),
					),
				),
				DIV(AttrClass("mt-2 flex gap-6"))(
					P()(SPAN(AttrClass("font-bold"))(params.FollowersCount), SPAN(AttrClass("text-base-content/60"))(" followers")),
					P()(SPAN(AttrClass("font-bold"))(params.PostsCount), SPAN(AttrClass("text-base-content/60"))(" posts")),
				),
				P(AttrId("profile-card-bio"), AttrClass("mt-2"))(IfElse(params.Bio == "", "---", params.Bio)),
			),
		),

		If(params.IsOwner,
			DIV(AttrClass("mt-4 flex justify-center lg:justify-start gap-2"))(
				BUTTON(AttrClass("btn btn-soft"), AttrOnClick("$('#edit-profile-modal').show()"))(
					RawText(lucide.UserPen()), "edit profile",
				),
				DIALOG(AttrId("edit-profile-modal"), AttrClass("modal"))(
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
				BUTTON(AttrClass("btn btn-soft"), AttrOnClick("$('#post-video-modal').show()"))(
					RawText(lucide.Plus()), "post a video",
				),
				DIALOG(AttrId("post-video-modal"), AttrClass("modal"))(
					DIV(AttrClass("modal-box"))(
						PostVideoForm(),
					),
					FORM(AttrMethod(MethodDialog), AttrClass("modal-backdrop"))(
						BUTTON()("close"),
					),
				),
			),
		),

		videosContainer(params.Videos),
	)
}

type FollowButtonParams struct {
	ProfileOwnerId uuid.UUID
	IsFollowed     bool
}

func FollowButton(params FollowButtonParams) HyperNode {
	return DIV(AttrId("follow-button"))(
		If(params.IsFollowed,
			BUTTON(
				AttrClass("btn btn-outline btn-accent hover:btn-error group/follow"),
				AttrHxDelete(strf("/follow/%s", params.ProfileOwnerId)),
				AttrHxTarget("#follow-button"),
				AttrHxSwap(SwapOuterHtml),
				AttrHxDisable("this"),
			)(
				SPAN(AttrClass("group-hover/follow:hidden"))("Following"),
				SPAN(AttrClass("hidden group-hover/follow:inline"))("Unfollow"),
			),
		).Else(
			BUTTON(
				AttrClass("btn btn-accent"),
				AttrHxPost(strf("/follow/%s", params.ProfileOwnerId)),
				AttrHxTarget("#follow-button"),
				AttrHxSwap(SwapOuterHtml),
				AttrHxDisable("this"),
			)(
				"Follow",
			),
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
		AttrId("edit-profile-form"),
		AttrClass("space-y-4"),
		AttrHxPut("/profiles"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator("find .submit-button"),
		AttrHxDisable("find .submit-button"),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-name"))(
				SPAN(AttrClass("label-text"))("Name"),
			),
			INPUT(
				AttrId("form-name"),
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
			LABEL(AttrClass("label"), AttrFor("form-username"))(
				SPAN(AttrClass("label-text"))("Username"),
			),
			INPUT(
				AttrId("form-username"),
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
			LABEL(AttrClass("label"), AttrFor("form-bio"))(
				SPAN(AttrClass("label-text"))("Bio"),
			),
			TEXTAREA(
				AttrId("form-bio"),
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
			BUTTON(AttrClass("btn btn-primary w-full submit-button group"), AttrType(TypeSubmit))(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Submit"),
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block")),
			),
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
		appPageContentLoader(strf("/videos/%s/content", videoId)),
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
	IsViewed                 bool
	IsFollowed               bool
	ReactionsParams          ReactionsWidgetParams
	WatchLaterButtonParams   WatchlaterButtonParams
	AddToPlaylistModalParams AddToPlaylistModalParams
	CurrentUserId            uuid.UUID
	CurrentUsername          string
}

func VideoPageContent(params VideoPageContentParams) HyperNode {
	ownerProfileLink := "/@" + params.OwnerUsername
	visitProfileAttrs := []Attribute{
		AttrHxGet(strf("%s/content", ownerProfileLink)),
		AttrHxPushUrl(ownerProfileLink),
		AttrHxTarget("#app-page-content"),
		AttrHxSwap(SwapInnerHtml),
		AttrHxTrigger("click consume"),
		AttrHxIndicator("#page-content-indicator"),
	}

	return DIV(AttrClass("max-w-6xl mx-auto"))(
		If(!params.IsViewed,
			DIV(AttrHxPost(strf("/videos/%s/views", params.Id)), AttrHxTrigger("load")),
		),
		VIDEO(
			AttrId("video-player"),
			AttrClass("w-full aspect-video bg-black rounded-box"),
			AttrSrc(params.SourceUrl),
			AttrControls(true),
			AttrPlaysInline(true),
		),
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

				ReactionsWidget(params.ReactionsParams),
				WatchlaterButton(params.WatchLaterButtonParams),
				addToPlaylistButton(),
				addToPlaylistModal(params.AddToPlaylistModalParams),
			),
			DIV(AttrClass("mt-4 card bg-base-200 p-4 space-y-2"))(
				DIV(AttrClass("text-sm text-base-content/60 flex gap-4"))(
					SPAN()(strf("%d views", params.ViewsCount)),
					SPAN()(params.Timestamp.Format(time.DateOnly)),
				),
				P(AttrClass("text-base-content/80"))(IfElse(params.Description == "", "---", params.Description)),
			),
		),

		commentSection(CommentSectionParams{
			VideoId:         params.Id,
			CurrentUsername: params.CurrentUsername,
		}),

		SCRIPT()(RawText(strf(`
			(() => {
				const v = $('#video-player')
				let attempts = 0
				v.addEventListener('error', async () => {
					if (v.error && v.error.code !== v.error.MEDIA_ERR_NETWORK) return
					if (++attempts > 3) return
					try {
						const r = await fetch('/videos/%s/stream_url')
						const d = await r.json()
						const t = v.currentTime
						const p = !v.paused
						v.src = d.url
						v.currentTime = t
						if (p) await v.play()
					} catch(e) {
						console.error(e)
					}
				})
				v.addEventListener('playing', () => { attempts = 0 })
			})
		`,
			params.Id,
		))),
	)
}

type PostVideoFormParams struct {
	Title            string
	TitleErr         error
	Description      string
	DescriptionErr   error
	VideoErr         error
	CloseDialogModal bool
}

func PostVideoForm(params ...PostVideoFormParams) HyperNode {
	var p PostVideoFormParams
	if len(params) > 0 {
		p = params[0]
	}

	pendingVideoId := uuid.New()

	return FORM(
		AttrId("post-video-form"),
		AttrClass("space-y-4"),
		AttrHxPost("/videos"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator("find .submit-button"),
		AttrHxDisable("find .submit-button"),
		AttrHxVals(strf(`js:{
			contentType:    $('#form-video').files[0].type,
			fileSize:       $('#form-video').files[0].size,
			pendingVideoId: %q,
		}`, pendingVideoId)),
		AttrHxOn(EventHtmxBeforeRequest, strf(`window._pendingVideos[%q] = $('#form-video').files[0]`, pendingVideoId)),
		AttrHxOn(EventHtmxAfterRequest, strf(`if (event.detail.ctx.response.status >= 400) delete window._pendingVideos[%q]`, pendingVideoId)),
		AttrHxOn(EventHtmxAfterProcess, IfElseZero(p.CloseDialogModal, `$('#post-video-modal').close()`)),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("form-title"))(
				SPAN(AttrClass("label-text"))("Title"),
			),
			INPUT(
				AttrId("form-title"),
				AttrClass(Classes("input w-full", IfElseZero(p.TitleErr != nil, "input-error"))),
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
			LABEL(AttrClass("label"), AttrFor("form-description"))(
				SPAN(AttrClass("label-text"))("Description"),
			),
			TEXTAREA(
				AttrId("form-description"),
				AttrClass(Classes("textarea block w-full", IfElseZero(p.DescriptionErr != nil, "textarea-error"))),
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
			LABEL(AttrClass("label"), AttrFor("form-video"))(
				SPAN(AttrClass("label-text"))("Video"),
			),
			INPUT(
				// doesn't have a name, so htmx will not send it with the form
				AttrId("form-video"),
				AttrClass(Classes("file-input w-full", IfElseZero(p.VideoErr != nil, "file-input-error"))),
				AttrType(TypeFile),
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
			BUTTON(AttrClass("btn btn-primary w-full submit-button group"), AttrType(TypeSubmit))(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Post"),
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block")),
			),
		),
	)
}

type VideoUploaderParams struct {
	PendingVideoId uuid.UUID
	VideoTitle     string
	VideoId        uuid.UUID
	UploadId       string
	PartSize       int64
	UploadUrls     []string
}

// TODO: refactor this to use htmx and ws to store progress in media service
// so all user sessions can see the uploading videos (with the initiating session)
// and also the processing ones.
func VideoUploader(params VideoUploaderParams) HyperNode {
	return SCRIPT()(RawText(strf(`
		(async () => {
			const pendingVideoId = %q
			const videoTitle     = %q
			const videoId        = %q
			const uploadId       = %q
			const partSize       = %d
			const uploadUrls     = %s

			try {
				await window._videoUploadManager.upload({
					pendingVideoId,
					videoTitle,
					videoId,
					uploadId,
					partSize,
					uploadUrls,
					completeUploadUrl: "/videos/complete_upload",
				})
			} catch (err) {
				console.error(err)
				window._videoUploadManager.removeUpload(pendingVideoId)
			}
		})
	`,
		params.PendingVideoId,
		params.VideoTitle,
		params.VideoId,
		params.UploadId,
		params.PartSize,
		Json(params.UploadUrls),
	)))
}

func videoUploadersContainer() HyperNode {
	return DIV(AttrId("video-uploaders-container"))(
		SCRIPT()(RawText(`
			window._pendingVideos = {}

			window._videoUploadManager = {
				_uploads: {},
				addUpload(id, title, totalChunks) {
					this._uploads[id] = { title, totalChunks, completedChunks: 0 }
					this._updateIndicator()
				},
				markChunkComplete(id) {
					const u = this._uploads[id]
					if (!u) return
					u.completedChunks++
					if (u.completedChunks >= u.totalChunks) delete this._uploads[id]
					this._updateIndicator()
				},
				removeUpload(id) {
					delete this._uploads[id]
					this._updateIndicator()
				},
				_updateIndicator() {
					const count = Object.keys(this._uploads).length
					$('#upload-indicator-count').textContent = count
					this._renderUploadList()
					$('#upload-indicator').classList.toggle('hidden', count === 0)
				},
				_renderUploadList() {
					const entries = Object.entries(this._uploads)
					if (entries.length === 0) {
						$('#upload-list-dialog').close()
						return
					}
					let html = ''
					for (const [, u] of entries) {
						html += '<div class="flex flex-col gap-1 py-2">'
						     +  '<div class="flex justify-between text-sm">'
						     +  '<span class="truncate">' + u.title + '</span>'
						     +  '<span class="shrink-0">' + u.completedChunks + '/' + u.totalChunks + '</span>'
						     +  '</div>'
						     +  '<progress class="progress progress-primary w-full" value="' + u.completedChunks + '" max="' + u.totalChunks + '"></progress>'
						     +  '</div>'
					}
					$('#upload-list-body').innerHTML = html
				},
			  async upload({ pendingVideoId, videoTitle, partSize, uploadUrls, videoId, uploadId, completeUploadUrl }) {
					this.addUpload(pendingVideoId, videoTitle, uploadUrls.length)

					const file = window._pendingVideos[pendingVideoId]
					if (!file) throw new Error("pending video not found")

					const completedParts = []
					const uploads = uploadUrls.map(async (url, i) => {
						const start = i * partSize
						const end = i === uploadUrls.length - 1 ? file.size : start + partSize
						const blob = file.slice(start, end)

						const response = await fetch(url, { method: 'PUT', body: blob })
						if (!response.ok) throw new Error("upload failed")

						completedParts.push({
							etag: (response.headers.get('ETag') ?? '').replaceAll('"', ''),
							partNumber: i + 1,
						})

						this.markChunkComplete(pendingVideoId)
					})

					await Promise.all(uploads)

					await fetch(completeUploadUrl, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ videoId, uploadId, parts: completedParts }),
					})

					delete window._pendingVideos[pendingVideoId]
			 	}
			}
	`)),
	)
}

func videoUploadIndicator() HyperNode {
	return DIV(
		AttrId("upload-indicator"),
		AttrClass("hidden fixed bottom-6 right-6 z-50"),
	)(
		DIV(AttrClass("indicator"))(
			BUTTON(
				AttrClass("btn btn-circle btn-primary btn-lg shadow-lg relative"),
				AttrOnClick("$('#upload-list-dialog').showModal()"),
			)(
				RawText(lucide.ArrowUpFromLine(icons.Params{Class: "w-5 h-5 animate-bounce"})),
			),
			SPAN(
				AttrId("upload-indicator-count"),
				AttrClass("indicator-item indicator-bottom indicator-center badge badge-secondary"),
			)("0"),
		),
		DIALOG(AttrId("upload-list-dialog"), AttrClass("modal"))(
			DIV(AttrClass("modal-box"))(
				H3(AttrClass("text-lg font-bold"))("Uploading Videos"),
				DIV(AttrId("upload-list-body"), AttrClass("mt-4 space-y-2")),
			),
			FORM(AttrMethod(MethodDialog), AttrClass("modal-backdrop"))(
				BUTTON()("close"),
			),
		),
	)
}

func videosContainer(videos []VideoCardParams) HyperNode {
	return DIV(AttrClass("mt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"))(
		Range(videos, func(p VideoCardParams) any {
			return VideoCard(p)
		}),
	)
}

type VideoCardParams struct {
	VideoId       uuid.UUID
	Title         string
	Timestamp     time.Time
	OwnerName     string
	OwnerUsername string
	ViewsCount    int
	ThumbnailUrl  string
	AvatarUrl     string
}

func VideoCard(params VideoCardParams) HyperNode {
	ownerProfilePageLink := "/@" + params.OwnerUsername
	videoPageLink := strf("/videos/%s", params.VideoId)
	visiteProfileAttrs := []Attribute{
		AttrHxGet(strf("%s/content", ownerProfilePageLink)),
		AttrHxPushUrl(ownerProfilePageLink),
		AttrHxTarget("#app-page-content"),
		AttrHxSwap(SwapInnerHtml),
		AttrHxTrigger("click consume"),
		AttrHxIndicator("#page-content-indicator"),
	}

	return DIV(
		AttrClass("card bg-base-100 cursor-pointer"),
		AttrHxGet(strf("%s/content", videoPageLink)),
		AttrHxPushUrl(videoPageLink),
		AttrHxTarget("#app-page-content"),
		AttrHxSwap(SwapInnerHtml),
		AttrHxIndicator("#page-content-indicator"),
	)(
		FIGURE(AttrClass("relative aspect-video overflow-hidden"))(
			DIV(AttrClass("w-full h-full"))(
				If(params.ThumbnailUrl != "",
					IMG(AttrClass("w-full h-full object-cover"), AttrSrc(params.ThumbnailUrl)),
				).Else(
					videoCardThumbnailPlaceholder(),
				),
			),
		),
		DIV(AttrClass("card-body flex flex-row gap-3 p-3"))(
			DIV()(
				DIV(append(visiteProfileAttrs, AttrClass("shrink-0 cursor-pointer"))...)(
					If(params.AvatarUrl != "",
						IMG(AttrClass("w-9 h-9 rounded-full"), AttrSrc(params.AvatarUrl)),
					).Else(
						videoCardOwnerAvatarPlaceholder(),
					),
				),
			),
			DIV(AttrClass("min-w-0 flex-1"))(
				H2(AttrClass("card-title text-base font-bold leading-tight line-clamp-2"))(params.Title),
				A(append(visiteProfileAttrs, AttrClass("link link-hover text-xs text-base-content/60"))...)(params.OwnerName),
				DIV(AttrClass("text-xs text-base-content/60"))(
					normalizeViewsCount(params.ViewsCount), " views", " . ", normalizeTimestamp(params.Timestamp), " ago",
				),
			),
		),
	)
}

func normalizeTimestamp(t time.Time) any {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute"
		}
		return strf("%d minutes", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour"
		}
		return strf("%d hours", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return strf("%d days", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / (24 * 30))
		if months == 1 {
			return "1 month"
		}
		return strf("%d months", months)
	default:
		years := int(d.Hours() / (24 * 365))
		if years == 1 {
			return "1 year"
		}
		return strf("%d years", years)
	}
}

func normalizeViewsCount(i int) any {
	switch {
	case i >= 1_000_000_000:
		return strf("%.1fB", float64(i)/1_000_000_000)
	case i >= 1_000_000:
		return strf("%.1fM", float64(i)/1_000_000)
	case i >= 1_000:
		return strf("%.1fK", float64(i)/1_000)
	default:
		return strf("%d", i)
	}
}

func videoCardOwnerAvatarPlaceholder() HyperNode {
	return DIV(AttrClass("avatar placeholder"))(
		DIV(AttrClass("bg-neutral text-neutral-content rounded-full w-10 h-10 flex items-center justify-center text-xs"))(
			RawText(lucide.User()),
		),
	)
}

type WatchlaterButtonParams struct {
	VideoId  uuid.UUID
	IsActive bool
}

func WatchlaterButton(params WatchlaterButtonParams) HyperNode {
	return DIV(AttrId("watchlater-button"))(
		If(params.IsActive,
			BUTTON(
				AttrClass("btn btn-soft btn-sm tooltip tooltip-top"),
				Attr("data-tip", "Remove from Watch Later"),
				AttrHxDelete(strf("/videos/%s/watchlater", params.VideoId)),
				AttrHxTarget("#watchlater-button"),
				AttrHxSwap(SwapOuterHtml),
			)(
				RawText(lucide.Clock(icons.Params{Class: "text-primary"})),
			),
		).Else(
			BUTTON(
				AttrClass("btn btn-soft btn-sm tooltip tooltip-top"),
				Attr("data-tip", "Add to Watch Later"),
				AttrHxPost(strf("/videos/%s/watchlater", params.VideoId)),
				AttrHxTarget("#watchlater-button"),
				AttrHxSwap(SwapOuterHtml),
			)(
				RawText(lucide.Clock()),
			),
		),
	)
}

type PlaylistCheckboxParams struct {
	VideoId    uuid.UUID
	PlaylistId uuid.UUID
	Name       string
	Checked    bool
}

func PlaylistCheckbox(params PlaylistCheckboxParams) HyperNode {
	return DIV(AttrId("playlist-checkbox"), AttrClass("form-control"))(
		LABEL(AttrClass("label cursor-pointer flex justify-between"))(
			SPAN(AttrClass("text-base-content"))(params.Name),
			INPUT(
				AttrType(TypeCheckbox),
				AttrClass("checkbox checkbox-sm checkbox-primary"),
				IfElse(params.Checked,
					AttrHxDelete(strf("/videos/%s/playlists/%s", params.VideoId, params.PlaylistId)),
					AttrHxPost(strf("/videos/%s/playlists/%s", params.VideoId, params.PlaylistId)),
				),
				AttrHxTarget("#playlist-checkbox"),
				AttrHxSwap(SwapOuterHtml),
				AttrHxVals(Json(Object{"playlistName": params.Name})),
				AttrChecked(params.Checked),
			),
		),
	)
}

type AddToPlaylistModalParams struct {
	VideoId   uuid.UUID
	Playlists []PlaylistCheckboxParams
}

func addToPlaylistButton() HyperNode {
	return DIV(AttrClass("tooltip tooltip-top"), Attr("data-tip", "Add to Playlist"))(
		BUTTON(AttrClass("btn btn-soft btn-sm"), AttrOnClick("$('#add-to-playlist-modal').show()"))(
			RawText(lucide.ListVideo()),
		),
	)
}

func addToPlaylistModal(params AddToPlaylistModalParams) HyperNode {
	return DIALOG(AttrId("add-to-playlist-modal"), AttrClass("modal"))(
		DIV(AttrClass("modal-box"))(
			H3(AttrClass("text-lg font-bold mb-4"))("Add to Playlist"),
			DIV(AttrId("playlist-checkbox-list"), AttrClass("space-y-2"))(
				If(len(params.Playlists) == 0,
					P(AttrClass("text-sm text-base-content/60"))("No playlists yet. Create one below."),
				).Else(
					Range(params.Playlists, func(p PlaylistCheckboxParams) any {
						return PlaylistCheckbox(p)
					}),
				),
			),
			DIV(AttrClass("divider my-4")),
			CreatePlaylistForm(CreatePlaylistFormParams{VideoId: params.VideoId}),
		),
		FORM(AttrMethod(MethodDialog), AttrClass("modal-backdrop"))(
			BUTTON()("close"),
		),
	)
}

type CreatePlaylistFormParams struct {
	VideoId uuid.UUID
	Name    string
	NameErr error
}

func CreatePlaylistForm(params ...CreatePlaylistFormParams) HyperNode {
	var p CreatePlaylistFormParams
	if len(params) > 0 {
		p = params[0]
	}

	return FORM(
		AttrId("create-playlist-form"),
		AttrClass("flex gap-2"),
		AttrHxPost("/playlists"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxTarget("this"),
		AttrHxVals(Json(Object{"videoId": p.VideoId})),
	)(
		INPUT(
			AttrId("form-playlist-name"),
			AttrType(TypeText),
			AttrName("name"),
			AttrValue(p.Name),
			AttrPlaceholder("New playlist name..."),
			AttrClass(Classes("input w-full", IfElseZero(p.NameErr != nil, "input-error"))),
			AttrRequired(true),
		),
		BUTTON(AttrClass("btn btn-primary"), AttrType(TypeSubmit))("Create"),
		If(p.NameErr != nil,
			P(AttrClass("text-xs text-error"))(p.NameErr),
		),
	)
}

func videoCardThumbnailPlaceholder() HyperNode {
	return DIV(AttrClass("w-full h-full flex items-center justify-center bg-base-200"))(
		RawText(lucide.Clapperboard(icons.Params{Class: "w-10 h-10 text-base-content/30"})),
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
	return DIV(AttrId("reactions-widget"), AttrClass("join ml-auto"))(
		BUTTON(
			AttrClass("join-item btn btn-soft btn-sm tooltip tooltip-top"),
			Attr("data-tip", "Like"),
			IfElse(params.IsLiked,
				AttrHxDelete(strf("/videos/%s/reactions?kind=like", params.VideoId)),
				AttrHxPost(strf("/videos/%s/reactions?kind=like", params.VideoId)),
			),
			AttrHxTarget("#reactions-widget"),
			AttrHxSwap(SwapOuterHtml),
		)(
			RawText(lucide.ThumbsUp(icons.Params{Class: IfElseZero(params.IsLiked, "text-primary")})),
			SPAN(AttrClass("text-sm"))(params.LikesCount),
		),
		BUTTON(
			AttrClass("join-item btn btn-soft btn-sm tooltip tooltip-top"),
			Attr("data-tip", "Dislike"),
			IfElse(params.IsDisliked,
				AttrHxDelete(strf("/videos/%s/reactions?kind=dislike", params.VideoId)),
				AttrHxPost(strf("/videos/%s/reactions?kind=dislike", params.VideoId)),
			),
			AttrHxTarget("#reactions-widget"),
			AttrHxSwap(SwapOuterHtml),
		)(
			RawText(lucide.ThumbsDown(icons.Params{Class: IfElseZero(params.IsDisliked, "text-primary")})),
			SPAN(AttrClass("text-sm"))(params.DislikesCount),
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
		appPageContentLoader("/playlists/content"),
	)
}

func PlaylistsPageContent(params PlaylistsPageContentParams) HyperNode {
	return DIV(AttrId("playlists-container"), AttrClass("space-y-4"))(
		H1(AttrClass("text-2xl font-bold mb-4"))("Playlists"),
		If(len(params.Playlists) == 0,
			P(AttrClass("text-center text-base-content/60 mt-20"))("No playlists yet."),
		).Else(
			Range(params.Playlists, func(p PlaylistCardParams) any {
				playlistLink := strf("/playlists/%s", p.Id)
				return DIV(
					AttrClass("card bg-base-100 p-0 cursor-pointer transition-shadow duration-200 flex flex-row items-stretch overflow-hidden"),
					AttrHxGet(strf("%s/content", playlistLink)),
					AttrHxPushUrl(playlistLink),
					AttrHxTarget("#app-page-content"),
					AttrHxSwap(SwapInnerHtml),
					AttrHxIndicator("#page-content-indicator"),
				)(
					DIV(AttrClass("flex items-center justify-center bg-base-200 px-6"))(
						RawText(lucide.ListVideo(icons.Params{Class: "text-base-content/80"})),
					),
					DIV(AttrClass("p-4"))(
						H2(AttrClass("card-title text-lg font-bold"))(p.Name),
						P(AttrClass("text-sm text-base-content/60"))(strf("%d videos", p.VideosCount)),
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
		appPageContentLoader(strf("/playlists/%s/content", playlistId)),
	)
}

func PlaylistDetailPageContent(params PlaylistDetailPageContentParams) HyperNode {
	return DIV(AttrId("playlist-detail-container"))(
		H1(AttrClass("text-2xl font-bold mb-6"))(params.Playlist.Name),
		If(len(params.Videos) == 0,
			P(AttrClass("text-center text-base-content/60 mt-20"))("This playlist is empty."),
		).Else(
			videosContainer(params.Videos),
		),
	)
}

type WatchlaterPageContentParams struct {
	Username string
	Videos   []VideoCardParams
}

func WatchlaterPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageWatchLater,
			Username:    username,
		},
	})(
		appPageContentLoader("/watchlater/content"),
	)
}

func WatchlaterPageContent(params WatchlaterPageContentParams) HyperNode {
	return Group(
		H1(AttrClass("text-2xl font-bold mb-4"))("Watch Later"),
		If(len(params.Videos) == 0,
			P(AttrClass("text-center text-base-content/60 mt-20"))("No videos in your watch later list."),
		).Else(
			videosContainer(params.Videos),
		),
	)
}

type CommentSectionParams struct {
	VideoId         uuid.UUID
	CurrentUsername string
}

func commentSection(params CommentSectionParams) HyperNode {
	return DIV(AttrId("comment-section"), AttrClass("mt-8"))(
		SCRIPT(AttrSrc("/assets/js/comment_section.js"), AttrDefer(true)),
		H2(AttrClass("text-xl font-bold mb-4"))("Comments"),
		CreateCommentForm(CreateCommentFormParams{VideoId: params.VideoId, CurrentUsername: params.CurrentUsername}),
		DIV(AttrId("comments-list"), AttrClass("mt-4 space-y-4"))(
			CommentsLoader(CommentsLoaderParams{VideoId: params.VideoId}),
		),
	)
}

type CreateCommentFormParams struct {
	VideoId         uuid.UUID
	CurrentUsername string
	ContentErr      error
}

func CreateCommentForm(params CreateCommentFormParams) HyperNode {
	return FORM(
		AttrId("create-comment-form"),
		AttrClass("w-full flex flex-col gap-2 bg-base-100 p-2 rounded-xl border border-base-300 shadow-sm focus-within:border-primary focus-within:ring-1 focus-within:ring-primary transition-all"),
		AttrHxPost(strf("/videos/%s/comments", params.VideoId)),
		AttrHxTarget("this"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator("#comment-post-btn"),
		AttrHxDisable("#comment-buttons button"),
	)(
		TEXTAREA(
			AttrId("comment-input"),
			AttrClass(Classes(
				"textarea textarea-ghost outline-none w-full resize-none field-sizing-content overflow-hidden min-h-[1lh]",
				IfElseZero(params.ContentErr != nil, "textarea-error"),
			)),
			AttrRows("1"),
			AttrMaxLength("500"),
			AttrPlaceholder("Write a cool comment..."),
			AttrName("comment"),
			AttrRequired(true),
		),

		DIV(AttrClass("flex justify-between items-center border-t border-base-300 pt-2"))(
			DIV(AttrClass("flex items-center gap-2"))(
				commentOwnerAvatarPlaceholder(),
				SPAN(AttrClass("text-sm font-medium"))("@"+params.CurrentUsername),
				If(params.ContentErr != nil,
					SPAN(AttrClass("text-xs text-error"))(params.ContentErr),
				),
			),
			DIV(AttrId("comment-buttons"), AttrClass("flex gap-2"))(
				BUTTON(AttrId("comment-clear-btn"), AttrClass("btn btn-soft btn-error btn-sm"), AttrType(TypeButton))("Clear"),
				BUTTON(AttrId("comment-post-btn"), AttrClass("btn btn-primary btn-sm group"), AttrType(TypeSubmit))(
					"Post Comment", SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block")),
				),
			),
		),
	)
}

type CreateReplyFormParams struct {
	VideoId    uuid.UUID
	CommentId  uuid.UUID
	ContentErr error
}

func CreateReplyForm(params CreateReplyFormParams) HyperNode {
	return FORM(
		AttrId(strf("create-reply-form-%s", params.CommentId)),
		AttrClass("w-full flex flex-col gap-2 bg-base-100 p-2 rounded-xl border border-base-300 shadow-sm focus-within:border-primary focus-within:ring-1 focus-within:ring-primary transition-all"),
		AttrHxPost(strf("/videos/%s/comments/%s/replies", params.VideoId, params.CommentId)),
		AttrHxTarget("this"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator(strf("#reply-post-btn-%s", params.CommentId)),
		AttrHxDisable(strf("#reply-buttons-%s button", params.CommentId)),
	)(
		TEXTAREA(
			AttrId(strf("reply-input-%s", params.CommentId)),
			AttrClass(Classes(
				"textarea textarea-ghost outline-none w-full resize-none field-sizing-content overflow-hidden min-h-[1lh]",
				IfElseZero(params.ContentErr != nil, "textarea-error"),
			)),
			AttrRows("1"),
			AttrMaxLength("500"),
			AttrPlaceholder(strf("Write a cool reply...")),
			AttrName("comment"),
			AttrRequired(true),
		),

		DIV(AttrClass("flex items-center border-t border-base-300 pt-2"))(
			If(params.ContentErr != nil,
				SPAN(AttrClass("text-xs text-error"))(params.ContentErr),
			),
			DIV(AttrClass("flex gap-2 ml-auto"), AttrId(strf("reply-buttons-%s", params.CommentId)))(
				BUTTON(
					AttrClass("btn btn-soft btn-error btn-sm"),
					AttrType(TypeButton),
					Attr("data-reply-cancel", ""),
					Attr("data-comment-id", params.CommentId.String()),
				)(
					"Cancel",
				),
				BUTTON(AttrId(strf("reply-post-btn-%s", params.CommentId)), AttrClass("btn btn-primary btn-sm group"), AttrType(TypeSubmit))(
					"Reply", SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block")),
				),
			),
		),
	)
}

type CommentsLoaderParams struct {
	VideoId       uuid.UUID
	LastCommentId uuid.UUID
}

func CommentsLoader(params CommentsLoaderParams) HyperNode {
	var urlQuery string
	if params.LastCommentId != uuid.Nil {
		urlQuery = strf("last_comment_id=%s", params.LastCommentId)
	} else {
		// This is to avoid duplication when posting a comment before initail load.
		urlQuery = strf("max_timestamp=%s", url.QueryEscape(time.Now().Format(time.RFC3339)))
	}

	return DIV(
		AttrHxGet(strf("/videos/%s/comments?%s", params.VideoId, urlQuery)),
		AttrHxTrigger("intersect"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator("find .htmx-indicator"),
	)(
		DIV(AttrClass("htmx-indicator hidden [.htmx-request]:flex justify-center pt-4"))(
			SPAN(AttrClass("loading loading-spinner loading-sm")),
		),
	)
}

type CommentParams struct {
	Id            uuid.UUID
	VideoId       uuid.UUID
	OwnerUsername string
	Content       string
	CreatedAt     time.Time
	IsOwner       bool
}

func Comment(params CommentParams) HyperNode {
	const commentCharLimit = 100
	isLong := len(params.Content) > commentCharLimit
	var displayContent string
	if isLong {
		displayContent = params.Content[:commentCharLimit] + "..."
	} else {
		displayContent = params.Content
	}

	return DIV(AttrId(strf("comment-%s", params.Id)), AttrClass("flex gap-2"))(
		commentOwnerAvatarPlaceholder(),
		DIV(AttrClass("flex-1"))(
			DIV(AttrClass("flex items-center gap-2 text-sm"))(
				A(AttrClass("link link-hover font-bold"), AttrHref("/@"+params.OwnerUsername))("@"+params.OwnerUsername),
				SPAN(AttrClass("text-base-content/60 text-xs"))(normalizeTimestamp(params.CreatedAt)),
			),
			DIV(AttrClass("comment-text-container"))(
				P(AttrClass("text-sm mt-1"))(displayContent),
				If(isLong,
					BUTTON(AttrClass("comment-read-more text-xs link link-primary mt-1"), Attr("data-full-text", params.Content))("Read more"),
				),
			),
			DIV(AttrClass("flex items-center gap-1 mt-1 text-base-content/60"))(
				BUTTON(
					AttrClass("btn btn-ghost btn-xs"),
					Attr("data-reply-toggle", ""),
					Attr("data-comment-id", params.Id.String()),
				)(
					RawText(lucide.Reply()), " Reply",
				),
				If(params.IsOwner,
					BUTTON(
						AttrClass("btn btn-ghost btn-xs"),
						AttrHxDelete(strf("/videos/%s/comments/%s", params.VideoId, params.Id)),
						AttrHxTarget(strf("#comment-%s", params.Id)),
						AttrHxSwap(SwapDelete),
						AttrHxDisable("this"),
					)(
						RawText(lucide.Trash2(icons.Params{Class: "w-4 h-4"})), " Delete",
					),
				),
				SPAN(AttrId(strf("show-replies-btn-%s", params.Id)))(
					ShowRepliesButton(params.Id),
				),
			),
			DIV(AttrId(strf("reply-form-%s", params.Id)), AttrClass("hidden mt-2"))(
				CreateReplyForm(CreateReplyFormParams{VideoId: params.VideoId, CommentId: params.Id}),
			),
			DIV(AttrId(strf("replies-%s", params.Id)), AttrClass("hidden mt-2 border-l-2 border-base-300 space-y-2"))(
				RepliesLoader(RepliesLoaderParams{VideoId: params.VideoId, CommentId: params.Id}),
			),
		),
	)
}

// TODO : this is always visible event if the comment doesn't have any replies.
// it's confusing to the user.
// handle state for comments/replies count.
func ShowRepliesButton(commentId uuid.UUID) HyperNode {
	return BUTTON(
		AttrClass("btn btn-ghost btn-xs"),
		Attr("data-replies-toggle", ""),
		Attr("data-comment-id", commentId.String()),
		Attr("data-view-text", "View replies"),
	)(
		"View replies",
	)
}

type RepliesLoaderParams struct {
	VideoId       uuid.UUID
	LastCommentId uuid.UUID
	CommentId     uuid.UUID
}

func RepliesLoader(params RepliesLoaderParams) HyperNode {
	var urlQuery string
	if params.LastCommentId != uuid.Nil {
		urlQuery = strf("last_comment_id=%s", params.LastCommentId)
	} else {
		// This is to avoid duplication when posting a reply before initail load.
		urlQuery = strf("max_timestamp=%s", url.QueryEscape(time.Now().Format(time.RFC3339)))
	}

	return DIV(
		AttrHxGet(strf("/videos/%s/comments/%s/replies?%s", params.VideoId, params.CommentId, urlQuery)),
		AttrHxTrigger("intersect"),
		AttrHxSwap(SwapOuterHtml),
		AttrHxIndicator("find .htmx-indicator"),
	)(
		DIV(AttrClass("htmx-indicator hidden [.htmx-request]:flex justify-center pt-4"))(
			SPAN(AttrClass("loading loading-spinner loading-sm")),
		),
	)
}

func commentOwnerAvatarPlaceholder() HyperNode {
	return DIV(AttrClass("avatar placeholder shrink-0"))(
		DIV(AttrClass("bg-neutral text-neutral-content rounded-full w-8 h-8 flex items-center justify-center text-xs"))(
			RawText(lucide.User()),
		),
	)
}

func DiscoverPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageDiscover,
			Username:    username,
		},
	})(
		appPageContentLoader("/discover/content"),
	)
}

func DiscoverPageContent() HyperNode {
	return Text("Discover Page")
}

func FeedPage(username string) HyperNode {
	return appPageLayout(appPageLayoutParams{
		navbarParams: NavbarParams{
			CurrentPage: PageFeed,
			Username:    username,
		},
	})(
		appPageContentLoader("/feed/content"),
	)
}

func FeedPageContent(videos []VideoCardParams) HyperNode {
	return Group(
		H1(AttrClass("text-2xl font-bold mb-4"))("Your Feed"),
		If(len(videos) == 0,
			DIV(AttrClass("text-center mt-20 space-y-2"))(
				P(AttrClass("text-base-content/60"))("No videos in your feed yet."),
				P(AttrClass("text-sm text-base-content/40"))("Follow some users to see their latest videos here."),
			),
		).Else(
			videosContainer(videos),
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
		appPageContentLoader("/notifications/content"),
	)
}

func NotificationsPageContent() HyperNode {
	return Text("Notifications Page")
}
