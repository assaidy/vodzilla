package events

import "time"

const VideoUploadedEvent = "media_service.video_uploaded"

type VideoUploadedEventPayload struct {
	VideoId   string
	Timestamp time.Time
}

// TODO: define events for cascading data cleanup:
//   - UserDeletedEvent   — consumed by video, reaction, media, etc. to mark user data as deleted (soft deletion)
//   - VideoDeletedEvent  — consumed by reaction, media, etc.
//   - UploadExpiredEvent — published by media cleanup worker when an expired upload is found,
//     consumed by video service to delete the pending video metadata
