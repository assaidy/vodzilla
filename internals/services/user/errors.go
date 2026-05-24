package user

import "errors"

var (
	ErrEmailConflict    = errors.New("email conflict")
	ErrUsernameConflict = errors.New("username conflict")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrUnverified       = errors.New("unverified")
	ErrUserNotFound     = errors.New("user not found")
	ErrSessionNotFound  = errors.New("session not found")
	ErrTokenNotFound    = errors.New("token not found")
)
