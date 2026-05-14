package events

import "time"

const VideoUploadedEvent = "media_service.video_uploaded"

type VideoUploadedEventPayload struct {
	ObjectKey string
	Timestamp time.Time
}
