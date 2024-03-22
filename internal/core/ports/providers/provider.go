package providers

import "context"

type IdentityProvider interface {
	GetProviderID() string
	Identify(ctx context.Context, token string, opts ...CustomOption) (string, error)
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
