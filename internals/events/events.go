package events

import "time"

const VideoUploadedEvent = "media_service.video_uploaded"

type VideoUploadedEventPayload struct {
	VideoId   string
	Timestamp time.Time
}
