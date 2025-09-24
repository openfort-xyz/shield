package notificationsmockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/notifications"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type MockNotificationsRepository struct {
	mock.Mock
}

var _ repositories.NotificationsRepository = (*MockNotificationsRepository)(nil)

func (m *MockNotificationsRepository) Save(ctx context.Context, notif *notifications.Notification) error {
	args := m.Mock.Called(ctx, notif)
	return args.Error(0)
}
