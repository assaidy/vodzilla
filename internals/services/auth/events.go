package auth

const EmailVerificationEvent = "auth.EmailVerificationEvent"

type EmailVerificationEventPayload struct {
	Email            string
	VerificationLink string
}
