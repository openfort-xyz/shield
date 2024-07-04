package factories

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/authentication"
)

type AuthenticationFactory interface {
	CreateProjectAuthenticator(apiKey, apiSecret string) Authenticator
	CreateUserAuthenticator(apiKey, token string, identityFactory Identity) Authenticator
}

type Authenticator interface {
	Authenticate(ctx context.Context) (*authentication.Authentication, error)
}
