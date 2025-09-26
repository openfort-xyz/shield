package errors

import "errors"

var (
	ErrUserNotFound              = errors.New("user not found")
	ErrExternalUserNotFound      = errors.New("external user not found")
	ErrExternalUserAlreadyExists = errors.New("external user already exists")
	ErrUserContactNotFound       = errors.New("user contact information not found")
)
