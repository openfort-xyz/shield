package providersvc

import "errors"

var (
	ErrNoProviderConfig      = errors.New("no provider config found")
	ErrInvalidProviderConfig = errors.New("invalid provider config")
	ErrUnknownProviderType   = errors.New("unknown provider type")
	ErrProviderAlreadyExists = errors.New("custom authentication already registered for this project")
)
