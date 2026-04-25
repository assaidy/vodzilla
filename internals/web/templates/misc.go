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
			BODY()(root),
		),
	)
}

func pageWithSse(title string, root HyperNode) HyperNode {
	clientID := ulid.Make()
	return page(title, Group(
		DIV(
			AttrHidden(true),
			Attr("data-init", fmt.Sprintf("@get('/data_sse/%s')", clientID)),
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

// TODO: create general Toast(kind ToastKind, message string) component. knid is (Note, Warning, Error).
