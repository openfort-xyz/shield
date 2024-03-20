package authentication

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type UserAuthenticator interface {
	Authenticate(ctx context.Context, apiKey, token string, providerType provider.Type) (userID string, err error)
}
