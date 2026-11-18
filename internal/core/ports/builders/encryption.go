package builders

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

type EncryptionKeyBuilder interface {
	SetProjectPart(ctx context.Context, identifier string) error
	SetDatabasePart(ctx context.Context, identifier string) error

	GetDatabasePart(ctx context.Context) project.StoredPart
	// GetKeyCheck returns the stored key check Build verifies against, nil
	// for a key created before key checks existed.
	GetKeyCheck(ctx context.Context) *project.KeyCheck

	// Build reconstructs the key and, when the project stores a key check,
	// rejects a key that does not match it with ErrInvalidEncryptionPart.
	Build(ctx context.Context) (string, error)
}
