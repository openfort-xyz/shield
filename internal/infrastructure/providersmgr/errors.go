package providersmgr

import "errors"

var (
	ErrProviderNotSupported     = errors.New("provider not supported")
	ErrProviderNotConfigured    = errors.New("provider not configured")
	ErrProviderConfigMismatch   = errors.New("provider config mismatch")
	ErrInvalidToken             = errors.New("invalid token")
	ErrMissingOpenfortProvider  = errors.New("missing openfort provider")
	ErrMissingOpenfortTokenType = errors.New("missing openfort token type")
	ErrUnexpectedStatusCode     = errors.New("unexpected status code")
	ErrCertTypeNotSupported     = errors.New("certificate type not supported")
	ErrProviderMisconfigured    = errors.New("provider misconfigured")
)
