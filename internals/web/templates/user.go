package templates

import (
	. "github.com/assaidy/hyper/v2"
)

// TODO: adapt forms to daisy ui

var registerPageCache NodeCache

func RegisterPage() HyperNode {
	return Cache(&registerPageCache,
		page("Register", Group(
			DIV(AttrClass("min-h-screen flex items-center justify-center bg-yt-bg"))(
				DIV(AttrClass("w-full max-w-md p-8 bg-yt-surface rounded-lg shadow-lg"))(
					H1(AttrClass("text-2xl font-bold text-center mb-6 text-yt-text"))("Create Your Account"),
					RegisterForm(),
					P(AttrClass("mt-4 text-center text-yt-text-secondary text-sm"))(
						"Already have an account? ",
						A(AttrClass("text-yt-red hover:text-yt-red-hover font-medium"), AttrHref("/login"))("Login"),
					),
				),
			),
		)),
	)
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
		AttrID("register-form"),
		AttrClass("space-y-4"),
		Attr("hx-post", "/register"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .register-button"),
		Attr("hx-disabled-elt", "find button"),
	)(
		DIV()(
			LABEL(AttrClass("block text-sm font-medium text-yt-text mb-1"), AttrFor("name"))("Name"),
			INPUT(AttrClass("w-full px-4 py-2 bg-yt-surface-hover border border-yt-border rounded text-yt-text placeholder-yt-text-secondary focus:outline-none focus:border-yt-red"), AttrType(TypeText), AttrID("name"), AttrName("name"), AttrValue(p.Name), AttrRequired(true)),
			If(p.NameErr != nil, P(AttrClass("mt-1 text-sm text-red-500"))(p.NameErr)),
		),
		DIV()(
			LABEL(AttrClass("block text-sm font-medium text-yt-text mb-1"), AttrFor("username"))("Username"),
			INPUT(AttrClass("w-full px-4 py-2 bg-yt-surface-hover border border-yt-border rounded text-yt-text placeholder-yt-text-secondary focus:outline-none focus:border-yt-red"), AttrType(TypeText), AttrID("username"), AttrName("username"), AttrValue(p.Username), AttrRequired(true)),
			If(p.UsernameErr != nil, P(AttrClass("mt-1 text-sm text-red-500"))(p.UsernameErr)),
		),
		DIV()(
			LABEL(AttrClass("block text-sm font-medium text-yt-text mb-1"), AttrFor("email"))("Email"),
			INPUT(AttrClass("w-full px-4 py-2 bg-yt-surface-hover border border-yt-border rounded text-yt-text placeholder-yt-text-secondary focus:outline-none focus:border-yt-red"), AttrType(TypeEmail), AttrID("email"), AttrName("email"), AttrValue(p.Email), AttrRequired(true)),
			If(p.EmailErr != nil, P(AttrClass("mt-1 text-sm text-red-500"))(p.EmailErr)),
		),
		DIV()(
			LABEL(AttrClass("block text-sm font-medium text-yt-text mb-1"), AttrFor("password"))("Password"),
			INPUT(AttrClass("w-full px-4 py-2 bg-yt-surface-hover border border-yt-border rounded text-yt-text placeholder-yt-text-secondary focus:outline-none focus:border-yt-red"), AttrType(TypePassword), AttrID("password"), AttrName("password"), AttrValue(p.Password), AttrRequired(true)),
			If(p.PasswordErr != nil, P(AttrClass("mt-1 text-sm text-red-500"))(p.PasswordErr)),
		),
		DIV(AttrClass("pt-2"))(
			BUTTON(
				AttrClass("btn group register-button w-full font-medium"),
				AttrType(TypeSubmit),
			)(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Register"),
				spinner(),
			),
		),
	)
}

var verificationEmailSentPageCache NodeCache

func VerificationEmailSentPage() HyperNode {
	return Cache(&verificationEmailSentPageCache,
		page("Verification Email Sent", Group(
			DIV(AttrClass("min-h-screen flex items-center justify-center bg-yt-bg"))(
				DIV(AttrClass("w-full max-w-md p-8 bg-yt-surface rounded-lg shadow-lg text-center"))(
					DIV(AttrClass("mb-4"))(RawText(`<svg class="w-16 h-16 mx-auto text-yellow-500" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-mail-open-icon lucide-mail-open"><path d="M21.2 8.4c.5.38.8.97.8 1.6v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V10a2 2 0 0 1 .8-1.6l8-6a2 2 0 0 1 2.4 0l8 6Z"/><path d="m22 10-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 10"/></svg>`)),
					H1(AttrClass("text-2xl font-bold text-yt-text mb-4"))("Verifictaion Email Sent"),
					P(AttrClass("text-yt-text-secondary mb-6"))("We have sent you a verification email. Please check your inbox."),
					// TODO: resend verification email
				),
			),
		)),
	)

}

