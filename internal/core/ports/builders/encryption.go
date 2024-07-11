package builders

import (
	"context"
)

type EncryptionKeyBuilder interface {
	SetProjectPart(ctx context.Context, identifier string) error
	SetDatabasePart(ctx context.Context, identifier string) error
	Build(ctx context.Context) (string, error)
}
