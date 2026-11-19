package userrepo

import (
	"context"
	stdsql "database/sql"
	"errors"
	"log/slog"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/core/domain/user"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/pkg/logger"
	"gorm.io/gorm"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.UserRepository = (*repository)(nil)

func New(db *sql.Client) repositories.UserRepository {
	return &repository{
		db:     db,
		logger: logger.New("user_repository"),
		parser: newParser(),
	}
}

func (r *repository) Get(ctx context.Context, userID string) (*user.User, error) {
	r.logger.InfoContext(ctx, "getting user", slog.String("user_id", userID))

	dbUsr := &User{}
	err := r.db.Where("id = ?", userID).First(dbUsr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrUserNotFound
		}
		r.logger.ErrorContext(ctx, "error getting user", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbUsr), nil
}

func (r *repository) FindExternalBy(ctx context.Context, opts ...repositories.Option) ([]*user.ExternalUser, error) {
	r.logger.InfoContext(ctx, "finding external user")

	options := &options{
		query: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(options)
	}

	var dbExtUsrs []ExternalUser
	err := r.db.Where(options.query).Find(&dbExtUsrs).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []*user.ExternalUser{}, nil
		}
		r.logger.ErrorContext(ctx, "error finding external user", logger.Error(err))
		return nil, err
	}

	extUsrs := make([]*user.ExternalUser, len(dbExtUsrs))
	for i, dbExtUsr := range dbExtUsrs {
		extUsrs[i] = r.parser.toDomainExternalUser(dbExtUsr)
	}

	return extUsrs, nil
}

func (r *repository) GetOrCreate(ctx context.Context, projectID, externalUserID, providerID string) (*user.User, error) {
	r.logger.InfoContext(ctx, "getting or creating user by external user ID and provider ID")

	dbUsr := &User{}
	// Read committed so the lookup after the lock sees a row committed while waiting.
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owned int64
		err := tx.Table("shld_providers").
			Where("id = ? AND project_id = ? AND deleted_at IS NULL", providerID, projectID).
			Count(&owned).Error
		if err != nil {
			return err
		}
		if owned == 0 {
			return domainErrors.ErrProviderNotFound
		}

		// Serializes first resolution of one identity without a unique index, which
		// existing duplicate rows would prevent from being created.
		err = tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))", providerID, externalUserID).Error
		if err != nil {
			return err
		}

		err = tx.
			Joins("JOIN shld_external_users ON shld_users.id = shld_external_users.user_id").
			Where("shld_users.project_id = ? AND shld_external_users.external_user_id = ? AND shld_external_users.provider_id = ?",
				projectID, externalUserID, providerID).
			First(dbUsr).Error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		dbUsr = &User{ID: uuid.NewString(), ProjectID: projectID}
		if err := tx.Create(dbUsr).Error; err != nil {
			return err
		}
		return tx.Create(&ExternalUser{
			ID:             uuid.NewString(),
			UserID:         dbUsr.ID,
			ExternalUserID: externalUserID,
			ProviderID:     providerID,
		}).Error
	}, &stdsql.TxOptions{Isolation: stdsql.LevelReadCommitted})
	if err != nil {
		if !errors.Is(err, domainErrors.ErrProviderNotFound) {
			r.logger.ErrorContext(ctx, "error getting or creating user", logger.Error(err))
		}
		return nil, err
	}

	return r.parser.toDomain(dbUsr), nil
}
