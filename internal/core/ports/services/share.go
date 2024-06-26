package services

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/share"
)

type ShareService interface {
	Create(ctx context.Context, share *share.Share, opts ...ShareOption) error
}

type ShareOption func(*ShareOptions)

type ShareOptions struct {
	EncryptionKey *string
}

func WithEncryptionKey(key string) ShareOption {
	return func(o *ShareOptions) {
		o.EncryptionKey = &key
	}
}
