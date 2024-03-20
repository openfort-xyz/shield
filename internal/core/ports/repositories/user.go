package repositories

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/user"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrExternalUserNotFound = errors.New("external user not found")
)

type UserRepository interface {
	Create(ctx context.Context, user *user.User) error
	Get(ctx context.Context, userID string) (*user.User, error)

	CreateExternal(ctx context.Context, user *user.ExternalUser) error
	FindExternalBy(ctx context.Context, opts ...Option) ([]*user.ExternalUser, error)

	WithUserID(userID string) Option
	WithExternalUserID(externalUserID string) Option
	WithProviderID(providerID string) Option
}

type Option func(Options)

type Options interface{}
