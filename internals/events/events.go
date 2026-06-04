package events

import "github.com/google/uuid"

const VideoUploadedEvent = "media_service.video_uploaded"

type VideoUploadedEventPayload struct {
	VideoId uuid.UUID
}

const UserDeletedEvent = "user_serivice.user_deleted"

type UserDeletedEventPayload struct {
	UserId uuid.UUID
}

const VideoDeletedEvent = "video_serivice.video_deleted"

type VideoDeletedEventPayload struct {
	VideoId uuid.UUID
}

const UploadExpiredEvent = "media_serivice.upload_expired"

type UploadExpiredEventPayload struct {
	VideoId uuid.UUID
}
