package handlers

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHandleCreatePlaylist(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "plcre@example.com", "Password123", "PL Create", "plcre")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "My Playlist"}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "My Playlist"}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("validation empty name", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": ""}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation name too long", func(t *testing.T) {
		longName := ""
		for i := 0; i < 51; i++ {
			longName += "a"
		}
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": longName}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation whitespace only", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "   "}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "My Playlist"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		playlistID, _ := data["playlistId"].(string)
		require.NotEmpty(t, playlistID)
		_, err := uuid.Parse(playlistID)
		require.NoError(t, err)
	})

}

func TestHandleGetPlaylists(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "plget@example.com", "Password123", "PL Get", "plget")
	otherUser := createVerifiedUser(t, app, "plgetother@example.com", "Password123", "PL Get Other", "plgetother")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("empty list", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String(), nil, user.Session)
		status, data := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, data)
	})

	t.Run("success with playlists", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "PL One"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		pl1, _ := data["playlistId"].(string)

		resp = testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "PL Two"}, user.Session)
		status, data = parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		pl2, _ := data["playlistId"].(string)

		resp = testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String(), nil, user.Session)
		status, dataArr := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, dataArr, 2)
		ids := make(map[string]bool)
		for _, item := range dataArr {
			m := item.(map[string]any)
			ids[m["id"].(string)] = true
		}
		require.True(t, ids[pl1])
		require.True(t, ids[pl2])
	})

	t.Run("other user has no playlists", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/"+otherUser.ID.String(), nil, user.Session)
		status, data := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, data)
	})
}

func TestHandleGetPlaylistsWithVideoStatus(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "plwvs@example.com", "Password123", "PL WVS", "plwvs")
	videoID := createVerifiedVideo(t, user.ID, "PL WVS Video", "")

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String()+"/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/00000000-0000-0000-0000-000000000000/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "UserNotFound")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String()+"/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("empty list", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String()+"/videos/"+videoID.String(), nil, user.Session)
		status, data := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, data)
	})

	t.Run("success with hasVideo", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "WVS PL"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		plID, _ := data["playlistId"].(string)

		resp = testRequest(t, app, http.MethodPost, "/playlists/"+plID+"/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String()+"/videos/"+videoID.String(), nil, user.Session)
		status, dataArr := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, dataArr, 1)
		item := dataArr[0].(map[string]any)
		require.Equal(t, plID, item["id"])
		require.Equal(t, true, item["hasVideo"])
	})
}

func TestHandleGetPlaylist(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "plgetpl@example.com", "Password123", "PL Get PL", "plgetpl")
	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "Get PL Test"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/"+playlistID, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/"+playlistID, nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 200, "")
		name, _ := data["name"].(string)
		require.Equal(t, "Get PL Test", name)
	})
}

func TestHandleRenamePlaylist(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "plren@example.com", "Password123", "PL Ren", "plren")
	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "Old Name"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/playlists/"+playlistID, fiber.Map{"name": "New Name"}, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPut, "/playlists/"+playlistID, fiber.Map{"name": "New Name"}, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("validation empty name", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/playlists/"+playlistID, fiber.Map{"name": ""}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/playlists/00000000-0000-0000-0000-000000000000", fiber.Map{"name": "New Name"}, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPut, "/playlists/"+playlistID, fiber.Map{"name": "New Name"}, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String(), nil, user.Session)
		status, dataArr := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, dataArr, 1)
		item := dataArr[0].(map[string]any)
		require.Equal(t, "New Name", item["name"])
	})
}

func TestHandleDeletePlaylist(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "pldel@example.com", "Password123", "PL Del", "pldel")
	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "To Delete"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID, nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID, nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID, nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/playlists/users/"+user.ID.String(), nil, user.Session)
		status, dataArr := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, dataArr)
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID, nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})
}

func TestHandleAddVideoToPlaylist(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "pladdv@example.com", "Password123", "PL AddV", "pladdv")
	videoID := createVerifiedVideo(t, user.ID, "PL Add Video", "")
	otherUser := createVerifiedUser(t, app, "pladdvoth@example.com", "Password123", "PL AddV Oth", "pladdvoth")
	otherVideo := createVerifiedVideo(t, otherUser.ID, "PL Add Other Video", "")

	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "Add Video PL"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists/00000000-0000-0000-0000-000000000000/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("other user's video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/"+otherVideo.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("already in playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 409, "PlaylistVideoConflict")
	})
}

func TestHandleDeleteVideoFromPlaylist(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "pldelv@example.com", "Password123", "PL DelV", "pldelv")
	videoID := createVerifiedVideo(t, user.ID, "PL Del Video", "")

	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "Del Video PL"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	resp = testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, user.Session)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("no CSRF", func(t *testing.T) {
		noCsrf := &testSession{Cookies: user.Session.Cookies}
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, noCsrf)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent video", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID+"/videos/00000000-0000-0000-0000-000000000000", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "VideoNotFound")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/00000000-0000-0000-0000-000000000000/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, user.Session)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		resp = testRequest(t, app, http.MethodGet, "/playlists/"+playlistID+"/videos", nil, user.Session)
		status, dataArr := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, dataArr)
	})

	t.Run("delete again", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodDelete, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistVideoNotFound")
	})
}

func TestHandleGetPlaylistVideos(t *testing.T) {
	defer resetDb(t)
	app := newTestApp(t)
	user := createVerifiedUser(t, app, "plgetpv@example.com", "Password123", "PL Get PV", "plgetpv")
	videoID := createVerifiedVideo(t, user.ID, "PL Playlist Video", "")

	resp := testRequest(t, app, http.MethodPost, "/playlists", fiber.Map{"name": "Video List PL"}, user.Session)
	status, data := parseResponse(t, resp)
	assertKind(t, status, data, 200, "")
	playlistID, _ := data["playlistId"].(string)

	resp = testRequest(t, app, http.MethodPost, "/playlists/"+playlistID+"/videos/"+videoID.String(), nil, user.Session)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Run("no session", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/"+playlistID+"/videos", nil, nil)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 401, "Unauthorized")
	})

	t.Run("non-existent playlist", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/00000000-0000-0000-0000-000000000000/videos", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 404, "PlaylistNotFound")
	})

	t.Run("validation limit too low", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/"+playlistID+"/videos?limit=1", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("validation limit too high", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/"+playlistID+"/videos?limit=200", nil, user.Session)
		status, data := parseResponse(t, resp)
		assertKind(t, status, data, 400, "InvalidData")
	})

	t.Run("success", func(t *testing.T) {
		resp := testRequest(t, app, http.MethodGet, "/playlists/"+playlistID+"/videos", nil, user.Session)
		status, dataArr := parseArrayResponse(t, resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, dataArr, 1)
		item := dataArr[0].(map[string]any)
		require.NotEmpty(t, item["playlistVideoId"])
		require.Equal(t, videoID.String(), item["id"])
	})
}
