package events

import "github.com/google/uuid"

const UserDeletedEvent = "user_service.user_deleted"

type UserDeletedEventPayload struct {
	UserId uuid.UUID
}

const VideoDeletedEvent = "video_service.video_deleted"

type VideoDeletedEventPayload struct {
	VideoId uuid.UUID
}
