package handlers

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHandleRegister(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	t.Run("empty body", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"password": "Password123",
			"name":     "Test",
			"username": "test",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing password", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "test@example.com",
			"name":     "Test",
			"username": "test",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing name", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "test@example.com",
			"password": "Password123",
			"username": "test",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "test@example.com",
			"password": "Password123",
			"name":     "Test",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("invalid email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "bad",
			"password": "Password123",
			"name":     "Test",
			"username": "test",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("short password", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "test@example.com",
			"password": "short",
			"name":     "Test",
			"username": "test",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("empty name", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "test@example.com",
			"password": "Password123",
			"name":     "",
			"username": "test",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("invalid username chars", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "test@example.com",
			"password": "Password123",
			"name":     "Test",
			"username": "bad user!",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("valid registration", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "auth@example.com",
			"password": "Password123",
			"name":     "Auth Tester",
			"username": "authtester",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("duplicate email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "auth@example.com",
			"password": "Password123",
			"name":     "Auth Tester",
			"username": "other",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "EmailConflict")
	})

	t.Run("duplicate username", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
			"email":    "other@example.com",
			"password": "Password123",
			"name":     "Other",
			"username": "authtester",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "UsernameConflict")
	})
}

func TestHandleSendVerificationEmail(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "verify@example.com",
		"password": "Password123",
		"name":     "Verify Tester",
		"username": "verifytester",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	t.Run("empty body", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
			"baseUrl": "http://localhost/auth/verification_email/verify",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing baseUrl", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
			"email": "verify@example.com",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("invalid email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
			"email":   "bad",
			"baseUrl": "http://localhost/auth/verification_email/verify",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("invalid baseUrl", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
			"email":   "verify@example.com",
			"baseUrl": "not-a-url",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("non-existent email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
			"email":   "nonexistent@example.com",
			"baseUrl": "http://localhost/auth/verification_email/verify",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("valid request", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
			"email":   "verify@example.com",
			"baseUrl": "http://localhost/auth/verification_email/verify",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleVerifyEmail(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "verifytoken@example.com",
		"password": "Password123",
		"name":     "Verify Token",
		"username": "verifytoken",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
		"email":   "verifytoken@example.com",
		"baseUrl": "http://localhost/auth/verification_email/verify",
	}, nil)

	token := extractVerificationToken()

	t.Run("missing token query", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/auth/verification_email/verify", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "TokenNotFound")
	})

	t.Run("invalid token", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/auth/verification_email/verify?token=bad-token-123", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "TokenNotFound")
	})

	t.Run("valid token", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/auth/verification_email/verify?token="+token, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleLogin(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "login@example.com",
		"password": "Password123",
		"name":     "Login Tester",
		"username": "logintester",
	}, nil)

	createVerifiedUser(t, app, "login_verified@example.com", "Password123", "Login Verified", "loginverified")

	t.Run("wrong email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "wrong@example.com",
			"password": "Password123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("wrong password", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "login@example.com",
			"password": "WrongPassword123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("unverified cannot login", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "login@example.com",
			"password": "Password123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 403, "EmailNotVerified")
	})

	t.Run("verified user can login", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "login_verified@example.com",
			"password": "Password123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("verified user login with wrong password", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "login_verified@example.com",
			"password": "WrongPassword123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("verified user login with wrong email", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "wrong@example.com",
			"password": "Password123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})
}

func TestHandleLogout(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	user := createVerifiedUser(t, app, "logout@example.com", "Password123", "Logout Tester", "logouttester")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/logout", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("valid logout", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/logout", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("logout again (dead session)", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/logout", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})
}

func TestHandleEditPassword(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	user := createVerifiedUser(t, app, "editpass@example.com", "Password123", "Edit Pass", "editpass")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/auth/password", fiber.Map{}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/auth/password", fiber.Map{}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("empty body", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/auth/password", fiber.Map{}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing currentPassword", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/auth/password", fiber.Map{
			"newPassword": "NewPassword456",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("missing newPassword", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/auth/password", fiber.Map{
			"currentPassword": "Password123",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("wrong currentPassword", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/auth/password", fiber.Map{
			"currentPassword": "WrongPassword123",
			"newPassword":     "NewPassword456",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "InvalidPassword")
	})

	t.Run("valid edit", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/auth/password", fiber.Map{
			"currentPassword": "Password123",
			"newPassword":     "NewPassword456",
		}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("login with new password", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "editpass@example.com",
			"password": "NewPassword456",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("login with old password rejected", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
			"email":    "editpass@example.com",
			"password": "Password123",
		}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})
}
