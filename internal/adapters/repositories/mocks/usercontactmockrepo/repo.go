package usercontactmockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/usercontact"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type MockUserContactRepository struct {
	mock.Mock
}

var _ repositories.UserContactRepository = (*MockUserContactRepository)(nil)

func (m *MockUserContactRepository) Save(ctx context.Context, notif *usercontact.UserContact) error {
	args := m.Mock.Called(ctx, notif)
	return args.Error(0)
}

func (m *MockUserContactRepository) GetByUserID(ctx context.Context, userID string) (*usercontact.UserContact, error) {
	args := m.Mock.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usercontact.UserContact), args.Error(1)
}
