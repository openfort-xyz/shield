package repositories

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/share"
)

var (
	ErrShareNotFound = errors.New("share not found")
)

type ShareRepository interface {
	Create(ctx context.Context, shr *share.Share) error
	GetByUserID(ctx context.Context, userID string) (*share.Share, error)
}
