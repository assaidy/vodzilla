package social

import "errors"

var (
	ErrSelfFollowNotAllowed = errors.New("self follow not allowed")
	ErrAlreadyFollowing     = errors.New("already following")
	ErrNotFollowing         = errors.New("not following")
)
