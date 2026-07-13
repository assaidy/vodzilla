package handlers

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHandleGetProfile(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	session := createVerifiedUser(t, app, "profown@example.com", "Password123", "Profile Own", "profown")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("own profile", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles", nil, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleGetProfileByUsername(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	session := createVerifiedUser(t, app, "profuser@example.com", "Password123", "Profile Username", "profuser")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/usernames/profuser", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("by username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/usernames/profuser", nil, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("non-existent username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/usernames/nonexistent_user", nil, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})
}

func TestHandleGetProfileById(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	session := createVerifiedUser(t, app, "profid@example.com", "Password123", "Profile ID", "profid")

	resp := testRequest(t, app, http.MethodGet, "/profiles", nil, session)
	status, data := parseResponse(t, resp)
	_ = status
	myId, _ := data["id"].(string)

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/id/not-a-uuid", nil, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent ID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/id/00000000-0000-0000-0000-000000000000", nil, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("by ID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles/id/"+myId, nil, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleEditProfile(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	session := createVerifiedUser(t, app, "profcrd@example.com", "Password123", "Profile Edit", "profcrd")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "x", "username": "y",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "x", "username": "y",
		}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("empty body", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{}, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("empty name", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "", "username": "newname", "bio": "",
		}, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("empty username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "New Name", "username": "", "bio": "",
		}, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("invalid username chars", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "New Name", "username": "bad user!", "bio": "",
		}, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("valid edit", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/profiles", fiber.Map{
			"name": "Updated Profile", "username": "profupdated", "bio": "This is my bio",
		}, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("verify edited profile", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/profiles", nil, session)
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
		}, session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "UsernameConflict")
	})
}

func TestHandleDeleteProfile(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	session := createVerifiedUser(t, app, "profdell@example.com", "Password123", "Profile Del", "profdell")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/profiles", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/profiles", nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("delete profile", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/profiles", nil, session)
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
	t.Skip("TODO: implement test for HandleEditProfileAvatar")
}

func TestHandleConfirmProfileAvatarUpload(t *testing.T) {
	t.Skip("TODO: implement test for HandleConfirmProfileAvatarUpload")
}

func TestHandleDeleteProfileAvatar(t *testing.T) {
	t.Skip("TODO: implement test for HandleDeleteProfileAvatar")
}
