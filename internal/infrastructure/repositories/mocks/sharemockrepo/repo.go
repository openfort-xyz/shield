package sharemockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type MockShareRepository struct {
	mock.Mock
}

var _ repositories.ShareRepository = (*MockShareRepository)(nil)

func (m *MockShareRepository) Create(ctx context.Context, shr *share.Share) error {
	args := m.Mock.Called(ctx, shr)
	return args.Error(0)
}

func (m *MockShareRepository) GetByUserID(ctx context.Context, userID string) (*share.Share, error) {
	args := m.Mock.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*share.Share), args.Error(1)
}
