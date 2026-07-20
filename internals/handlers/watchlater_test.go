package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleGetWatchlaters(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "wlget@example.com", "Password123", "WL Get", "wlget")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/watchlaters", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("validation limit too low", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/watchlaters?limit=1", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("validation limit too high", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/watchlaters?limit=200", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("empty list", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/watchlaters", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("success with one video", func(t *testing.T) {
		videoID := createVerifiedVideo(t, user.ID, "WL Video 1", "")
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/watchlaters", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 1)
		item, _ := items[0].(map[string]any)
		require.Equal(t, videoID.String(), item["id"])
	})

	t.Run("respects limit", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			vid := createVerifiedVideo(t, user.ID, "WL Limit Video", "")
			resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/"+vid.String(), nil, user.Session)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}
		resp := testRequest(t, app, http.MethodGet, "/watchlaters?limit=15", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 15)
	})
}

func TestHandleAddToWatchLaters(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "wladd@example.com", "Password123", "WL Add", "wladd")
	videoID := createVerifiedVideo(t, user.ID, "Add to WL", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("invalid video id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("already in watchlater", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "WatchlaterConflict")
	})
}

func TestHandleDeleteFromWatchLaters(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "wldel@example.com", "Password123", "WL Del", "wldel")
	videoID := createVerifiedVideo(t, user.ID, "Delete from WL", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/watchlaters/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/watchlaters/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/watchlaters/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("invalid video id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/watchlaters/videos/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("video not in watchlater", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/watchlaters/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "WatchlaterVideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/watchlaters/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodDelete, "/watchlaters/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/watchlaters", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/watchlaters/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "WatchlaterVideoNotFound")
	})
}
