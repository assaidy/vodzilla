package handlers

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/starfederation/datastar-go/datastar"
)

func WithSseHelperData(c fiber.Ctx) error {
	c.SetContext(context.WithValue(c.Context(), "fiberApp", c.App()))
	c.SetContext(context.WithValue(c.Context(), "clientID", fiber.Params[string](c, "clientID")))
	return c.Next()
}

var HandleDatastarPersistentSse = adaptor.HTTPHandlerWithContext(
	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := adaptor.LocalContextFromHTTPRequest(r)
		if !ok {
			panic("fiber context not found")
		}

		_ = ctx.Value("fiberApp").(*fiber.App)
		_ = ctx.Value("clientID").(string)
		_ = datastar.NewSSE(w, r)

		// TODO: because we are working in a distributed env,
		// subscribe to sse topic, and send events if clientID is identical.
		// if failed to send close the connection and return. also close the subscription.
	}),
)
