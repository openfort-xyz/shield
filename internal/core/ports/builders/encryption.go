package builders

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

type EncryptionKeyBuilder interface {
	SetProjectPart(ctx context.Context, identifier string) error
	SetDatabasePart(ctx context.Context, identifier string) error

	GetProjectPart(ctx context.Context) project.ProjectPart
	GetDatabasePart(ctx context.Context) project.StoredPart

	// Build reconstructs the key and, when the project stores a key check,
	// rejects a key that does not match it with ErrInvalidEncryptionPart.
	Build(ctx context.Context) (string, error)
}
