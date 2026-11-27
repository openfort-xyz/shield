package errors

import "errors"

var (
	ErrShareChanged       = errors.New("share changed during update")
	ErrShareNotFound      = errors.New("share not found")
	ErrShareAlreadyExists = errors.New("share already exists")
)
