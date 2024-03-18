package usersvc

import "errors"

var (
	ErrUserNotFound              = errors.New("user not found")
	ErrExternalUserAlreadyExists = errors.New("external user already exists")
)
