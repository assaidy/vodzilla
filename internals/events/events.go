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

const EmailVerificationEvent = "user_service.email_verification"

type EmailVerificationEventPayload struct {
	Email            string
	VerificationLink string
}

func queueName(eventName string) string {
	return "queue:" + eventName
}
