package notificationsmockrepo

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/notifications"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/stretchr/testify/mock"
)

type MockNotificationsRepository struct {
	mock.Mock
}

var _ repositories.NotificationsRepository = (*MockNotificationsRepository)(nil)

func (m *MockNotificationsRepository) Save(ctx context.Context, notif *notifications.Notification) error {
	args := m.Mock.Called(ctx, notif)
	return args.Error(0)
}
