package handlers

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestVideoUploadFlow(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "vidupload@example.com", "Password123", "Video Upload", "vidupload")

	t.Run("generate no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/videos/upload", fiber.Map{
			"contentType": "video/mp4",
			"fileSize":    1,
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("generate no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/videos/upload", fiber.Map{
			"contentType": "video/mp4",
			"fileSize":    1,
		}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("generate validation errors", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/videos/upload", fiber.Map{}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPost, "/videos/upload", fiber.Map{
			"fileSize": 1,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPost, "/videos/upload", fiber.Map{
			"contentType": "image/png",
			"fileSize":    1,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPost, "/videos/upload", fiber.Map{
			"contentType": "video/mp4",
			"fileSize":    32*1024*1024*1024 + 1,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	var uploadID string
	var objectKey string
	var chunkURL string

	t.Run("generate success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/videos/upload", fiber.Map{
			"contentType": "video/mp4",
			"fileSize":    1,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")

		uploadID, _ = data["uploadId"].(string)
		objectKey, _ = data["objectKey"].(string)
		require.NotEmpty(t, uploadID)
		require.NotEmpty(t, objectKey)

		chunks, _ := data["chunks"].([]any)
		require.Len(t, chunks, 1)
		chunk, _ := chunks[0].(map[string]any)
		chunkURL, _ = chunk["Url"].(string)
		require.NotEmpty(t, chunkURL)
	})

	t.Run("confirm no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"objectKey": objectKey,
			"uploadId":  uploadID,
			"parts":     []fiber.Map{{"etag": "x", "partNumber": 1}},
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("confirm no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"objectKey": objectKey,
			"uploadId":  uploadID,
			"parts":     []fiber.Map{{"etag": "x", "partNumber": 1}},
		}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("confirm validation errors", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"uploadId": uploadID,
			"parts":    []fiber.Map{{"etag": "x", "partNumber": 1}},
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"objectKey": objectKey,
			"parts":     []fiber.Map{{"etag": "x", "partNumber": 1}},
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"objectKey": objectKey,
			"uploadId":  uploadID,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("confirm wrong objectKey", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"objectKey": uuid.Must(uuid.NewV7()).String(),
			"uploadId":  "some-upload-id",
			"parts":     []fiber.Map{{"etag": "x", "partNumber": 1}},
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "NoPendingVideoUpload")
	})

	t.Run("confirm wrong uploadId", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"objectKey": objectKey,
			"uploadId":  "wrong-upload-id",
			"parts":     []fiber.Map{{"etag": "x", "partNumber": 1}},
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 422, "InvalidConfirmVideoUploadData")
	})

	var etag string

	t.Run("upload data to presigned URL", func(t *testing.T) {
		body := make([]byte, 5*1024*1024)
		req, err := http.NewRequest(http.MethodPut, chunkURL, bytes.NewReader(body))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		etag = resp.Header.Get("ETag")
		etag = strings.Trim(etag, "\"")
		require.NotEmpty(t, etag)
	})

	t.Run("confirm success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/upload/confirm", fiber.Map{
			"objectKey": objectKey,
			"uploadId":  uploadID,
			"parts":     []fiber.Map{{"etag": etag, "partNumber": 1}},
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("post no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/videos", fiber.Map{
			"title":       "Test Video",
			"description": "Test description",
			"objectKey":   objectKey,
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("post no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/videos", fiber.Map{
			"title":       "Test Video",
			"description": "Test description",
			"objectKey":   objectKey,
		}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("post validation errors", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/videos", fiber.Map{}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPost, "/videos", fiber.Map{
			"description": "Test description",
			"objectKey":   objectKey,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPost, "/videos", fiber.Map{
			"title":       "Test Video",
			"description": "Test description",
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("post wrong objectKey", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/videos", fiber.Map{
			"title":       "Test Video",
			"description": "Test description",
			"objectKey":   uuid.Must(uuid.NewV7()).String(),
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "ObjectNotFound")
	})

	var videoID uuid.UUID

	t.Run("post and verify video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/videos", fiber.Map{
			"title":       "My Test Video",
			"description": "Best video ever",
			"objectKey":   objectKey,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")

		videoIDStr, _ := data["videoId"].(string)
		var err error
		videoID, err = uuid.Parse(videoIDStr)
		require.NoError(t, err)

		resp = testRequest(t, app, http.MethodGet, "/videos/"+videoID.String(), nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		title, _ := data["title"].(string)
		if title != "My Test Video" {
			t.Errorf("title: want 'My Test Video', got '%s'", title)
		}
		ownerID, _ := data["userId"].(string)
		if ownerID != user.ID.String() {
			t.Errorf("ownerId: want '%s', got '%s'", user.ID.String(), ownerID)
		}
	})
}

func TestHandleGetVideo(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "getvideo@example.com", "Password123", "Get Video", "getvideo")
	videoID := createVerifiedVideo(t, user.ID, "Test Video", "A test video")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		title, _ := data["title"].(string)
		if title != "Test Video" {
			t.Errorf("title: want 'Test Video', got '%s'", title)
		}
	})
}

func TestHandleGetVideoStreamUrl(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "stream@example.com", "Password123", "Stream", "stream")
	videoID := createVerifiedVideo(t, user.ID, "Stream Video", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/stream_url", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/00000000-0000-0000-0000-000000000000/stream_url", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/stream_url", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		url, _ := data["url"].(string)
		if url == "" {
			t.Error("expected non-empty stream URL")
		}
	})
}

func TestHandleDeleteVideo(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "delvid@example.com", "Password123", "Del Video", "delvid")
	otherUser := createVerifiedUser(t, app, "other@example.com", "Password123", "Other", "otheruser")
	videoID := createVerifiedVideo(t, user.ID, "To Delete", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("not owner", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String(), nil, otherUser.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleGetVideosForUser(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "vidsuser@example.com", "Password123", "Vids User", "vidsuser")
	createVerifiedVideo(t, user.ID, "Video 1", "First video")
	createVerifiedVideo(t, user.ID, "Video 2", "Second video")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/"+user.ID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("limit too small", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/"+user.ID.String()+"?limit=5", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("limit too large", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/"+user.ID.String()+"?limit=200", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/"+user.ID.String()+"?limit=15", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		if len(items) != 2 {
			t.Errorf("expected 2 videos, got %d", len(items))
		}
	})

	t.Run("empty user (no videos)", func(t *testing.T) {
		other := createVerifiedUser(t, app, "empty@example.com", "Password123", "Empty", "emptyuser")
		resp := testRequest(t, app, http.MethodGet, "/videos/users/"+other.ID.String()+"?limit=15", nil, user.Session)
		status, data := parseResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		items, _ := data["items"].([]any)
		if len(items) != 0 {
			t.Errorf("expected 0 videos, got %d", len(items))
		}
	})
}

func TestHandleGetVideosCountForUser(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "vidcnt@example.com", "Password123", "Vid Count", "vidcnt")
	createVerifiedVideo(t, user.ID, "Only Video", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/"+user.ID.String()+"/count", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/00000000-0000-0000-0000-000000000000/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/users/"+user.ID.String()+"/count", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		count, _ := data["count"].(float64)
		if count != 1 {
			t.Errorf("count: want 1, got %v", count)
		}
	})
}

func TestHandleEditVideoThumbnail(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "thmbedit@example.com", "Password123", "Thumb Edit", "thmbedit")
	otherUser := createVerifiedUser(t, app, "thmbother@example.com", "Password123", "Thumb Other", "thmbother")
	videoID := createVerifiedVideo(t, user.ID, "Thumbnail Test", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("validation errors", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"fileSize": 100,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "video/mp4",
			"fileSize":    100,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    5*1024*1024 + 1,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/00000000-0000-0000-0000-000000000000/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("not owner", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, otherUser.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		uploadURL, _ := data["uploadUrl"].(string)
		require.NotEmpty(t, uploadURL)
	})
}

func TestHandleConfirmVideoThumbnailUpload(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "thmbconf@example.com", "Password123", "Thumb Confirm", "thmbconf")
	otherUser := createVerifiedUser(t, app, "thmbconfother@example.com", "Password123", "Thumb Conf Other", "thmbconfother")
	videoID := createVerifiedVideo(t, user.ID, "Confirm Thumbnail", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail/confirm_upload", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail/confirm_upload", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/00000000-0000-0000-0000-000000000000/thumbnail/confirm_upload", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("not owner", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail/confirm_upload", nil, otherUser.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("no pending upload", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail/confirm_upload", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "ThumbnailNotFound")
	})

	t.Run("full flow and verify in video response", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		uploadURL, _ := data["uploadUrl"].(string)
		require.NotEmpty(t, uploadURL)

		uploadToPresignedURL(t, uploadURL, "image/jpeg", make([]byte, 100))

		resp = testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail/confirm_upload", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		thumbnailURL, _ := data["thumbnailUrl"].(string)
		require.NotEmpty(t, thumbnailURL)

		resp = testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		respThumbnailURL, _ := data["thumbnailUrl"].(string)
		if respThumbnailURL == "" {
			t.Error("expected non-empty thumbnailUrl from thumbnail endpoint")
		}
	})
}

