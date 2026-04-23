package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/starfederation/datastar-go/datastar"
)

func WithSseHelperData(c fiber.Ctx) error {
	c.SetContext(context.WithValue(c.Context(), "fiber_app", c.App()))
	c.SetContext(context.WithValue(c.Context(), "client_id", fiber.Params[string](c, "client_id")))
	return c.Next()
}

var HandleDatastarSse = adaptor.HTTPHandlerWithContext(
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := adaptor.LocalContextFromHTTPRequest(r)
		if !ok {
			panic("fiber context not found")
		}

		app := ctx.Value("fiber_app").(*fiber.App)
		clientID := ctx.Value("client_id").(string)
		setSseState(app, clientID, datastar.NewSSE(w, r))
	}),
)

func setSseState(app *fiber.App, clientID string, sse *datastar.ServerSentEventGenerator) {
	app.State().Set(fmt.Sprintf("sse_%s", clientID), sse)
}

func getSseState(app *fiber.App, clientID string) *datastar.ServerSentEventGenerator {
	return fiber.MustGetState[*datastar.ServerSentEventGenerator](app.State(), fmt.Sprintf("sse_%s", clientID))
}
