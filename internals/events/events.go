package events

import "time"

const VideoUploadedEvent = "media_service.video_uploaded"

type VideoUploadedEventPayload struct {
	VideoId   string
	Timestamp time.Time
}

const UserDeletedEvent = "user_serivice.user_deleted"

type UserDeletedEventPayload struct {
	UserId    string
	Timestamp time.Time
}

const VideoDeletedEvent = "video_serivice.video_deleted"

type VideoDeletedEventPayload struct {
	VideoId   string
	Timestamp time.Time
}

const UploadDeletedEvent = "media_serivice.upload_expired"

type UploadExpiredEventPayload struct {
	VideoId   string
	Timestamp time.Time
}
