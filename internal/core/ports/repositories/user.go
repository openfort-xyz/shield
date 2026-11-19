package repositories

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/user"
)

type UserRepository interface {
	Get(ctx context.Context, userID string) (*user.User, error)

	FindExternalBy(ctx context.Context, opts ...Option) ([]*user.ExternalUser, error)

	WithUserID(userID string) Option
	WithExternalUserID(externalUserID string) Option
	WithProviderID(providerID string) Option

	// GetOrCreate resolves the project's user for an external identity, creating it on first
	// sight. It returns ErrProviderNotFound when the provider is not the project's.
	GetOrCreate(ctx context.Context, projectID, externalUserID, providerID string) (*user.User, error)
}

type Option func(Options)

type Options interface{}
