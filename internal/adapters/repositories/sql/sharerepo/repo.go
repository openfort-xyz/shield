package sharerepo

import (
	"context"
	"errors"
	"log/slog"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/adapters/repositories/sql"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
	"gorm.io/gorm"
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
		logger: logger.New("share_repository"),
		parser: newParser(),
	}
}

func (r *repository) Create(ctx context.Context, shr *share.Share) error {
	r.logger.InfoContext(ctx, "creating share")

	if shr.ID == "" {
		shr.ID = uuid.NewString()
	}

	dbShr := r.parser.toDatabase(shr)
	err := r.db.Create(dbShr).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating share", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*share.Share, error) {
	r.logger.InfoContext(ctx, "getting share", slog.String("user_id", userID))

	dbShr := &Share{}
	err := r.db.Where("user_id = ?", userID).First(dbShr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrShareNotFound
		}
		r.logger.ErrorContext(ctx, "error getting share", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbShr), nil
}

func (r *repository) Delete(ctx context.Context, shareID string) error {
	r.logger.InfoContext(ctx, "deleting share", slog.String("id", shareID))

	err := r.db.Where("id = ?", shareID).Delete(&Share{}).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error deleting share", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) ListDecryptedByProjectID(ctx context.Context, projectID string) ([]*share.Share, error) {
	r.logger.InfoContext(ctx, "listing shares", slog.String("project_id", projectID))

	var dbShares []*Share
	err := r.db.Joins("JOIN shld_users ON shld_shares.user_id = shld_users.id").
		Joins("JOIN shld_projects ON shld_users.project_id = shld_projects.id").
		Where("shld_projects.id = ?", projectID).
		Where("shld_shares.entropy = ?", EntropyNone).
		Find(&dbShares).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error listing shares", logger.Error(err))
		return nil, err
	}

	var shares []*share.Share
	for _, dbShr := range dbShares {
		shares = append(shares, r.parser.toDomain(dbShr))
	}

	return shares, nil
}

func (r *repository) UpdateProjectEncryption(ctx context.Context, shareID string, encrypted string) error {
	r.logger.InfoContext(ctx, "updating share", slog.String("id", shareID))

	err := r.db.Model(&Share{}).Where("id = ?", shareID).Update("data", encrypted).Update("entropy", EntropyProject).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error updating share", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(ctx context.Context, shr *share.Share) error {
	r.logger.InfoContext(ctx, "updating share", slog.String("id", shr.ID))

	dbShr := r.parser.toUpdates(shr)
	err := r.db.Model(&Share{}).Where("id = ?", shr.ID).Updates(dbShr).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error updating share", logger.Error(err))
		return err
	}

	return nil
}
