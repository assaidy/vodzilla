package video

import "errors"

var (
	ErrVideoNotFound           = errors.New("video not found")
	ErrPlaylistNotFound        = errors.New("playlist not found")
	ErrWatchlaterConflict      = errors.New("already in watch later")
	ErrWatchlaterVideoNotFound = errors.New("video not found in watchlater")
	ErrPlaylistVideoConflict   = errors.New("video already in playlist")
	ErrPlaylistVideoNotFound   = errors.New("video not found in playlist")
)
