package repositories

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/share"
)

type ShareRepository interface {
	Create(ctx context.Context, shr *share.Share) error
	Get(ctx context.Context, shareID string) (*share.Share, error)
	GetByUserID(ctx context.Context, userID string) (*share.Share, error)
	GetByReference(ctx context.Context, reference, keychainID string) (*share.Share, error)
	Delete(ctx context.Context, shareID string) error
	ListByKeychainID(ctx context.Context, keychainID string) ([]*share.Share, error)
	ListProjectIDAndEntropy(ctx context.Context, projectID string, entropy share.Entropy) ([]*share.Share, error)
	UpdateProjectEncryption(ctx context.Context, shareID string, encrypted string) error
	Update(ctx context.Context, shr *share.Share) error
	BulkUpdate(ctx context.Context, shrs []*share.Share) error
	GetShareStorageMethods(ctx context.Context) ([]*share.StorageMethod, error)
}
