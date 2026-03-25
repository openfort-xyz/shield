package repositories

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/notifications"
)

type NotificationsRepository interface {
	Save(ctx context.Context, project *notifications.Notification) error
}
