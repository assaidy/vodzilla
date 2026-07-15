package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleGetFeed(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "feeda@example.com", "Password123", "Feed A", "feeda")
	userB := createVerifiedUser(t, app, "feedb@example.com", "Password123", "Feed B", "feedb")

	createVerifiedVideo(t, userB.ID, "User B Video", "From user B")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/feed", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid cursor", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/feed?cursor=invalid", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidCursor")
	})

	t.Run("limit too low", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/feed?limit=14", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("limit too high", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/feed?limit=200", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("empty feed (not following anyone)", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/feed", nil, userA.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("feed with videos after following", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/feed", nil, userA.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 1)
		item := items[0].(map[string]any)
		require.Equal(t, "User B Video", item["title"])
	})
}
