package reaction

import "errors"

var (
	ErrParentCommentNotFound = errors.New("parent comment not found")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrFeelingNotFound       = errors.New("feeling not found")
)
