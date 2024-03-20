package providers

import "context"

type IdentityProvider interface {
	GetProviderID() string
	Identify(ctx context.Context, token string) (string, error)
}
