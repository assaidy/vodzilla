package video

import "errors"

var (
	ErrVideoNotFound    = errors.New("video not found")
	ErrPlaylistNotFound = errors.New("playlist not found")
	ErrConflict         = errors.New("conflict")
)
