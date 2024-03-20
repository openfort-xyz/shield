package domain

import "errors"

var (
	// Project errors
	ErrProjectNotFound = errors.New("project not found")

	// Provider errors
	ErrNoProviderConfig      = errors.New("no provider config found")
	ErrInvalidProviderConfig = errors.New("invalid provider config")
	ErrUnknownProviderType   = errors.New("unknown provider type")
	ErrProviderAlreadyExists = errors.New("custom authentication already registered for this project")
	ErrProviderNotFound      = errors.New("custom authentication not found")

	// Share errors
	ErrShareNotFound      = errors.New("share not found")
	ErrShareAlreadyExists = errors.New("share already exists")

	// User errors
	ErrUserNotFound              = errors.New("user not found")
	ErrExternalUserNotFound      = errors.New("external user not found")
	ErrExternalUserAlreadyExists = errors.New("external user already exists")
)