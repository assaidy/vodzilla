package media

import "errors"

var (
	ErrObjectNotFound            = errors.New("object not found")
	ErrInvalidCompleteUploadData = errors.New("invalid upload parts")
)
