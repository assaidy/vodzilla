package media

import "errors"

var (
	ErrObjectNotFound            = errors.New("object not found")
	ErrUploadAlreadyCompleted    = errors.New("upload already complete")
	ErrUploadExpired             = errors.New("upload expired")
	ErrInvalidCompleteUploadData = errors.New("invalid upload parts")
)
