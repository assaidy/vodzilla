package video

import "errors"

var (
	ErrVideoNotFound          = errors.New("video not found")
	ErrPlaylistNotFound       = errors.New("playlist not found")
	ErrWatchlaterConflict     = errors.New("already in watch later")
	ErrPlaylistNameConflict   = errors.New("playlist name already exists")
	ErrPlaylistVideoConflict  = errors.New("video already in playlist")
)
