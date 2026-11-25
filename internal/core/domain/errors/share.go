package errors

import "errors"

var (
	ErrAmbiguousShareReference = errors.New("share reference belongs to multiple users")
	ErrShareNotFound           = errors.New("share not found")
	ErrShareAlreadyExists      = errors.New("share already exists")
)
