package errors

import "errors"

var (
	ErrInvalidProviderConfig = errors.New("invalid provider config")
	ErrUnknownProviderType   = errors.New("unknown provider type")
	ErrProviderAlreadyExists = errors.New("custom authentication already registered for this project")
	ErrProviderNotFound      = errors.New("custom authentication not found")
)
