package templates

import (
	"fmt"

	. "github.com/assaidy/hyper/v2"
	"github.com/oklog/ulid/v2"
)

func page(title string, root HyperNode) HyperNode {
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
				LINK(AttrRel("stylesheet"), AttrHref("/public/css/style.css")),
				SCRIPT(AttrType("module"), AttrSrc("/public/js/lib/datastar@v1.0.1.js"))(),
				SCRIPT(AttrDefer(true), AttrSrc("/public/js/script.js"))(),
			),
			BODY()(
				DIV(AttrID("alertToast"), AttrClass("toast toast-top toast-center w-md"))(),
				root,
			),
		),
	)
}

func pageWithSse(title string, root HyperNode) HyperNode {
	clientID := ulid.Make()
	return page(title, Group(
		DIV(
			AttrHidden(true),
			Attr("data-init", fmt.Sprintf("@get('/persistent_sse/%s')", clientID)),
			Attr("data-signals", fmt.Sprintf("{clientID: '%s'}", clientID)),
		)(),
		root,
	))
}

func spinner(signal string) HyperNode {
	return RawText(fmt.Sprintf(`<svg data-class:loading="%s" class="animate-spin ml-2 hidden [&.loading]:inline-block" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-loader-circle-icon lucide-loader-circle"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>`,
		signal,
	))
}

type AlertLevel string

const (
	AlertInfo    AlertLevel = "info"
	AlertSuccess AlertLevel = "success"
	AlertWarning AlertLevel = "warning"
	AlertError   AlertLevel = "error"
)

func Alert(level AlertLevel, message string) HyperNode {
	var icon string
	switch level {
	case AlertInfo:
		icon = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="h-6 w-6 shrink-0 stroke-current">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
  </svg>`
	case AlertSuccess:
		icon = `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>`
	case AlertWarning:
		icon = `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
  </svg>`
	case AlertError:
		icon = `<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>`
	}

	return DIV(AttrRole("alert"), AttrClass(fmt.Sprintf("alert alert-%s", level)))(
		RawText(icon), SPAN()(message),
	)
}
