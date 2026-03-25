package factories

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/authentication"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

type AuthenticationFactory interface {
	CreateProjectAuthenticator(apiKey, apiSecret string) Authenticator
	CreateUserAuthenticator(proj *project.Project, token string, identityFactory Identity) Authenticator
}

type Authenticator interface {
	Authenticate(ctx context.Context) (*authentication.Authentication, error)
}
