package handlers

import (
	"net/http"
	"testing"
)

func TestHandleFollow(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "soca@example.com", "Password123", "Social A", "soca")
	userB := createVerifiedUser(t, app, "socb@example.com", "Password123", "Social B", "socb")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: userA.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid user_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/not-a-uuid", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/00000000-0000-0000-0000-000000000000", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("self-follow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userA.ID.String(), nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 403, "SelfFollowNotAllowed")
	})

	t.Run("valid follow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("duplicate follow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "AlreadyFollowing")
	})
}

func TestHandleIsFollowing(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "isfa@example.com", "Password123", "IsF A", "isfa")
	userB := createVerifiedUser(t, app, "isfb@example.com", "Password123", "IsF B", "isfb")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/is_following", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/is_following", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/is_following", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("not following returns false", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/is_following", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		isFollowing, _ := data["isFollowing"].(bool)
		if isFollowing {
			t.Error("expected isFollowing to be false, got true")
		}
	})

	t.Run("following returns true", func(t *testing.T) {
		testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)

		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/is_following", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		isFollowing, _ := data["isFollowing"].(bool)
		if !isFollowing {
			t.Error("expected isFollowing to be true, got false")
		}
	})
}

func TestHandleGetFollowCounts(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "cnta@example.com", "Password123", "Count A", "cnta")
	userB := createVerifiedUser(t, app, "cntb@example.com", "Password123", "Count B", "cntb")

	testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userA.ID.String()+"/counts", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/counts", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/counts", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("A's followeds = 1", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userA.ID.String()+"/counts", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followeds, _ := data["followeds"].(float64)
		if followeds != 1 {
			t.Errorf("A's followeds: want 1, got %v", followeds)
		}
	})

	t.Run("B's followers = 1", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/counts", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followers, _ := data["followers"].(float64)
		if followers != 1 {
			t.Errorf("B's followers: want 1, got %v", followers)
		}
	})
}

func TestHandleGetFollowers(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "flwra@example.com", "Password123", "Follower A", "flwra")
	userB := createVerifiedUser(t, app, "flwrb@example.com", "Password123", "Follower B", "flwrb")

	testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/followers", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/followers", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/followers", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("valid", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/followers?limit=15", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("limit too small", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/followers?limit=5", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})

	t.Run("limit too large", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/followers?limit=200", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidLimit")
	})
}

func TestHandleGetFolloweds(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "fldsa@example.com", "Password123", "Followeds A", "fldsa")
	userB := createVerifiedUser(t, app, "fldsb@example.com", "Password123", "Followeds B", "fldsb")

	testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)

	t.Run("invalid UUID", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/not-a-uuid/followeds", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/00000000-0000-0000-0000-000000000000/followeds", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("valid", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/followeds?limit=15", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})
}

func TestHandleUnfollow(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "unfa@example.com", "Password123", "Unfollow A", "unfa")
	userB := createVerifiedUser(t, app, "unfb@example.com", "Password123", "Unfollow B", "unfb")

	testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+userB.ID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: userA.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+userB.ID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("invalid user_id", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/not-a-uuid", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/00000000-0000-0000-0000-000000000000", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("valid unfollow", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+userB.ID.String(), nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
	})

	t.Run("unfollow again (not following)", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/follows/"+userB.ID.String(), nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "NotFollowing")
	})
}

func TestHandleFollowCountsAfterUnfollow(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)

	userA := createVerifiedUser(t, app, "afta@example.com", "Password123", "After A", "afta")
	userB := createVerifiedUser(t, app, "aftb@example.com", "Password123", "After B", "aftb")

	testRequest(t, app, http.MethodPost, "/follows/"+userB.ID.String(), nil, userA.Session)
	testRequest(t, app, http.MethodDelete, "/follows/"+userB.ID.String(), nil, userA.Session)

	t.Run("A's followeds = 0", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userA.ID.String()+"/counts", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followeds, _ := data["followeds"].(float64)
		if followeds != 0 {
			t.Errorf("A's followeds: want 0, got %v", followeds)
		}
	})

	t.Run("B's followers = 0", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/follows/"+userB.ID.String()+"/counts", nil, userA.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		followers, _ := data["followers"].(float64)
		if followers != 0 {
			t.Errorf("B's followers: want 0, got %v", followers)
		}
	})
}
