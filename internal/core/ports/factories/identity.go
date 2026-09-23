package factories

import (
	"context"
)

type IdentityFactory interface {
	CreateCustomIdentity(ctx context.Context, projectID string) (Identity, error)
	// sessionCookie is the raw Cookie header of a cookie-session request, empty for bearer requests.
	CreateOpenfortIdentity(ctx context.Context, projectID string, authenticationProvider, tokenType *string, sessionCookie string) (Identity, error)
}

type Identity interface {
	GetProviderID() string
	GetCookieFieldName() string
	Identify(ctx context.Context, token string) (string, error)
}
