package authentication

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type UserAuthenticator interface {
	Authenticate(ctx context.Context, apiKey, token string, providerType provider.Type, opts ...CustomOption) (*Authentication, error)
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

type Authentication struct {
	UserID    string
	ProjectID string
}
