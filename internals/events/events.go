package events

import "github.com/google/uuid"

const VideoIsReadyEvent = "media_service.video_is_ready"

type VideoIsReadyEventPayload struct {
	VideoId uuid.UUID
}

const UserDeletedEvent = "user_service.user_deleted"

type UserDeletedEventPayload struct {
	UserId uuid.UUID
}

const VideoDeletedEvent = "video_service.video_deleted"

type VideoDeletedEventPayload struct {
	VideoId uuid.UUID
}

const UploadExpiredEvent = "media_service.upload_expired"

type UploadExpiredEventPayload struct {
	VideoId uuid.UUID
}