var emailVerifiedPageCache NodeCache

func EmailVerifiedPage() HyperNode {
	return Cache(&emailVerifiedPageCache,
		page("Email Verified", Group(
			DIV(AttrClass("min-h-screen flex items-center justify-center bg-yt-bg"))(
				DIV(AttrClass("w-full max-w-md p-8 bg-yt-surface rounded-lg shadow-lg text-center"))(
					DIV(AttrClass("mb-4"))(RawText(`<svg class="w-16 h-16 mx-auto text-green-500" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>`)),
					H1(AttrClass("text-2xl font-bold text-yt-text mb-4"))("Email Verified"),
					P(AttrClass("text-yt-text-secondary mb-6"))("Your email has been verified successfully. You can now log in to your account."),
					BUTTON(
						AttrClass("py-2 px-6 bg-yt-red hover:bg-yt-red-hover text-white font-medium rounded transition-colors duration-200"),
					)(A(AttrClass("text-white no-underline"), AttrHref("/login"))("Go to Login")),
				),
			),
		)),
	)
}

var invalidVerificationLinkPageCache NodeCache

func InvalidVerificationLinkPage() HyperNode {
	return Cache(&invalidVerificationLinkPageCache,
		page("Invalid Verification Link", Group(
			DIV(AttrClass("min-h-screen flex items-center justify-center bg-yt-bg"))(
				DIV(AttrClass("w-full max-w-md p-8 bg-yt-surface rounded-lg shadow-lg text-center"))(
					DIV(AttrClass("mb-4"))(RawText(`<svg class="w-16 h-16 mx-auto text-red-500" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>`)),
					H1(AttrClass("text-2xl font-bold text-yt-text mb-4"))("Invalid Link"),
					P(AttrClass("text-yt-text-secondary"))("This verification link is invalid or has expired."),
					// TODO: resend verification email
				),
			),
		)),
	)
}

var loginPageCache NodeCache

func LoginPage() HyperNode {
	return Cache(&loginPageCache,
		page("Login", Group(
			DIV(AttrClass("min-h-screen flex items-center justify-center bg-yt-bg"))(
				DIV(AttrClass("w-full max-w-md p-8 bg-yt-surface rounded-lg shadow-lg"))(
					H1(AttrClass("text-2xl font-bold text-center mb-6 text-yt-text"))("Login"),
					LoginForm(),
					P(AttrClass("mt-4 text-center text-yt-text-secondary text-sm"))(
						"Don't have an account? ",
						A(AttrClass("text-yt-red hover:text-yt-red-hover font-medium"), AttrHref("/register"))("Register"),
					),
				),
			),
		)),
	)
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
		AttrID("login-form"),
		AttrClass("space-y-4"),
		Attr("hx-post", "/login"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .login-button"),
		Attr("hx-disabled-elt", "find button"),
	)(
		DIV()(
			LABEL(AttrClass("block text-sm font-medium text-yt-text mb-1"), AttrFor("email"))("Email"),
			INPUT(AttrClass("w-full px-4 py-2 bg-yt-surface-hover border border-yt-border rounded text-yt-text placeholder-yt-text-secondary focus:outline-none focus:border-yt-red"), AttrType(TypeEmail), AttrID("email"), AttrName("email"), AttrValue(p.Email), AttrRequired(true)),
		),
		DIV()(
			LABEL(AttrClass("block text-sm font-medium text-yt-text mb-1"), AttrFor("password"))("Password"),
			INPUT(AttrClass("w-full px-4 py-2 bg-yt-surface-hover border border-yt-border rounded text-yt-text placeholder-yt-text-secondary focus:outline-none focus:border-yt-red"), AttrType(TypePassword), AttrID("password"), AttrName("password"), AttrValue(p.Password), AttrRequired(true)),
		),
		DIV(AttrClass("pt-2"))(
			BUTTON(
				AttrClass("btn group login-button w-full font-medium"),
				AttrType(TypeSubmit),
			)(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Login"),
				spinner(),
			),
		),

		If(p.Err == ErrInvalidCredentials,
			P(AttrClass("mt-2 text-sm text-red-500 text-center"))("Invalid email or password."),
		).ElseIf(p.Err == ErrEmailNotVerified,
			P(AttrClass("mt-2 text-sm text-red-500 text-center"))("Email not verified. Please check your inbox for verification email."),
			// TODO: resend verification email
		),
	)
}
