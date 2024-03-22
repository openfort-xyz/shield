package providers

import "context"

type IdentityProvider interface {
	GetProviderID() string
	Identify(ctx context.Context, token string, opts ...CustomOption) (string, error)
}

type CustomOption func(*CustomOptions)
type CustomOptions struct {
	OpenfortProvider  *string
	OpenfortTokenType *string
}

func WithOpenfortProvider(value string) CustomOption {
	return func(opts *CustomOptions) {
		opts.OpenfortProvider = &value
	}
}

func WithOpenfortTokenType(value string) CustomOption {
	return func(opts *CustomOptions) {
		opts.OpenfortTokenType = &value
	}
}
