package usermockedrepo

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/user"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
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

func (m *MockUserRepository) Create(ctx context.Context, usr *user.User) error {
	args := m.Mock.Called(ctx, usr)
	return args.Error(0)
}

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

func (m *MockUserRepository) CreateExternal(ctx context.Context, extUsr *user.ExternalUser) error {
	args := m.Mock.Called(ctx, extUsr)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserIDsByExternalID(ctx context.Context, externalUserID string) ([]string, error) {
	args := m.Mock.Called(ctx, externalUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
