package handlers

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestHandleGetNotifications(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "notifa@example.com", "Password123", "Notif A", "notifa")
	userB := createVerifiedUser(t, app, "notifb@example.com", "Password123", "Notif B", "notifb")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid cursor", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications?cursor=invalid", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidCursor")
	})

	t.Run("limit too low", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications?limit=14", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("limit too high", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications?limit=200", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("empty list", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications", nil, userA.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	videoA := createVerifiedVideo(t, userA.ID, "Notif Video A", "")
	videoB := createVerifiedVideo(t, userB.ID, "Notif Video B", "")

	t.Run("notifications for all kinds", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// userB comments on userA's video → video_comment for userA
		resp = testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoA.String(), fiber.Map{"content": "B's comment on A's video"}, userB.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		commentB := data["commentId"].(string)

		// userA comments on userB's video → video_comment for userB; this comment is owned by userA
		resp = testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoB.String(), fiber.Map{"content": "A's comment on B's video"}, userA.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		commentA := data["commentId"].(string)

		// userA feelings on videoB → video_feeling for userB
		resp = testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoB.String(), fiber.Map{"kind": "like"}, userA.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// userB replies to userA's comment → comment_reply for userA
		resp = testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentA+"/replies", fiber.Map{"content": "B's reply"}, userB.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		replyID := data["replyId"].(string)
		require.NotEmpty(t, replyID)

		// userA feels userB's comment → comment_feeling for userB
		resp = testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/"+commentB, fiber.Map{"kind": "dislike"}, userA.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/notifications/notifications?limit=100", nil, userA.Session)
		status, data = parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		userAKinds := make(map[string]bool)
		for _, item := range data["items"].([]any) {
			userAKinds[item.(map[string]any)["kind"].(string)] = true
		}
		require.True(t, userAKinds["video_comment"], "expected video_comment for userA")
		require.True(t, userAKinds["comment_reply"], "expected comment_reply for userA")

		resp = testRequest(t, app, http.MethodGet, "/notifications/notifications?limit=100", nil, userB.Session)
		status, data = parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		userBKinds := make(map[string]bool)
		for _, item := range data["items"].([]any) {
			userBKinds[item.(map[string]any)["kind"].(string)] = true
		}
		require.True(t, userBKinds["follow"], "expected follow for userB")
		require.True(t, userBKinds["video_comment"], "expected video_comment for userB")
		require.True(t, userBKinds["video_feeling"], "expected video_feeling for userB")
		require.True(t, userBKinds["comment_feeling"], "expected comment_feeling for userB")
	})
}

func TestHandleGetUnreadNotificationsCount(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "cnta@example.com", "Password123", "Count A", "cnta")
	userB := createVerifiedUser(t, app, "cntb@example.com", "Password123", "Count B", "cntb")
	videoID := createVerifiedVideo(t, userB.ID, "Count Video", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications/count", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("zero count", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications/count", nil, userA.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(0), count)
	})

	t.Run("count after follow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/notifications/notifications/count", nil, userB.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(1), count)
	})

	t.Run("count after feeling", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoID.String(), fiber.Map{"kind": "like"}, userA.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/notifications/notifications/count", nil, userB.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(2), count)
	})
}

func TestHandleMarkNotificationAsRead(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "reada@example.com", "Password123", "Read A", "reada")
	userB := createVerifiedUser(t, app, "readb@example.com", "Password123", "Read B", "readb")

	resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = testRequest(t, app, http.MethodGet, "/notifications/notifications", nil, userB.Session)
	status, data := parseResponse(t, resp)
	require.Equal(t, http.StatusOK, status)
	items, _ := data["items"].([]any)
	require.Len(t, items, 1)
	notificationID := items[0].(map[string]any)["id"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/notifications/"+notificationID+"/mark_read", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: userB.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/notifications/"+notificationID+"/mark_read", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid notification_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/notifications/not-a-uuid/mark_read", nil, userB.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "NotificationNotFound")
	})

	t.Run("non-existent notification", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/notifications/00000000-0000-0000-0000-000000000000/mark_read", nil, userB.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "NotificationNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/notifications/"+notificationID+"/mark_read", nil, userB.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("count is zero after marking read", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/notifications/notifications/count", nil, userB.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(0), count)
	})
}
