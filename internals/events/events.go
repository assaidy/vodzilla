package events

const VideoUploadedEvent = "media_service.video_uploaded"

type VideoUploadedEventPayload struct {
	VideoId string
}

const UserDeletedEvent = "user_serivice.user_deleted"

type UserDeletedEventPayload struct {
	UserId string
}

const VideoDeletedEvent = "video_serivice.video_deleted"

type VideoDeletedEventPayload struct {
	VideoId string
}

const UploadExpiredEvent = "media_serivice.upload_expired"

type UploadExpiredEventPayload struct {
	VideoId string
}
