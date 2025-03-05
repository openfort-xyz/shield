package repositories

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/keychain"
)

type KeychainRepository interface {
	Create(ctx context.Context, keychain *keychain.Keychain) error
	Get(ctx context.Context, keychainID string) (*keychain.Keychain, error)
	GetByUserID(ctx context.Context, userID string) (*keychain.Keychain, error)
	Delete(ctx context.Context, keychainID string) error
}
