package factories

import (
	"context"
)

type IdentityFactory interface {
	CreateCustomIdentity(ctx context.Context, projectID string) (Identity, error)
	CreateOpenfortIdentity(ctx context.Context, projectID string, authenticationProvider, tokenType *string) (Identity, error)
}

type Identity interface {
	GetProviderID() string
	GetCookieFieldName() string
	Identify(ctx context.Context, token string) (string, error)
}
