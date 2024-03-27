package services

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/share"
)

type ShareService interface {
	Create(ctx context.Context, share *share.Share) error
	GetByUserID(ctx context.Context, userID string) (*share.Share, error)
}
