package errors

import "errors"

var (
	ErrShareNotFound      = errors.New("share not found")
	ErrShareAlreadyExists = errors.New("share already exists")
)
