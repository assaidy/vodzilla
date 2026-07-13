package handlers

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHandleFollow(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	sessionA := createVerifiedUser(t, app, "soca@example.com", "Password123", "Social A", "soca")

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "socb@example.com",
		"password": "Password123",
		"name":     "Social B",
		"username": "socb",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	resp = testRequest(t, app, http.MethodGet, "/profiles/usernames/socb", nil, sessionA)
	status, data := parseResponse(t, resp)
	_ = status
	bId, _ := data["id"].(string)

	resp = testRequest(t, app, http.MethodGet, "/profiles", nil, sessionA)
	status, data = parseResponse(t, resp)
	_ = status
	aId, _ := data["id"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: sessionA.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid user_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/not-a-uuid", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/00000000-0000-0000-0000-000000000000", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("self-follow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+aId, nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 403, "SelfFollowNotAllowed")
	})

	t.Run("valid follow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("duplicate follow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "AlreadyFollowing")
	})
}

func TestHandleIsFollowing(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	sessionA := createVerifiedUser(t, app, "isfa@example.com", "Password123", "IsF A", "isfa")

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "isfb@example.com",
		"password": "Password123",
		"name":     "IsF B",
		"username": "isfb",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	resp = testRequest(t, app, http.MethodGet, "/profiles/usernames/isfb", nil, sessionA)
	status, data := parseResponse(t, resp)
	_ = status
	bId, _ := data["id"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/is_following", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/is_following", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/is_following", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("not following returns false", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/is_following", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		isFollowing, _ := data["isFollowing"].(bool)
		if isFollowing {
			t.Error("expected isFollowing to be false, got true")
		}
	})

	t.Run("following returns true", func(t *testing.T) {
		testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)

		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/is_following", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		isFollowing, _ := data["isFollowing"].(bool)
		if !isFollowing {
			t.Error("expected isFollowing to be true, got false")
		}
	})
}

func TestHandleGetFollowCounts(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	sessionA := createVerifiedUser(t, app, "cnta@example.com", "Password123", "Count A", "cnta")

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "cntb@example.com",
		"password": "Password123",
		"name":     "Count B",
		"username": "cntb",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	resp = testRequest(t, app, http.MethodGet, "/profiles/usernames/cntb", nil, sessionA)
	status, data := parseResponse(t, resp)
	_ = status
	bId, _ := data["id"].(string)

	resp = testRequest(t, app, http.MethodGet, "/profiles", nil, sessionA)
	status, data = parseResponse(t, resp)
	_ = status
	aId, _ := data["id"].(string)

	testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+aId+"/counts", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/counts", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/counts", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("A's followeds = 1", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+aId+"/counts", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followeds, _ := data["followeds"].(float64)
		if followeds != 1 {
			t.Errorf("A's followeds: want 1, got %v", followeds)
		}
	})

	t.Run("B's followers = 1", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/counts", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followers, _ := data["followers"].(float64)
		if followers != 1 {
			t.Errorf("B's followers: want 1, got %v", followers)
		}
	})
}

func TestHandleGetFollowers(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	sessionA := createVerifiedUser(t, app, "flwra@example.com", "Password123", "Follower A", "flwra")

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "flwrb@example.com",
		"password": "Password123",
		"name":     "Follower B",
		"username": "flwrb",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	resp = testRequest(t, app, http.MethodGet, "/profiles/usernames/flwrb", nil, sessionA)
	status, data := parseResponse(t, resp)
	_ = status
	bId, _ := data["id"].(string)

	testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/followers", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/followers", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/followers", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("valid", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/followers?limit=15", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("limit too small", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/followers?limit=5", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("limit too large", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/followers?limit=200", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})
}

func TestHandleGetFolloweds(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	sessionA := createVerifiedUser(t, app, "fldsa@example.com", "Password123", "Followeds A", "fldsa")

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "fldsb@example.com",
		"password": "Password123",
		"name":     "Followeds B",
		"username": "fldsb",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	resp = testRequest(t, app, http.MethodGet, "/profiles/usernames/fldsb", nil, sessionA)
	status, data := parseResponse(t, resp)
	_ = status
	bId, _ := data["id"].(string)

	testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/followeds", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/followeds", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("valid", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/followeds?limit=15", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleUnfollow(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	sessionA := createVerifiedUser(t, app, "unfa@example.com", "Password123", "Unfollow A", "unfa")

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "unfb@example.com",
		"password": "Password123",
		"name":     "Unfollow B",
		"username": "unfb",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	resp = testRequest(t, app, http.MethodGet, "/profiles/usernames/unfb", nil, sessionA)
	status, data := parseResponse(t, resp)
	_ = status
	bId, _ := data["id"].(string)

	testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+bId, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: sessionA.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+bId, nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid user_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/not-a-uuid", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidRequest")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/00000000-0000-0000-0000-000000000000", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("valid unfollow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+bId, nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("unfollow again (not following)", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+bId, nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "NotFollowing")
	})
}

func TestHandleFollowCountsAfterUnfollow(t *testing.T) {
	cleanupDB(t)
	app := newTestApp(t)

	sessionA := createVerifiedUser(t, app, "afta@example.com", "Password123", "After A", "afta")

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    "aftb@example.com",
		"password": "Password123",
		"name":     "After B",
		"username": "aftb",
	}, nil)
	status, _ := parseResponse(t, resp)
	_ = status

	resp = testRequest(t, app, http.MethodGet, "/profiles/usernames/aftb", nil, sessionA)
	status, data := parseResponse(t, resp)
	_ = status
	bId, _ := data["id"].(string)

	resp = testRequest(t, app, http.MethodGet, "/profiles", nil, sessionA)
	status, data = parseResponse(t, resp)
	_ = status
	aId, _ := data["id"].(string)

	testRequest(t, app, http.MethodPost, "/follows/"+bId, nil, sessionA)
	testRequest(t, app, http.MethodDelete, "/follows/"+bId, nil, sessionA)

	t.Run("A's followeds = 0", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+aId+"/counts", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followeds, _ := data["followeds"].(float64)
		if followeds != 0 {
			t.Errorf("A's followeds: want 0, got %v", followeds)
		}
	})

	t.Run("B's followers = 0", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+bId+"/counts", nil, sessionA)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followers, _ := data["followers"].(float64)
		if followers != 0 {
			t.Errorf("B's followers: want 0, got %v", followers)
		}
	})
}
