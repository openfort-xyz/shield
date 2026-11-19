package repositories

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/keychain"
)

type KeychainRepository interface {
	// Create stores keychain as its user's keychain. If the user already has
	// one, nothing is stored and keychain.ID is set to the existing one, so
	// concurrent first requests for a user end with a single keychain.
	Create(ctx context.Context, keychain *keychain.Keychain) error
	Get(ctx context.Context, keychainID string) (*keychain.Keychain, error)
	GetByUserID(ctx context.Context, userID string) (*keychain.Keychain, error)
	Delete(ctx context.Context, keychainID string) error
}
