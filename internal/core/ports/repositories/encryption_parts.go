package repositories

import (
	"context"

	"github.com/tidwall/buntdb"
)

type EncryptionPartsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, options *buntdb.SetOptions) error
	Update(ctx context.Context, key, value string, options *buntdb.SetOptions) error
	Delete(ctx context.Context, key string) error
}
