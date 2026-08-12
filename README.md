# Vodzilla

Vodzilla Is A Video Sharing Platform (YouTube-Like) Backend API.

## Run the Application

```sh
docker compose up
make goose-up-all
make run
```

## Functional Features

### Authentication & Accounts
- **Registration:** Email, password, name, and username with strict validation (email format, password length 8–50, username charset `[A-Za-z0-9_]`).
- **Email Verification:** Verification email with a token link; login is blocked until the email is verified.
- **Sessions:** Login/logout using HTTP-only session cookies with a separate session token and an expiry.
- **Password Management:** Change password after verifying the current one (bcrypt hashed).
- **Username Retirement:** Deleted usernames are retired to prevent reuse.

### User Profiles
- **Profile Management:** View your own profile, by username, or by ID; edit name, username, and bio; delete your account.
- **Avatars:** Presigned upload, upload confirmation, deletion, and retrieval via S3-compatible object storage.
- **Cascading Account Deletion:** Deleting a profile asynchronously removes the user's videos, pending uploads, watchlaters, playlists, saved playlists, reactions, notifications, and history.

### Videos
- **Chunked Uploads:** Multipart uploads to S3-compatible storage via presigned put URLs (5 MB parts), with upload confirmation and validation.
- **Video Metadata:** Post videos with title and description.
- **Thumbnails:** Presigned upload, confirmation, deletion, and retrieval.
- **Streaming:** Presigned streaming URLs, video details, and video deletion (owner-only).
- **Per-User Listing:** Paginated list of a user's videos and a video count.

### Search
- **Full-Text Search:** Ranked search across videos (title, description) and profiles (name, username, bio) using PostgreSQL `tsvector` + `ts_rank`, with cursor-based pagination.

### Social
- **Follow System:** Follow/unfollow users, follower and followed lists, counts, and is-following checks, with self-follow prevention.

### Reactions
- **Views:** Record views on videos and playlists, with view counts.
- **Feelings:** Like/dislike videos and comments, feeling counts, and the current user's feeling.
- **Comments:** Comment on videos, nested replies, edit, delete (cascades to replies), counts, and paginated listings.

### Watch Later & Playlists
- **Watch Later:** Add, remove, and list videos saved for later.
- **Playlists:** Create, edit, delete (public/private), add/remove videos, view videos by playlist and playlist status per video.
- **Saved Playlists:** Save/unsave other users' playlists and list saved playlists.

### Feed
- **Following Feed:** Paginated feed of videos from users you follow.

### Notifications
- **Six Kinds:** Follow, new video, video feeling, comment feeling, video comment, and comment reply.
- **Real-Time Delivery:** Pushed to connected clients over WebSocket via Redis Pub/Sub.
- **Management:** Paginated listing, unread count, and mark-as-read.

### Watch History
- **Tracking:** Record a watch entry per video view.
- **Management:** Paginated history, delete a single entry, or clear the entire history.

### Misc
- **Health Check** endpoint for readiness probes.

## Non-Functional Features

### Performance & Scalability
- **Prefork Mode:** Fiber runs in multi-process prefork mode.
- **Cursor Pagination:** All list endpoints paginate with encoded cursors and bounded limits (15–100).
- **Full-Text Indexing:** GIN indexes and generated `tsvector` columns for fast, ranked search.
- **Connection Pooling:** Tuned PostgreSQL pool (max open/idle connections, lifetimes).

### Reliability
- **Background Workers:** Scheduled jobs (weekly, single-instance via Redis distributed locks) that clean up expired sessions, expired email verification tokens, pending video uploads, and orphaned uploads.
- **Event-Driven Cleanup:** Redis-backed event queues drive asynchronous cascading deletes (user deleted, video deleted) across services with retry and decorrelated-jitter backoff.
- **Graceful Shutdown:** Signal handling with a bounded shutdown timeout.
- **Distributed Locking:** Redis read/write locks serialize critical operations (e.g., logout vs. session use, profile deletion vs. concurrent requests).

### Security
- **Password Security:** bcrypt hashing and minimum password strength rules.
- **Session Cookies:** HTTP-only cookies plus a separate session token and expiry.
- **CSRF Protection:** All mutating endpoints require a CSRF token.
- **Input Validation:** Every endpoint validates request data (formats, lengths, upload content types, and size limits).
- **Presigned URL Expiry:** Short-lived presigned URLs for uploads and retrieval.

### Observability
- **Request Logging:** Structured logs with method, path, status, duration, and client IP.
- **Service-Scoped Logging:** Grouped structured logging (slog) per service.

### Architecture & Tooling
- **Modular Services:** Isolated domains (user, video, media, reaction, social, notification, history), each with its own DB schema, goose migrations, and sqlc-generated type-safe queries.
- **Lifecycle Management:** Central service manager for coordinated start/stop.
- **Docker Compose Dev Stack:** PostgreSQL, Redis, Papercut (SMTP), and RustFS (S3-compatible) with a Makefile-driven workflow (`make build`, `make run`, goose targets, CLI helpers).
- **Testability:** Integration tests with testcontainers (Postgres, Redis, MinIO) plus unit tests for locking and pagination primitives.

## Tech Stack

- **Backend:** Go (Fiber v3)
- **Database:** PostgreSQL (with sqlc for type-safe queries and goose for migrations)
- **Cache, Queue & Pub/Sub:** Redis (go-redis) with distributed locking
- **Object Storage:** S3-compatible storage via RustFS (AWS SDK for Go v2)
- **Email:** Papercut (SMTP) with gomail
- **Real-Time:** WebSocket (Fiber v3)
- **Background Jobs:** <github.com/assaidy/workers> with Redis locks
- **Tooling:** Make, Docker (including Docker Compose), sqlc, and goose
