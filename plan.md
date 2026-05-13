# Implementation Plan

A staged, event-driven backend using schema-per-service architecture.

---

# Stage 0 — Foundations

## Goals
- Basic infrastructure
- Service scaffolding
- Event system baseline

## Tasks
- [x] Create repo structure (`/services/*`)
- [x] Define services:
  - [x] auth
  - [x] user
  - [x] video
  - [x] media
  - [x] reaction
  - [x] social
  - [x] search
  - [x] feed
  - [x] history
  - [x] moderation

- [x] Setup infrastructure (Docker):
  - [x] PostgreSQL (per service DB)
  - [x] Redis
  - [x] Message broker ~(NATS/Kafka)~ (redis pubsub)

- [x] API Gateway:
  - [x] routing

- [x] Shared libraries:
  - [x] structured logging
  - [x] error handling

---

# Stage 1 — Auth + User (Profiles)

## Goals

* Authentication
* Profile management

## Tasks

### Auth Service

* [x] Schema:

  * [x] users
  * [x] sessions

* [ ] Interface:

  ```go
  type AuthService interface {
      Signup(email, password string) (User, error)
      Login(email, password string) (Session, error)
      Logout(userID string) error
      DeleteAccount(userID string) error
  }
  ```

* [ ] Implement:

  * [ ] ~Argon2~ bcrypt password hashing
  * [ ] ~JWT (access + refresh)~ sessions

---

### User Service

* [ ] Schema:

  * [ ] profiles

* [ ] Interface:

  ```go
  type UserService interface {
      GetByUsername(username string) (Profile, error)
      UpdateProfile(userID string, input UpdateProfileInput) error
      UploadAvatar(userID string, fileURL string) error
      UploadBanner(userID string, fileURL string) error
  }
  ```

* [ ] Events:

  * [ ] profile_created
  * [ ] profile_updated
  * [ ] username_changed

---

# Stage 2 — Video + Media

## Goals

* Upload videos
* Process to HLS

## Tasks

### Video Service

* [ ] Schema:

  * [ ] videos
  * [ ] playlists
  * [ ] playlist_items

* [ ] Interface:

  ```go
  type VideoService interface {
      CreateVideo(ownerID string, input CreateVideoInput) (Video, error)
      GetVideo(id string) (Video, error)
      UpdateVideo(id string, input UpdateVideoInput) error
      DeleteVideo(id string) error
  }
  ```

* [ ] Events:

  * [ ] ~video_created~
  * [ ] video_deleted

---

### Media Service

* [ ] Interface:

  ```go
  type MediaService interface {
      GenerateUploadUrls(videoID string, input GenerateUploadURLsInput) (string, error)
      ConfirmUpload(videoID string, input ConfirmUploadInput) error
  }
  ```

* [ ] Tasks:

  * [ ] Signed upload URLs
  * [ ] FFmpeg HLS pipeline
  * [ ] Upload segments to storage

* [ ] Events:

  * [ ] video_uploaded
  * [ ] video_processing_started
  * [ ] video_processed

---

# Stage 3 — Streaming

## Goals

* Video playback

## Tasks

* [ ] Interface:

  ```go
  type StreamingService interface {
      GetStreamURL(videoID string) (string, error)
  }
  ```

* [ ] Setup:

  * [ ] HLS playlist serving
  * [ ] CDN or Nginx

* [ ] Optional:

  * [ ] subtitles (.vtt)

---

# Stage 4 — Reactions + Social

## Goals

* Engagement features

## Tasks

### Reaction Service

* [ ] Schema:

  * [ ] reactions (watch/like/dislike counts)
  * [ ] comments
  * [ ] watch_later

* [ ] Interface:

  ```go
  type ReactionService interface {
      LikeVideo(userID, videoID string) error
      DislikeVideo(userID, videoID string) error
      Comment(userID, videoID, text string) error
      GetComments(videoID string) ([]Comment, error)
      AddToWatchLater(userID, videoID string) error
      ReportVideo(userID, videoID, reason string) error
  }
  ```

* [ ] Events:

  * [ ] video_liked
  * [ ] comment_created

---

### Social Service

* [ ] Schema:

  * [ ] follows

* [ ] Interface:

  ```go
  type SocialService interface {
      Follow(followerID, followeeID string) error
      Unfollow(followerID, followeeID string) error
      GetFollowers(userID string) ([]User, error)
      GetFollowing(userID string) ([]User, error)
      ReportUser(userID, targetID, reason string) error
  }
  ```

* [ ] Events:

  * [ ] user_followed

---

# Stage 5 — Search (Event-Driven)

## Goals

* Search videos and profiles

## Tasks

### Search Service

* [ ] Setup search engine (OpenSearch)

* [ ] Define indexes:

  * [ ] videos
  * [ ] profiles

* [ ] Interface:

  ```go
  type SearchService interface {
      Search(query string, filters SearchFilters) (SearchResult, error)
      IndexVideo(doc VideoDocument) error
      IndexProfile(doc ProfileDocument) error
  }
  ```

* [ ] Consumers:

  * [ ] video_created
  * [ ] video_updated
  * [ ] video_deleted
  * [ ] profile_created
  * [ ] profile_updated
  * [ ] username_changed

* [ ] Tasks:

  * [ ] denormalization (username, counts)
  * [ ] idempotent upserts
  * [ ] retry mechanism
  * [ ] reconciliation job

---

# Stage 6 — Feed / Explore

## Goals

* Recommendations

## Tasks

### Feed Service

* [ ] Interface:

  ```go
  type FeedService interface {
      GetFeed(userID string) ([]Video, error)
      GetExplore() ([]Video, error)
  }
  ```

* [ ] Tasks:

  * [ ] consume engagement events
  * [ ] implement scoring model
  * [ ] cache results (Redis)

---

# Stage 7 — History

## Goals

* Track user activity

## Tasks

### History Service

* [ ] Schema:

  * [ ] watch_history
  * [ ] search_history

* [ ] Interface:

  ```go
  type HistoryService interface {
      RecordWatch(userID, videoID string, progress float64) error
      GetWatchHistory(userID string) ([]HistoryItem, error)
      RecordSearch(userID, query string) error
  }
  ```

* [ ] Events:

  * [ ] video_viewed

---

# Stage 8 — Moderation

## Goals

* Reporting system

## Tasks

### Moderation Service

* [ ] Schema:

  * [ ] reports

* [ ] Interface:

  ```go
  type ModerationService interface {
      ReportVideo(userID, videoID, reason string) error
      ReportUser(userID, targetID, reason string) error
      GetReports() ([]Report, error)
  }
  ```

---

# Cross-Cutting Concerns

## Security

* [ ] rate limiting (Redis)

## Data Integrity

* [ ] outbox pattern
* [ ] idempotency constraints
* [ ] circuit breaker pattern
* [ ] timestamp-based conflict resolution

## Observability

* [ ] logging
* [ ] metrics
* [ ] tracing

## Testing

* [ ] unit tests
* [ ] integration tests (docker-compose)
* [ ] event contract tests
