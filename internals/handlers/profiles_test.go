package handlers

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestHandleGetProfile(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	user := createVerifiedUser(t, app, "profown@example.com", "Password123", "Profile Own", "profown")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("own profile", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleGetProfileByUsername(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	user := createVerifiedUser(t, app, "profuser@example.com", "Password123", "Profile Username", "profuser")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/usernames/profuser", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("by username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/usernames/profuser", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("non-existent username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/usernames/nonexistent_user", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})
}

func TestHandleGetProfileById(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	user := createVerifiedUser(t, app, "profid@example.com", "Password123", "Profile ID", "profid")

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/id/not-a-uuid", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent ID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/id/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("by ID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/id/"+user.ID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleEditProfile(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	user := createVerifiedUser(t, app, "profcrd@example.com", "Password123", "Profile Edit", "profcrd")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "x", "username": "y",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "x", "username": "y",
		}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("empty body", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("empty name", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "", "username": "newname", "bio": "",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("empty username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "New Name", "username": "", "bio": "",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("invalid username chars", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "New Name", "username": "bad user!", "bio": "",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("valid edit", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "Updated Profile", "username": "profupdated", "bio": "This is my bio",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("verify edited profile", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles", nil, user.Session)
		status, data := parseResponse(t, resp)
		_ = status
		name, _ := data["name"].(string)
		if name != "Updated Profile" {
			t.Errorf("name: want 'Updated Profile', got '%s'", name)
		}
	})

	createVerifiedUser(t, app, "second@example.com", "Password123", "Second User", "seconduser")

	t.Run("duplicate username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "Updated Profile", "username": "seconduser", "bio": "",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "UsernameConflict")
	})
}

func TestHandleDeleteProfile(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	user := createVerifiedUser(t, app, "profdell@example.com", "Password123", "Profile Del", "profdell")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/profiles", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/profiles", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("delete profile", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/profiles", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("login after delete", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "profdell@example.com",
			"password": "Password123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})
}

func TestHandleEditProfileAvatar(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "avataredit@example.com", "Password123", "Avatar Edit", "avataredit")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"contentType": "image/png",
			"fileSize":    100,
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"contentType": "image/png",
			"fileSize":    100,
		}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("validation errors", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"fileSize": 100,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"contentType": "video/mp4",
			"fileSize":    100,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")

		resp = testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"contentType": "image/png",
			"fileSize":    2*1024*1024 + 1,
		}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"contentType": "image/png",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		uploadURL, _ := data["uploadUrl"].(string)
		require.NotEmpty(t, uploadURL)
	})
}

func TestHandleConfirmProfileAvatarUpload(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "avatarconf@example.com", "Password123", "Avatar Confirm", "avatarconf")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar/confirm_upload", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar/confirm_upload", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no pending upload", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar/confirm_upload", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "AvatarNotFound")
	})

	t.Run("full flow and verify in profile", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"contentType": "image/png",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		uploadURL, _ := data["uploadUrl"].(string)
		require.NotEmpty(t, uploadURL)

		uploadToPresignedURL(t, uploadURL, "image/png", make([]byte, 100))

		resp = testRequest(t, app, http.MethodPut, "/profiles/avatar/confirm_upload", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		avatarURL, _ := data["avatarUrl"].(string)
		require.NotEmpty(t, avatarURL)

		resp = testRequest(t, app, http.MethodGet, "/profiles", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		profileAvatarURL, _ := data["avatarUrl"].(string)
		if profileAvatarURL == "" {
			t.Error("expected non-empty avatarUrl in profile response")
		}
	})
}

func TestHandleDeleteProfileAvatar(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "avatardel@example.com", "Password123", "Avatar Del", "avatardel")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/profiles/avatar", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/profiles/avatar", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no avatar", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/profiles/avatar", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "AvatarNotFound")
	})

	t.Run("full flow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles/avatar", fiber.Map{
			"contentType": "image/png",
			"fileSize":    100,
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		uploadURL, _ := data["uploadUrl"].(string)

		uploadToPresignedURL(t, uploadURL, "image/png", make([]byte, 100))

		resp = testRequest(t, app, http.MethodPut, "/profiles/avatar/confirm_upload", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")

		resp = testRequest(t, app, http.MethodDelete, "/profiles/avatar", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")

		resp = testRequest(t, app, http.MethodGet, "/profiles", nil, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		avatarURL, _ := data["avatarUrl"].(string)
		if avatarURL != "" {
			t.Errorf("expected empty avatarUrl after deletion, got %q", avatarURL)
		}
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/profiles/avatar", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "AvatarNotFound")
	})
}
