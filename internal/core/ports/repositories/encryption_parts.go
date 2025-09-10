package repositories

import "context"

type EncryptionPartsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Update(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}
