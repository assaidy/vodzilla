package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleGetWatchHistory(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "histget@example.com", "Password123", "Hist Get", "histget")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/history", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("validation limit too low", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/history?limit=1", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("validation limit too high", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/history?limit=200", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("empty list", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("success with entries", func(t *testing.T) {
		video1 := createVerifiedVideo(t, user.ID, "Hist Video 1", "first")
		video2 := createVerifiedVideo(t, user.ID, "Hist Video 2", "second")

		resp := testRequest(t, app, http.MethodPost, "/history/videos/"+video1.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodPost, "/history/videos/"+video2.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 2)

		// newest first
		first, _ := items[0].(map[string]any)
		require.Equal(t, video2.String(), first["videoId"])
		require.Equal(t, "Hist Video 2", first["title"])
		require.NotEmpty(t, first["entryId"])
		require.NotEmpty(t, first["watchedAt"])

		second, _ := items[1].(map[string]any)
		require.Equal(t, video1.String(), second["videoId"])
	})

	t.Run("re-adding same video creates multiple entries", func(t *testing.T) {
		video := createVerifiedVideo(t, user.ID, "Hist Re-add", "")

		resp := testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		_, data := parseResponse(t, resp)
		before, _ := data["items"].([]any)

		resp = testRequest(t, app, http.MethodPost, "/history/videos/"+video.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodPost, "/history/videos/"+video.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, len(before)+2)

		// the two newest entries both reference the same video but are distinct
		first, _ := items[0].(map[string]any)
		second, _ := items[1].(map[string]any)
		require.Equal(t, video.String(), first["videoId"])
		require.Equal(t, video.String(), second["videoId"])
		require.NotEqual(t, first["entryId"], second["entryId"])
	})

	t.Run("respects limit", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			vid := createVerifiedVideo(t, user.ID, "Hist Limit Video", "")
			resp := testRequest(t, app, http.MethodPost, "/history/videos/"+vid.String(), nil, user.Session)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		resp := testRequest(t, app, http.MethodGet, "/history?limit=15", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 15)
	})
}

func TestHandleAddToWatchHistory(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "histadd@example.com", "Password123", "Hist Add", "histadd")
	videoID := createVerifiedVideo(t, user.ID, "Add to History", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/history/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/history/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/history/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("invalid video id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/history/videos/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/history/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 1)
	})
}

func TestHandleDeleteWatchHistoryEntry(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "histdel@example.com", "Password123", "Hist Del", "histdel")
	videoID := createVerifiedVideo(t, user.ID, "Delete from History", "")

	addAndGetEntryId := func() string {
		resp := testRequest(t, app, http.MethodPost, "/history/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		_, data := parseResponse(t, resp)
		items, _ := data["items"].([]any)
		entry, _ := items[0].(map[string]any)
		return fmt.Sprintf("%v", entry["entryId"])
	}

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/history/1", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/history/1", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid entry id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/history/not-a-number", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "WatchHistoryEntryNotFound")
	})

	t.Run("non-existent entry", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/history/999999", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "WatchHistoryEntryNotFound")
	})

	t.Run("success", func(t *testing.T) {
		entryId := addAndGetEntryId()

		resp := testRequest(t, app, http.MethodDelete, "/history/"+entryId, nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("delete again", func(t *testing.T) {
		entryId := addAndGetEntryId()

		resp := testRequest(t, app, http.MethodDelete, "/history/"+entryId, nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodDelete, "/history/"+entryId, nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "WatchHistoryEntryNotFound")
	})
}

func TestHandleClearWatchHistory(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "histclear@example.com", "Password123", "Hist Clear", "histclear")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/history", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/history", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("success", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			vid := createVerifiedVideo(t, user.ID, "Hist Clear Video", "")
			resp := testRequest(t, app, http.MethodPost, "/history/videos/"+vid.String(), nil, user.Session)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		resp := testRequest(t, app, http.MethodDelete, "/history", nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/history", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("clear when empty still succeeds", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/history", nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}
