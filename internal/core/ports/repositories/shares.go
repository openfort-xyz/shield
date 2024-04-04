package repositories

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/share"
)

type ShareRepository interface {
	Create(ctx context.Context, shr *share.Share) error
	GetByUserID(ctx context.Context, userID string) (*share.Share, error)
	ListDecryptedByProjectID(ctx context.Context, projectID string) ([]*share.Share, error)
	Update(ctx context.Context, shr *share.Share) error
}
