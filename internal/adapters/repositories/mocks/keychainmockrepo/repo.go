package keychainmockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/keychain"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type MockKeychainRepository struct {
	mock.Mock
}

var _ repositories.KeychainRepository = (*MockKeychainRepository)(nil)

func (m *MockKeychainRepository) Create(ctx context.Context, keychain *keychain.Keychain) error {
	args := m.Called(ctx, keychain)
	return args.Error(0)
}

func (m *MockKeychainRepository) Get(ctx context.Context, keychainID string) (*keychain.Keychain, error) {
	args := m.Called(ctx, keychainID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*keychain.Keychain), args.Error(1)
}

func (m *MockKeychainRepository) GetByUserID(ctx context.Context, userID string) (*keychain.Keychain, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*keychain.Keychain), args.Error(1)
}

func (m *MockKeychainRepository) Delete(ctx context.Context, keychainID string) error {
	args := m.Called(ctx, keychainID)
	return args.Error(0)
}
