package templates

import (
	. "github.com/assaidy/hyper/v2"
)

var registerPageCache NodeCache

func RegisterPage() HyperNode {
	return Cache(&registerPageCache,
		basicPageLayout(basicLayoutParams{title: "Register"})(
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
		),
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

	inputClass := "input w-full"
	erroredInputClass := "input input-error w-full"

	return FORM(
		AttrId("REGISTER_FORM"),
		AttrClass("space-y-4"),
		Attr("hx-post", "/register"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .register-button"),
		Attr("hx-disable", "find button"),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_NAME"))(
				SPAN(AttrClass("label-text"))("Name"),
			),
			INPUT(
				AttrId("FORM_NAME"),
				AttrClass(IfElse(p.NameErr == nil, inputClass, erroredInputClass)),
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
			LABEL(AttrClass("label"), AttrFor("FORM_USERNAME"))(
				SPAN(AttrClass("label-text"))("Username"),
			),
			INPUT(
				AttrId("FORM_USERNAME"),
				AttrClass(IfElse(p.UsernameErr == nil, inputClass, erroredInputClass)),
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
			LABEL(AttrClass("label"), AttrFor("FORM_EMAIL"))(
				SPAN(AttrClass("label-text"))("Email"),
			),
			INPUT(
				AttrId("FORM_EMAIL"),
				AttrClass(IfElse(p.EmailErr == nil, inputClass, erroredInputClass)),
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
			LABEL(AttrClass("label"), AttrFor("FORM_PASSWORD"))(
				SPAN(AttrClass("label-text"))("Password"),
			),
			INPUT(
				AttrId("FORM_PASSWORD"),
				AttrClass(IfElse(p.PasswordErr == nil, inputClass, erroredInputClass)),
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
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block"))(),
			),
		),
	)
}

var verificationEmailSentPageCache NodeCache

func VerificationEmailSentPage() HyperNode {
	return Cache(&verificationEmailSentPageCache,
		basicPageLayout(basicLayoutParams{title: "Verification Email Sent"})(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body text-center"))(
						DIV(AttrClass("text-warning mb-4"))(
							RawText(`<svg class="w-16 h-16 mx-auto" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-mail-icon lucide-mail"><path d="m22 7-8.991 5.727a2 2 0 0 1-2.009 0L2 7"/><rect x="2" y="4" width="20" height="16" rx="2"/></svg>`),
						),
						H1(AttrClass("text-2xl font-bold"))("Verification Email Sent"),
						P(AttrClass("text-base-content/70"))(
							"We have sent you a verification email. Please check your inbox.",
						),
					),
				),
			),
		),
	)
}

var emailVerifiedPageCache NodeCache

func EmailVerifiedPage() HyperNode {
	return Cache(&emailVerifiedPageCache,
		basicPageLayout(basicLayoutParams{title: "Email Verified"})(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body text-center"))(
						DIV(AttrClass("text-success mb-4"))(
							RawText(`<svg class="w-16 h-16 mx-auto" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>`),
						),
						H1(AttrClass("text-2xl font-bold"))("Email Verified"),
						P(AttrClass("text-base-content/70"))(
							"Your email has been verified successfully. You can now log in to your account.",
						),
						DIV(AttrClass("card-actions justify-center mt-4"))(
							A(AttrClass("btn btn-primary"), AttrHref("/login"))("Go to Login"),
						),
					),
				),
			),
		),
	)
}

var invalidVerificationLinkPageCache NodeCache

func InvalidVerificationLinkPage() HyperNode {
	return Cache(&invalidVerificationLinkPageCache,
		basicPageLayout(basicLayoutParams{title: "Invalid Verification Link"})(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body text-center"))(
						DIV(AttrClass("text-error mb-4"))(
							RawText(`<svg class="w-16 h-16 mx-auto" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>`),
						),
						H1(AttrClass("text-2xl font-bold"))("Invalid Link"),
						P(AttrClass("text-base-content/70"))(
							"This verification link is invalid or has expired.",
						),
					),
				),
			),
		),
	)
}

var loginPageCache NodeCache

func LoginPage() HyperNode {
	return Cache(&loginPageCache,
		basicPageLayout(basicLayoutParams{title: "Login"})(
			DIV(AttrClass("min-h-screen flex items-center justify-center"))(
				DIV(AttrClass("card w-full max-w-md bg-base-100 border-base-300 shadow-lg"))(
					DIV(AttrClass("card-body"))(
						H1(AttrClass("card-title text-3xl justify-center mb-4"))("Login"),
						LoginForm(),
						P(AttrClass("text-center text-sm text-base-content/70 mt-4"))(
							"Don't have an account? ",
							A(AttrClass("link link-primary"), AttrHref("/register"))("Register"),
						),
					),
				),
			),
		),
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

	inputClass := "input w-full"
	erroredInputClass := "input input-error w-full"

	return FORM(
		AttrId("LOGIN_FORM"),
		AttrClass("space-y-4"),
		Attr("hx-post", "/login"),
		Attr("hx-swap", "outerHTML"),
		Attr("hx-indicator", "find .login-button"),
		Attr("hx-disable", "find button"),
	)(
		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_EMAIL"))(
				SPAN(AttrClass("label-text"))("Email"),
			),
			INPUT(
				AttrId("FORM_EMAIL"),
				AttrClass(IfElse(p.Err != ErrInvalidCredentials, inputClass, erroredInputClass)),
				AttrType(TypeEmail),
				AttrName("email"),
				AttrValue(p.Email),
				AttrRequired(true),
			),
		),

		DIV(AttrClass("fieldset"))(
			LABEL(AttrClass("label"), AttrFor("FORM_PASSWORD"))(
				SPAN(AttrClass("label-text"))("Password"),
			),
			INPUT(
				AttrId("FORM_PASSWORD"),
				AttrClass(IfElse(p.Err != ErrInvalidCredentials, inputClass, erroredInputClass)),
				AttrType(TypePassword),
				AttrName("password"),
				AttrValue(p.Password),
				AttrRequired(true),
			),
		),

		DIV(AttrClass("pt-2"))(
			BUTTON(
				AttrClass("btn btn-primary w-full login-button group"),
				AttrType(TypeSubmit),
			)(
				SPAN(AttrClass("group-[.htmx-request]:hidden"))("Login"),
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block"))(),
			),
		),

		If(p.Err == ErrInvalidCredentials,
			DIV(AttrClass("text-center text-error mt-2"))(
				SPAN()("Invalid email or password."),
			),
		).ElseIf(p.Err == ErrEmailNotVerified,
			DIV(AttrClass("text-center text-error mt-2"))(
				SPAN()("Email not verified. Please check your inbox for verification email."),
			),
		),
	)
}
