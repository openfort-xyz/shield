package sharerepo

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
	"go.openfort.xyz/shield/pkg/oflog"
	"gorm.io/gorm"
	"log/slog"
	"os"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.ShareRepository = (*repository)(nil)

func New(db *sql.Client) repositories.ShareRepository {
	return &repository{
		db:     db,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("share_repository"),
		parser: newParser(),
	}

}

func (r *repository) Create(ctx context.Context, shr *share.Share) error {
	r.logger.InfoContext(ctx, "creating share")

	dbShr := r.parser.toDatabase(shr)
	err := r.db.Create(dbShr).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating share", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("user_id", userID))

	var dbShr *Share
	err := r.db.Where("user_id = ?", userID).First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share", slog.String("error", err.Error()))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}
