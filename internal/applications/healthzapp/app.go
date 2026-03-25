package healthzapp

import (
	"context"
	"log/slog"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/pkg/logger"
)

type Application struct {
	db     *sql.Client
	logger *slog.Logger
}

func New(db *sql.Client) *Application {
	return &Application{
		db:     db,
		logger: logger.New("health_application"),
	}
}

func (a *Application) Healthz(ctx context.Context) error {
	db, err := a.db.DB.DB()
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get database connection", logger.Error(err))
		return errors.ErrDatabaseUnavailable
	}

	if err = db.PingContext(ctx); err != nil {
		a.logger.ErrorContext(ctx, "failed to ping database", logger.Error(err))
		return errors.ErrDatabaseUnavailable
	}

	return nil
}
