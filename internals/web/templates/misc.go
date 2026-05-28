package templates

import (
	"fmt"
	"time"

	. "github.com/assaidy/hyper/v2"
	"github.com/oklog/ulid/v2"
)

type basicLayoutParams struct {
	title string
}

func basicPageLayout(params basicLayoutParams) ChildrenInserter {
	clientId := ulid.Make()

	return func(children ...any) Element {
		return Group(
			DOCTYPE(),
			HTML(AttrLang("en"))(
				HEAD()(
					META(AttrCharset("UTF-8")),
					META(AttrName("viewport"), AttrContent("width=device-width, initial-scale=1.0")),
					TITLE()(params.title),
					LINK(AttrRel("preconnect"), AttrHref("https://fonts.googleapis.com")),
					LINK(AttrRel("preconnect"), AttrHref("https://fonts.gstatic.com"), AttrCrossOrigin(CrossOriginAnonymous)),
					LINK(AttrRel("stylesheet"), AttrHref("https://fonts.googleapis.com/css2?family=Roboto:wght@400;500;700&display=swap")),
					LINK(AttrRel("stylesheet"), AttrHref("/assets/css/style.css")),
					SCRIPT(AttrSrc("/assets/js/lib/htmx@4.0.0_beta2.js"))(),
					SCRIPT(AttrDefer(true))(RawText(fmt.Sprintf(`
					// TODO: Rename clientId To websocketId and only use it when initializing the ws connection.
					window._clientId = %q;

					htmx.on('htmx:config:request', (event) => {
						event.detail.ctx.request.headers['X-CSRF-Token'] = document.cookie
							.split("; ")
							.map((cookie) => cookie.split("="))
							.find(([key]) => key === 'csrf_token')
							?.map(decodeURIComponent)[1] || null;
						event.detail.ctx.request.headers['X-Client-ID']  = window._clientId;
					});
				`, clientId))),
				),
				BODY(
					AttrClass("min-h-screen bg-base-300"),
					Attr("hx-status:5xx:inherited", "swap:none"),
				)(
					DIV(AttrId("ALERT_TOAST"), AttrClass("toast toast-top w-md z-[1000000]"))(),
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

	t := 5 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}

	return DIV(AttrId("ALERT_TOAST"), Attr("hx-swap-oob", "prepend"))(
		DIV(
			AttrRole("alert"),
			AttrClass(fmt.Sprintf("alert alert-%s", level)),
			Attr("hx-on::after:process", fmt.Sprintf("setTimeout(() => this.remove(), %d)", t.Milliseconds())),
		)(
			RawText(icon), SPAN()(message),
		),
	)
}
