package authentication

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type UserAuthenticator interface {
	Authenticate(ctx context.Context, apiKey, token string, providerType provider.Type, opts ...CustomOption) (userID string, err error)
}

type CustomOption func(*CustomOptions)
type CustomOptions map[string]interface{}

func WithCustomOption(key string, value interface{}) CustomOption {
	return func(o *CustomOptions) {
		(*o)[key] = value
	}
}

const CustomOptionOpenfortProvider = "openfort_provider"
const CustomOptionOpenfortTokenType = "openfort_token_type"
