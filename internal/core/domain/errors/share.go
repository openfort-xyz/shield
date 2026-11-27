package errors

import "errors"

var (
	ErrShareChanged            = errors.New("share changed during update")
	ErrAmbiguousShareReference = errors.New("share reference belongs to multiple users")
	ErrShareNotFound           = errors.New("share not found")
	ErrShareAlreadyExists      = errors.New("share already exists")
)
