package providers

import "errors"

var (
	ErrProviderNotSupported   = errors.New("provider not supported")
	ErrProviderNotConfigured  = errors.New("provider not configured")
	ErrProviderConfigMismatch = errors.New("provider config mismatch")
	ErrFailedToGetJWSs        = errors.New("failed to get JWSs")
)
