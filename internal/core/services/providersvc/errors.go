package providersvc

import "errors"

var (
	ErrNoProviderConfig              = errors.New("no provider config found")
	ErrInvalidProviderConfig         = errors.New("invalid provider config")
	ErrUnknownProviderType           = errors.New("unknown provider type")
	ErrCustomProviderAlreadyExists   = errors.New("custom authentication already registered for this project")
	ErrOpenfortProviderAlreadyExists = errors.New("openfort already registered for this project")
	ErrSupabaseProviderAlreadyExists = errors.New("supabase already registered for this project")
)
