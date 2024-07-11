package repositories

import "context"

type EncryptionPartsRepository interface {
	Get(ctx context.Context, sessionID string) (string, error)
	Set(ctx context.Context, sessionID, part string) error
	Delete(ctx context.Context, sessionID string) error
}
