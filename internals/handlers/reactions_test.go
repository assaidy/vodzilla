package handlers

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHandleViewVideo(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "vv@example.com", "Password123", "View Video", "vv")
	videoID := createVerifiedVideo(t, user.ID, "View Test", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid video_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/videos/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestHandleViewPlaylist(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "vp@example.com", "Password123", "View Playlist", "vp")

	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "View Test PL"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/playlists/"+playlistID, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/playlists/"+playlistID, nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid playlist_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/playlists/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/playlists/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/playlists/"+playlistID, nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestHandleCreateVideoComment(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "cvc@example.com", "Password123", "Create V Comment", "cvc")
	videoID := createVerifiedVideo(t, user.ID, "Comment Test", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Great video!"}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Great video!"}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid video_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/not-a-uuid", fiber.Map{"content": "Great video!"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/00000000-0000-0000-0000-000000000000", fiber.Map{"content": "Great video!"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("validation empty content", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": ""}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation whitespace only", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "   "}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation too long", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 501; i++ {
			longContent += "a"
		}
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": longContent}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Great video!"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		commentID, _ := data["commentId"].(string)
		require.NotEmpty(t, commentID)
		_, err := uuid.Parse(commentID)
		require.NoError(t, err)
	})
}

func TestHandleGetVideoComments(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "gvc@example.com", "Password123", "Get V Comments", "gvc")
	videoID := createVerifiedVideo(t, user.ID, "Get Comments Test", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid video_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/videos/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("invalid cursor", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/videos/"+videoID.String()+"?cursor=invalid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidCursor")
	})

	t.Run("limit too low", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/videos/"+videoID.String()+"?limit=14", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("limit too high", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/videos/"+videoID.String()+"?limit=200", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("empty list", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("success with comments", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "First comment"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		c1, _ := data["commentId"].(string)

		resp = testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Second comment"}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		c2, _ := data["commentId"].(string)

		resp = testRequest(t, app, http.MethodGet, "/reactions/comments/videos/"+videoID.String(), nil, user.Session)
		status, data = parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 2)
		ids := make(map[string]bool)
		for _, item := range items {
			m := item.(map[string]any)
			ids[m["id"].(string)] = true
		}
		require.True(t, ids[c1])
		require.True(t, ids[c2])
	})
}

func TestHandleCreateCommentReply(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "ccr@example.com", "Password123", "Create C Reply", "ccr")
	videoID := createVerifiedVideo(t, user.ID, "Reply Test", "")

	resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Parent comment"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	commentID, _ := data["commentId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": "A reply"}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": "A reply"}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid comment_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/not-a-uuid/replies", fiber.Map{"content": "A reply"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("non-existent comment", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/00000000-0000-0000-0000-000000000000/replies", fiber.Map{"content": "A reply"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("validation empty content", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": ""}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation whitespace only", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": "   "}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation too long", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 501; i++ {
			longContent += "a"
		}
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": longContent}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": "A reply"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		replyID, _ := data["replyId"].(string)
		require.NotEmpty(t, replyID)
		_, err := uuid.Parse(replyID)
		require.NoError(t, err)
	})
}

func TestHandleGetCommentReplies(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "gcr@example.com", "Password123", "Get C Replies", "gcr")
	videoID := createVerifiedVideo(t, user.ID, "Get Replies Test", "")

	resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Parent comment"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	commentID, _ := data["commentId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/"+commentID+"/replies", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid comment_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/not-a-uuid/replies", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("non-existent comment", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/00000000-0000-0000-0000-000000000000/replies", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("invalid cursor", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/"+commentID+"/replies?cursor=invalid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidCursor")
	})

	t.Run("limit too low", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/"+commentID+"/replies?limit=14", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("limit too high", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/"+commentID+"/replies?limit=200", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("empty list", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/comments/"+commentID+"/replies", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Empty(t, items)
	})

	t.Run("success with replies", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": "First reply"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		r1, _ := data["replyId"].(string)

		resp = testRequest(t, app, http.MethodPost, "/reactions/comments/"+commentID+"/replies", fiber.Map{"content": "Second reply"}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		r2, _ := data["replyId"].(string)

		resp = testRequest(t, app, http.MethodGet, "/reactions/comments/"+commentID+"/replies", nil, user.Session)
		status, data = parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		require.Len(t, items, 2)
		ids := make(map[string]bool)
		for _, item := range items {
			m := item.(map[string]any)
			ids[m["id"].(string)] = true
		}
		require.True(t, ids[r1])
		require.True(t, ids[r2])
	})
}

