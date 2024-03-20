package providers

import "errors"

var (
	ErrProviderNotSupported  = errors.New("provider not supported")
	ErrProviderNotConfigured = errors.New("provider not configured")
	ErrFailedToGetJWSs       = errors.New("failed to get JWSs")
)
