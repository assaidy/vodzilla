package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/assaidy/vodzilla/internals/utils/mailer"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

type mockMailer struct {
	mu       sync.Mutex
	messages []mailer.Message
}

func newMockMailer() *mockMailer {
	return &mockMailer{}
}

func (m *mockMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockMailer) LastMessage() (mailer.Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return mailer.Message{}, false
	}
	return m.messages[len(m.messages)-1], true
}

func (m *mockMailer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
}

type testSession struct {
	Cookies   []*http.Cookie
	CsrfToken string
}

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()

	app := fiber.New(fiber.Config{})
	testHandler.RegisterRoutes(app)
	return app
}

func testRequest(t *testing.T, app *fiber.App, method, path string, body any, session *testSession) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, path, bodyReader)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if session != nil {
		for _, c := range session.Cookies {
			req.AddCookie(c)
		}
		if session.CsrfToken != "" {
			req.Header.Set("X-CSRF-Token", session.CsrfToken)
		}
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

func parseResponse(t *testing.T, resp *http.Response) (int, fiber.Map) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	if len(body) == 0 {
		return resp.StatusCode, nil
	}

	var data fiber.Map
	if err := json.Unmarshal(body, &data); err != nil {
		return resp.StatusCode, nil
	}

	return resp.StatusCode, data
}

func extractSession(resp *http.Response) *testSession {
	session := &testSession{}
	for _, c := range resp.Cookies() {
		session.Cookies = append(session.Cookies, c)
		if c.Name == "csrf_token" {
			session.CsrfToken = c.Value
		}
	}
	return session
}

var hrefRe = regexp.MustCompile(`href="([^"]+)"`)

func extractVerificationToken() string {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msg, ok := testMailer.LastMessage()
		if ok {
			m := hrefRe.FindStringSubmatch(msg.Body)
			if len(m) > 1 {
				u, err := url.Parse(m[1])
				if err == nil {
					if tok := u.Query().Get("token"); tok != "" {
						return tok
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ""
}

func assertKind(t *testing.T, status int, data fiber.Map, expectedStatus int, expectedKind string) bool {
	t.Helper()

	if status != expectedStatus {
		t.Errorf("status: want %d, got %d", expectedStatus, status)
		return false
	}

	if expectedKind != "" {
		kind, _ := data["kind"].(string)
		if kind != expectedKind {
			t.Errorf("kind: want %s, got %s", expectedKind, kind)
			return false
		}
	}

	return true
}

func createVerifiedUser(t *testing.T, app *fiber.App, email, password, name, username string) *testSession {
	t.Helper()

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    email,
		"password": password,
		"name":     name,
		"username": username,
	}, nil)
	status, _ := parseResponse(t, resp)
	require.Equal(t, 200, status, "register failed")

	resp = testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
		"email":   email,
		"baseUrl": "http://localhost/auth/verification_email/verify",
	}, nil)
	status, _ = parseResponse(t, resp)
	require.Equal(t, 200, status, "send verification email failed")

	token := extractVerificationToken()
	require.NotEmpty(t, token, "no verification token in redis")

	resp = testRequest(t, app, http.MethodGet, "/auth/verification_email/verify?token="+token, nil, nil)
	status, _ = parseResponse(t, resp)
	require.Equal(t, 200, status, "verify email failed")

	resp = testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
		"email":    email,
		"password": password,
	}, nil)
	status, _ = parseResponse(t, resp)
	require.Equal(t, 200, status, "login failed")

	return extractSession(resp)
}

func cleanupDB(t *testing.T) {
	t.Helper()

	_, err := testDB.Exec(`TRUNCATE TABLE
		user_service.sessions,
		user_service.email_verification_tokens,
		user_service.retired_usernames,
		user_service.users
		CASCADE`)
	require.NoError(t, err)

	_, err = testDB.Exec(`TRUNCATE TABLE
		video_service.pending_videos,
		video_service.playlist_videos,
		video_service.playlists,
		video_service.watchlaters,
		video_service.videos
		CASCADE`)
	require.NoError(t, err)

	_, err = testDB.Exec(`TRUNCATE TABLE
		media_service.orphan_uploads,
		media_service.avatars,
		media_service.thumbnails,
		media_service.videos
		CASCADE`)
	require.NoError(t, err)

	_, err = testDB.Exec(`TRUNCATE TABLE
		reaction_service.comments,
		reaction_service.feelings,
		reaction_service.views
		CASCADE`)
	require.NoError(t, err)

	_, err = testDB.Exec(`TRUNCATE TABLE
		social_service.follows
		CASCADE`)
	require.NoError(t, err)

	_, err = testDB.Exec(`TRUNCATE TABLE
		notification_service.notifications
		CASCADE`)
	require.NoError(t, err)

	testRedis.FlushAll(context.Background())
	testMailer.Clear()
}
