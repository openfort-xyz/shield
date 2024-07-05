package errors

import "errors"

var (
	ErrInvalidProviderConfig  = errors.New("invalid provider config")
	ErrUnknownProviderType    = errors.New("unknown provider type")
	ErrProviderAlreadyExists  = errors.New("custom authentication already registered for this project")
	ErrProviderNotFound       = errors.New("custom authentication not found")
	ErrProviderNotConfigured  = errors.New("provider not configured")
	ErrProviderConfigMismatch = errors.New("provider config mismatch")
	ErrUnexpectedStatusCode   = errors.New("unexpected status code")
	ErrCertTypeNotSupported   = errors.New("certificate type not supported")
	ErrProviderMisconfigured  = errors.New("provider misconfigured")
)