func TestHandleDeleteVideoThumbnail(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "thmbdel@example.com", "Password123", "Thumb Del", "thmbdel")
	otherUser := createVerifiedUser(t, app, "thmbdelother@example.com", "Password123", "Thumb Del Other", "thmbdelother")
	videoID := createVerifiedVideo(t, user.ID, "Delete Thumbnail", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String()+"/thumbnail", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String()+"/thumbnail", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/00000000-0000-0000-0000-000000000000/thumbnail", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("not owner", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String()+"/thumbnail", nil, otherUser.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("no thumbnail", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "ThumbnailNotFound")
	})

	t.Run("full flow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		uploadURL, _ := data["uploadUrl"].(string)

		uploadToPresignedURL(t, uploadURL, "image/jpeg", make([]byte, 100))

		resp = testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail/confirm_upload", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")

		resp = testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		respThumbnailURL, _ := data["thumbnailUrl"].(string)
		if respThumbnailURL == "" {
			t.Error("expected non-empty thumbnailUrl before deletion")
		}

		resp = testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")

		resp = testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 404, "ThumbnailNotFound")
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "ThumbnailNotFound")
	})
}

func TestHandleGetVideoThumbnailUrl(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "thumbget@example.com", "Password123", "Thumb Get", "thumbget")
	videoID := createVerifiedVideo(t, user.ID, "Get Thumbnail", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/thumbnail", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/00000000-0000-0000-0000-000000000000/thumbnail", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("no thumbnail", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "ThumbnailNotFound")
	})

	t.Run("thumbnail exists", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail", fiber.Map{
			"contentType": "image/jpeg",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		uploadURL, _ := data["uploadUrl"].(string)
		require.NotEmpty(t, uploadURL)

		uploadToPresignedURL(t, uploadURL, "image/jpeg", make([]byte, 100))

		resp = testRequest(t, app, http.MethodPut, "/videos/"+videoID.String()+"/thumbnail/confirm_upload", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		require.NotEmpty(t, data["thumbnailUrl"])

		resp = testRequest(t, app, http.MethodGet, "/videos/"+videoID.String()+"/thumbnail", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		thumbnailURL, _ := data["thumbnailUrl"].(string)
		if thumbnailURL == "" {
			t.Error("expected non-empty thumbnailUrl")
		}
	})
}
