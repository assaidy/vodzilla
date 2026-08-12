# API Docs

Auth requirements: **Session** = session cookies required (`session_id` + `session_token`). **CSRF** = `X-CSRF-Token` header required.

## Common Conventions

**Path params**: `:user_id` (uuid), `:video_id` (uuid), `:comment_id` (uuid), `:playlist_id` (uuid), `:entry_id` (int), `:notification_id` (uuid), `:username` (string).

**Query params (GET lists / search)**:
- `limit` (int, 15–100, default 15)
- `cursor` (base64 string, opaque pagination token)
- `query` (string, 1–50, required for search)

**JSON body field types**: `str`, `int`, `bool`. Length/format rules noted inline (e.g. `str(1-256)`).

---

## Misc

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| GET | `/health` | Health check | no | no | — |
| GET | `/ws` | Real-time notifications over WebSocket (requires upgrade) | yes | no | — |

---

## Auth

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| POST | `/auth/register` | Register a new user | no | no | body: `{email:str(email), password:str(8-50), name:str(1-256), username:str(1-32, [A-Za-z0-9_])}` |
| POST | `/auth/login` | Log in and set session cookies | no | no | body: `{email:str(email), password:str(8-50)}` |
| POST | `/auth/logout` | Log out and invalidate the session | yes | no | — |
| POST | `/auth/verification_email` | Send email verification link | no | no | body: `{email:str(email), baseUrl:str(url)}` |
| GET | `/auth/verification_email/verify` | Verify email with token | no | no | query: `token` (string, required) |
| PUT | `/auth/password` | Change password | yes | yes | body: `{currentPassword:str(8-50), newPassword:str(8-50)}` |

---

## Profiles

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| GET | `/profiles/me` | Get own profile | yes | no | — |
| GET | `/profiles` | Search profiles | yes | no | query: `query` (str 1-50, required), `limit?`, `cursor?` |
| GET | `/profiles/usernames/:username` | Get profile by username | yes | no | param: `:username` |
| GET | `/profiles/id/:user_id` | Get profile by ID | yes | no | param: `:user_id` (uuid) |
| PUT | `/profiles` | Edit name, username, bio | yes | yes | body: `{name:str(1-256), username:str(1-32,[A-Za-z0-9_]), bio:str(0-500)}` |
| DELETE | `/profiles` | Delete account | yes | yes | — |
| PUT | `/profiles/avatar` | Generate avatar upload URL | yes | yes | body: `{contentType:str(image/*), fileSize:int(≤5MB)}` |
| PUT | `/profiles/avatar/confirm_upload` | Confirm avatar upload | yes | yes | — |
| DELETE | `/profiles/avatar` | Delete avatar | yes | yes | — |
| GET | `/profiles/:user_id/avatar` | Get avatar URL | yes | no | param: `:user_id` (uuid) |

---

## Social

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| POST | `/follows/:user_id` | Follow a user | yes | yes | param: `:user_id` (uuid) |
| DELETE | `/follows/:user_id` | Unfollow a user | yes | yes | param: `:user_id` (uuid) |
| GET | `/follows/:user_id/counts` | Get followers/followeds counts | yes | no | param: `:user_id` (uuid) |
| GET | `/follows/:user_id/is_following` | Check if following | yes | no | param: `:user_id` (uuid) |
| GET | `/follows/:user_id/followers` | Get followers (paginated) | yes | no | param: `:user_id` (uuid); query: `limit?`, `cursor?` |
| GET | `/follows/:user_id/followeds` | Get followeds (paginated) | yes | no | param: `:user_id` (uuid); query: `limit?`, `cursor?` |

---

## Videos

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| POST | `/videos/upload` | Generate chunked upload URLs | yes | yes | body: `{contentType:str(video/*), fileSize:int(≤32GB)}` |
| PUT | `/videos/upload/confirm` | Confirm multipart upload | yes | yes | body: `{uploadId:str, objectKey:str, parts:[{etag:str, partNumber:int}]}` |
| POST | `/videos` | Post video metadata | yes | yes | body: `{title:str(1-256), description:str(0-500), objectKey:str}` |
| PUT | `/videos/:video_id/thumbnail` | Generate thumbnail upload URL | yes | yes | param: `:video_id` (uuid); body: `{contentType:str(image/*), fileSize:int(≤5MB)}` |
| PUT | `/videos/:video_id/thumbnail/confirm_upload` | Confirm thumbnail upload | yes | yes | param: `:video_id` (uuid) |
| DELETE | `/videos/:video_id/thumbnail` | Delete thumbnail | yes | yes | param: `:video_id` (uuid) |
| GET | `/videos/:video_id/thumbnail` | Get thumbnail URL | yes | no | param: `:video_id` (uuid) |
| GET | `/videos/:video_id` | Get video details | yes | no | param: `:video_id` (uuid) |
| GET | `/videos/:video_id/stream_url` | Get stream URL | yes | no | param: `:video_id` (uuid) |
| DELETE | `/videos/:video_id` | Delete video (owner only) | yes | yes | param: `:video_id` (uuid) |
| GET | `/videos/users/:user_id` | Get user's videos (paginated) | yes | no | param: `:user_id` (uuid); query: `limit?`, `cursor?` |
| GET | `/videos/users/:user_id/count` | Get user's video count | yes | no | param: `:user_id` (uuid) |
| GET | `/videos/` | Search videos | yes | no | query: `query` (str 1-50, required), `limit?`, `cursor?` |

---

## Watch Later

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| GET | `/watchlaters` | Get watch later videos | yes | no | query: `limit?`, `cursor?` |
| POST | `/watchlaters/videos/:video_id` | Add video to watch later | yes | yes | param: `:video_id` (uuid) |
| DELETE | `/watchlaters/videos/:video_id` | Remove video from watch later | yes | yes | param: `:video_id` (uuid) |

