package builders

import (
	"context"
)

type EncryptionKeyBuilder interface {
	SetProjectPart(ctx context.Context, identifier string) error
	SetDatabasePart(ctx context.Context, identifier string) error

	GetProjectPart(ctx context.Context) string
	GetDatabasePart(ctx context.Context) string

	Build(ctx context.Context) (string, error)
}
