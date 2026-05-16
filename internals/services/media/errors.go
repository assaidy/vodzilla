package media

import "errors"

var (
	ErrNotFound                  = errors.New("not found")
	ErrInvalidCompleteUploadData = errors.New("invalid upload parts")
)
