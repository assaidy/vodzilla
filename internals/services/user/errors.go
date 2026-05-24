package user

import "errors"

var (
	ErrEmailConflict    = errors.New("email conflict")
	ErrUsernameConflict = errors.New("username conflict")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrUnverified       = errors.New("unverified")
	ErrNotFound         = errors.New("not found")
)
