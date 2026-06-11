package templates

import (
	"fmt"
	"time"

	. "github.com/assaidy/hyper/v2"
	"github.com/assaidy/lucide"
	"github.com/google/uuid"
)

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
					SCRIPT(AttrSrc("/assets/js/lib/htmx@4.0.0_beta2.js"))(),
					SCRIPT(AttrDefer(true))(RawText(fmt.Sprintf(`
						window._clientId = %q;

						htmx.on('htmx:config:request', (event) => {
							event.detail.ctx.request.headers['X-CSRF-Token'] = document.cookie
								.split("; ")
								.map((cookie) => cookie.split("="))
								.find(([key]) => key === 'csrf_token')
								?.map(decodeURIComponent)[1] || null;
							event.detail.ctx.request.headers['X-Client-ID']  = window._clientId;
						});
					`,
						clientId,
					))),
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
