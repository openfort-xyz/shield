package usermockedrepo

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/user"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

var _ repositories.Options = (*MockUserRepository)(nil)

func (m *MockUserRepository) WithUserID(_ string) repositories.Option {
	return func(_ repositories.Options) {}
}

func (m *MockUserRepository) WithExternalUserID(_ string) repositories.Option {
	return func(_ repositories.Options) {}
}

func (m *MockUserRepository) WithProviderID(_ string) repositories.Option {
	return func(_ repositories.Options) {}
}

var _ repositories.UserRepository = (*MockUserRepository)(nil)

func (m *MockUserRepository) Get(ctx context.Context, userID string) (*user.User, error) {
	args := m.Mock.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) FindExternalBy(ctx context.Context, opts ...repositories.Option) ([]*user.ExternalUser, error) {
	args := m.Mock.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.ExternalUser), args.Error(1)
}

func (m *MockUserRepository) GetOrCreate(ctx context.Context, projectID, externalUserID, providerID string) (*user.User, error) {
	args := m.Mock.Called(ctx, projectID, externalUserID, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}
