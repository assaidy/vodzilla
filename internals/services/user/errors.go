package user

import "errors"

var (
	ErrEmailConflict    = errors.New("email conflict")
	ErrUsernameConflict = errors.New("username conflict")
	ErrValidation       = errors.New("invalid data")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrUnverified       = errors.New("unverified")
	ErrNotFound         = errors.New("not found")
)
