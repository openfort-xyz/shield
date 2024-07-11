package factories

import (
	"context"
)

type IdentityFactory interface {
	CreateCustomIdentity(ctx context.Context, apiKey string) (Identity, error)
	CreateOpenfortIdentity(ctx context.Context, apiKey string, authenticationProvider, tokenType *string) (Identity, error)
}

type Identity interface {
	GetProviderID() string
	Identify(ctx context.Context, token string) (string, error)
}
