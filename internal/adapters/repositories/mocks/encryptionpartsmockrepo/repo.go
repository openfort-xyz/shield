package encryptionpartsmockrepo

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/buntdb"
)

type MockEncryptionPartsRepository struct {
	mock.Mock
}

var _ repositories.EncryptionPartsRepository = (*MockEncryptionPartsRepository)(nil)

func (m *MockEncryptionPartsRepository) Get(ctx context.Context, sessionID string) (string, error) {
	args := m.Mock.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}

func (m *MockEncryptionPartsRepository) Set(ctx context.Context, sessionID, part string, options *buntdb.SetOptions) error {
	args := m.Mock.Called(ctx, sessionID, part)
	return args.Error(0)
}

func (m *MockEncryptionPartsRepository) Update(ctx context.Context, sessionID, part string, options *buntdb.SetOptions) error {
	args := m.Mock.Called(ctx, sessionID, part)
	return args.Error(0)
}

func (m *MockEncryptionPartsRepository) Delete(ctx context.Context, projectID string) error {
	args := m.Mock.Called(ctx, projectID)
	return args.Error(0)
}