---

## Playlists

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| POST | `/playlists` | Create a playlist | yes | yes | body: `{name:str(1-50), description:str(0-500), isPublic:bool}` |
| GET | `/playlists/users/:user_id` | Get user's playlists | yes | no | param: `:user_id` (uuid); query: `limit?`, `cursor?` |
| GET | `/playlists/videos/:video_id` | Get playlists with video status | yes | no | param: `:video_id` (uuid); query: `limit?`, `cursor?` |
| GET | `/playlists/:playlist_id` | Get playlist details | yes | no | param: `:playlist_id` (uuid) |
| GET | `/playlists/:playlist_id/videos` | Get playlist videos (paginated) | yes | no | param: `:playlist_id` (uuid); query: `limit?`, `cursor?` |
| DELETE | `/playlists/:playlist_id` | Delete a playlist | yes | yes | param: `:playlist_id` (uuid) |
| PUT | `/playlists/:playlist_id` | Edit a playlist | yes | yes | param: `:playlist_id` (uuid); body: `{name:str(1-50), description:str(0-500), isPublic:bool}` |
| POST | `/playlists/:playlist_id/videos/:video_id` | Add video to playlist | yes | yes | params: `:playlist_id` (uuid), `:video_id` (uuid) |
| DELETE | `/playlists/:playlist_id/videos/:video_id` | Remove video from playlist | yes | yes | params: `:playlist_id` (uuid), `:video_id` (uuid) |
| GET | `/playlists/saved/list` | Get saved playlists | yes | no | query: `limit?`, `cursor?` |
| POST | `/playlists/saved/:playlist_id` | Save a playlist | yes | yes | param: `:playlist_id` (uuid) |
| DELETE | `/playlists/saved/:playlist_id` | Remove from saved playlists | yes | yes | param: `:playlist_id` (uuid) |

---

## Reactions

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| POST | `/reactions/views/videos/:video_id` | Record a video view | yes | yes | param: `:video_id` (uuid) |
| GET | `/reactions/views/videos/:video_id/count` | Get video views count | yes | no | param: `:video_id` (uuid) |
| POST | `/reactions/views/playlists/:playlist_id` | Record a playlist view | yes | yes | param: `:playlist_id` (uuid) |
| GET | `/reactions/views/playlists/:playlist_id/count` | Get playlist views count | yes | no | param: `:playlist_id` (uuid) |
| POST | `/reactions/comments/videos/:video_id` | Create a comment on a video | yes | yes | param: `:video_id` (uuid); body: `{content:str(1-500)}` |
| GET | `/reactions/comments/videos/:video_id` | Get video comments (paginated) | yes | no | param: `:video_id` (uuid); query: `limit?`, `cursor?` |
| POST | `/reactions/comments/:comment_id/replies` | Create a comment reply | yes | yes | param: `:comment_id` (uuid); body: `{content:str(1-500)}` |
| GET | `/reactions/comments/:comment_id/replies` | Get comment replies (paginated) | yes | no | param: `:comment_id` (uuid); query: `limit?`, `cursor?` |
| PUT | `/reactions/comments/:comment_id` | Edit a comment | yes | yes | param: `:comment_id` (uuid); body: `{content:str(1-500)}` |
| DELETE | `/reactions/comments/:comment_id` | Delete a comment | yes | yes | param: `:comment_id` (uuid) |
| POST | `/reactions/feelings/videos/:video_id` | Add/update feeling on a video | yes | yes | param: `:video_id` (uuid); body: `{kind:str("like"|"dislike")}` |
| DELETE | `/reactions/feelings/videos/:video_id` | Remove feeling from a video | yes | yes | param: `:video_id` (uuid) |
| GET | `/reactions/feelings/videos/:video_id/counts` | Get video feeling counts | yes | no | param: `:video_id` (uuid) |
| GET | `/reactions/feelings/videos/:video_id/user` | Get current user's video feeling | yes | no | param: `:video_id` (uuid) |
| POST | `/reactions/feelings/comments/:comment_id` | Add/update feeling on a comment | yes | yes | param: `:comment_id` (uuid); body: `{kind:str("like"|"dislike")}` |
| DELETE | `/reactions/feelings/comments/:comment_id` | Remove feeling from a comment | yes | yes | param: `:comment_id` (uuid) |
| GET | `/reactions/feelings/comments/:comment_id/counts` | Get comment feeling counts | yes | no | param: `:comment_id` (uuid) |
| GET | `/reactions/feelings/comments/:comment_id/user` | Get current user's comment feeling | yes | no | param: `:comment_id` (uuid) |

---

## Feed

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| GET | `/feed` | Get feed of followed users' videos (paginated) | yes | no | query: `limit?`, `cursor?` |

---

## Notifications

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| GET | `/notifications/notifications` | Get notifications (paginated) | yes | no | query: `limit?`, `cursor?` |
| GET | `/notifications/notifications/count` | Get unread notifications count | yes | no | — |
| POST | `/notifications/:notification_id/mark_read` | Mark a notification as read | yes | yes | param: `:notification_id` (uuid) |

---

## History

| Method | Path | Description | Session | CSRF | Request |
|--------|------|-------------|:-------:|:----:|---------|
| GET | `/history` | Get watch history (paginated) | yes | no | query: `limit?`, `cursor?` |
| POST | `/history/videos/:video_id` | Add to watch history | yes | yes | param: `:video_id` (uuid) |
| DELETE | `/history/:entry_id` | Delete a history entry | yes | yes | param: `:entry_id` (int) |
| DELETE | `/history` | Clear watch history | yes | yes | — |
