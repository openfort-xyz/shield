package factories

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/authentication"
)

type IdentityFactory interface {
	CreateCustomIdentity(ctx context.Context, projectID string) (Identity, error)
	// sessionCookie is the raw Cookie header of a cookie-session request, empty for bearer requests.
	CreateOpenfortIdentity(ctx context.Context, projectID string, authenticationProvider, tokenType *string, sessionCookie string) (Identity, error)
}

type Identity interface {
	GetProviderID() string
	GetCookieFieldName() string
	Identify(ctx context.Context, token string) (authentication.Identity, error)
}