func TestHandleEditComment(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "ec@example.com", "Password123", "Edit Comment", "ec")
	videoID := createVerifiedVideo(t, user.ID, "Edit Comment Test", "")

	resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Original content"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	commentID, _ := data["commentId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/"+commentID, fiber.Map{"content": "Updated content"}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/"+commentID, fiber.Map{"content": "Updated content"}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid comment_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/not-a-uuid", fiber.Map{"content": "Updated content"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("non-existent comment", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/00000000-0000-0000-0000-000000000000", fiber.Map{"content": "Updated content"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("validation empty content", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/"+commentID, fiber.Map{"content": ""}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation whitespace only", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/"+commentID, fiber.Map{"content": "   "}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation too long", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 501; i++ {
			longContent += "a"
		}
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/"+commentID, fiber.Map{"content": longContent}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/reactions/comments/"+commentID, fiber.Map{"content": "Updated content"}, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestHandleDeleteComment(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "dc@example.com", "Password123", "Delete Comment", "dc")
	videoID := createVerifiedVideo(t, user.ID, "Delete Comment Test", "")

	resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "To delete"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	commentID, _ := data["commentId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/comments/"+commentID, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/reactions/comments/"+commentID, nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid comment_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/comments/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("non-existent comment", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/comments/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/comments/"+commentID, nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/comments/"+commentID, nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})
}

func TestHandleAddVideoFeeling(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "avf@example.com", "Password123", "Add V Feeling", "avf")
	videoID := createVerifiedVideo(t, user.ID, "Feeling Test", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoID.String(), fiber.Map{"kind": "like"}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoID.String(), fiber.Map{"kind": "like"}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid video_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/not-a-uuid", fiber.Map{"kind": "like"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/00000000-0000-0000-0000-000000000000", fiber.Map{"kind": "like"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("validation invalid kind", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoID.String(), fiber.Map{"kind": "invalid"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success with like", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoID.String(), fiber.Map{"kind": "like"}, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("switch to dislike", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoID.String(), fiber.Map{"kind": "dislike"}, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestHandleDeleteVideoFeeling(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "dvf@example.com", "Password123", "Delete V Feeling", "dvf")
	videoID := createVerifiedVideo(t, user.ID, "Del Feeling Test", "")

	resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/videos/"+videoID.String(), fiber.Map{"kind": "like"}, user.Session)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid video_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/videos/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("feeling not found on different video", func(t *testing.T) {
		otherVideo := createVerifiedVideo(t, user.ID, "Other Video", "")
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/videos/"+otherVideo.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "FeelingNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "FeelingNotFound")
	})
}

func TestHandleGetVideoViewsCount(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "gvvc@example.com", "Password123", "Get VV Count", "gvvc")
	videoID := createVerifiedVideo(t, user.ID, "Views Count Test", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/videos/"+videoID.String()+"/count", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid video_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/videos/not-a-uuid/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/videos/00000000-0000-0000-0000-000000000000/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("zero views", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/videos/"+videoID.String()+"/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(0), count)
	})

	t.Run("non-zero views", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/reactions/views/videos/"+videoID.String()+"/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(1), count)
	})
}

func TestHandleGetPlaylistViewsCount(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "gpvc@example.com", "Password123", "Get PV Count", "gpvc")

	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "Views Count PL"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/playlists/"+playlistID+"/count", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid playlist_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/playlists/not-a-uuid/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/playlists/00000000-0000-0000-0000-000000000000/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("zero views", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/reactions/views/playlists/"+playlistID+"/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(0), count)
	})

	t.Run("non-zero views", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/views/playlists/"+playlistID, nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/reactions/views/playlists/"+playlistID+"/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		count, _ := data["count"].(float64)
		require.Equal(t, float64(1), count)
	})
}

func TestHandleAddCommentFeeling(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "acf@example.com", "Password123", "Add C Feeling", "acf")
	videoID := createVerifiedVideo(t, user.ID, "C Feeling Test", "")

	resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Feeling target"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	commentID, _ := data["commentId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/"+commentID, fiber.Map{"kind": "like"}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/"+commentID, fiber.Map{"kind": "like"}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid comment_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/not-a-uuid", fiber.Map{"kind": "like"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("non-existent comment", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/00000000-0000-0000-0000-000000000000", fiber.Map{"kind": "like"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("validation invalid kind", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/"+commentID, fiber.Map{"kind": "invalid"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success with like", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/"+commentID, fiber.Map{"kind": "like"}, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("switch to dislike", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/"+commentID, fiber.Map{"kind": "dislike"}, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestHandleDeleteCommentFeeling(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "dcf@example.com", "Password123", "Delete C Feeling", "dcf")
	videoID := createVerifiedVideo(t, user.ID, "Del C Feeling Test", "")

	resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "C Feeling target"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	commentID, _ := data["commentId"].(string)

	resp = testRequest(t, app, http.MethodPost, "/reactions/feelings/comments/"+commentID, fiber.Map{"kind": "like"}, user.Session)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/comments/"+commentID, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/comments/"+commentID, nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid comment_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/comments/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("non-existent comment", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/comments/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "CommentNotFound")
	})

	t.Run("feeling not found on different comment", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/reactions/comments/videos/"+videoID.String(), fiber.Map{"content": "Another comment"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		otherCommentID, _ := data["commentId"].(string)

		resp = testRequest(t, app, http.MethodDelete, "/reactions/feelings/comments/"+otherCommentID, nil, user.Session)
		status2, data2 := parseResponse(t, resp)
		assertKind(t, status2, data2, 404, "FeelingNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/comments/"+commentID, nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/reactions/feelings/comments/"+commentID, nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "FeelingNotFound")
	})
}
