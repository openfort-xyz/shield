package repositories

import "context"

type EncryptionPartsRepository interface {
	Get(ctx context.Context, sessionId string) (string, error)
	Set(ctx context.Context, sessionId, part string) error
	Delete(ctx context.Context, sessionId string) error
}
