package templates

import (
	. "github.com/assaidy/hyper/v2"
	"github.com/assaidy/lucide"
)

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
			LABEL(AttrClass("label"), AttrFor("FORM_USERNAME"))(
				SPAN(AttrClass("label-text"))("Username"),
			),
			INPUT(
				AttrId("FORM_USERNAME"),
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
			LABEL(AttrClass("label"), AttrFor("FORM_EMAIL"))(
				SPAN(AttrClass("label-text"))("Email"),
			),
			INPUT(
				AttrId("FORM_EMAIL"),
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
			LABEL(AttrClass("label"), AttrFor("FORM_PASSWORD"))(
				SPAN(AttrClass("label-text"))("Password"),
			),
			INPUT(
				AttrId("FORM_PASSWORD"),
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
				SPAN(AttrClass("loading loading-spinner loading-sm htmx-indicator hidden group-[.htmx-request]:inline-block"))(),
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
							RawText(lucide.Mail(lucide.Params{
								Class: "w-16 h-16 mx-auto",
							})),
						),
						H1(AttrClass("text-2xl font-bold"))("Verification Email Sent"),
						P(AttrClass("text-base-content/70"))(
							"We have sent you a verification email. Please check your inbox.",
						),
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
							RawText(lucide.MailCheck(lucide.Params{
								Class: "w-16 h-16 mx-auto",
							})),
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
							RawText(lucide.MailWarning(lucide.Params{
								Class: "w-16 h-16 mx-auto",
							})),
						),
						H1(AttrClass("text-2xl font-bold"))("Invalid Link"),
						P(AttrClass("text-base-content/70"))(
							"This verification link is invalid or has expired.",
						),
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
							"Don't have an account? ",
							A(AttrClass("link link-primary"), AttrHref("/register"))("Register"),
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
				AttrClass(Classes("input w-full", IfElseZero(p.Err == ErrInvalidCredentials, "input-error"))),
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
				AttrClass(Classes("input w-full", IfElseZero(p.Err == ErrInvalidCredentials, "input-error"))),
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
