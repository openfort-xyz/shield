package errors

import "errors"

var (
	ErrUserNotFound              = errors.New("user not found")
	ErrExternalUserNotFound      = errors.New("external user not found")
	ErrExternalUserAlreadyExists = errors.New("external user already exists")
)
