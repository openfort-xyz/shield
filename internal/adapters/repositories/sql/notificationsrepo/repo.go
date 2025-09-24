package notificationsrepo

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/notifications"

	"log/slog"

	"go.openfort.xyz/shield/internal/adapters/repositories/sql"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.NotificationsRepository = &repository{}

func New(db *sql.Client) repositories.NotificationsRepository {
	return &repository{
		db:     db,
		logger: logger.New("notifications_repository"),
		parser: newParser(),
	}
}

func (r *repository) Save(ctx context.Context, notification *notifications.Notification) error {
	r.logger.InfoContext(ctx, "saving notifications", slog.String("project", notification.ProjectID))

	dbNotif := r.parser.toDatabase(notification)
	err := r.db.Create(dbNotif).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error saving notification", logger.Error(err))
		return err
	}

	return nil
}
