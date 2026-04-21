package auth

import "errors"

var (
	ErrEmailConflict = errors.New("email conflict")
	ErrValidation    = errors.New("invalid data")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrUnverified    = errors.New("unverified")
	ErrNotFound      = errors.New("not found")
)
