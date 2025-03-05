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

func (m *MockShareRepository) Get(ctx context.Context, shareID string) (*share.Share, error) {
	args := m.Mock.Called(ctx, shareID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*share.Share), args.Error(1)
}

func (m *MockShareRepository) GetByReference(ctx context.Context, reference, keychainID string) (*share.Share, error) {
	args := m.Mock.Called(ctx, reference, keychainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*share.Share), args.Error(1)
}

func (m *MockShareRepository) ListByKeychainID(ctx context.Context, keychainID string) ([]*share.Share, error) {
	args := m.Mock.Called(ctx, keychainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*share.Share), args.Error(1)
}

func (m *MockShareRepository) Delete(ctx context.Context, shareID string) error {
	args := m.Mock.Called(ctx, shareID)
	return args.Error(0)
}

func (m *MockShareRepository) ListProjectIDAndEntropy(ctx context.Context, projectID string, entropy share.Entropy) ([]*share.Share, error) {
	args := m.Mock.Called(ctx, projectID, entropy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*share.Share), args.Error(1)
}

func (m *MockShareRepository) UpdateProjectEncryption(ctx context.Context, shareID string, encrypted string) error {
	args := m.Mock.Called(ctx, shareID, encrypted)
	return args.Error(0)
}

func (m *MockShareRepository) Update(ctx context.Context, shr *share.Share) error {
	args := m.Mock.Called(ctx, shr)
	return args.Error(0)
}

func (m *MockShareRepository) BulkUpdate(ctx context.Context, shrs []*share.Share) error {
	args := m.Mock.Called(ctx, shrs)
	return args.Error(0)
}
