package events

import "time"

// TODO: why i'm putting timestamp in all events?
// should i put an general event with general metadata then the generic payload as a field?

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

const UploadExpiredEvent = "media_serivice.upload_expired"

type UploadExpiredEventPayload struct {
	VideoId   string
	Timestamp time.Time
}
