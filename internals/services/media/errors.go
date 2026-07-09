package media

import "errors"

var (
	ErrVideoNotFound                 = errors.New("video not found")
	ErrInvalidConfirmVideoUploadData = errors.New("invalid upload parts")
	ErrNoPendingVideoUpload          = errors.New("no pending video upload")
	ErrOrphanUploadNotFound          = errors.New("orphan upload not found")

	ErrAvatarNotFound        = errors.New("avatar not found")
	ErrNoPendingAvatarUpload = errors.New("no pending avatar upload")

	ErrThumbnailNotFound        = errors.New("thumbnail not found")
	ErrNoPendingThumbnailUpload = errors.New("no pending thumbnail upload")
)
