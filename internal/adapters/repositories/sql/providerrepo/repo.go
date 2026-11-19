package providerrepo

import (
	"context"
	"errors"
	"log/slog"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/projectrepo"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/pkg/logger"
	"gorm.io/gorm"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.ProviderRepository = (*repository)(nil)

func New(db *sql.Client) repositories.ProviderRepository {
	return &repository{
		db:     db,
		logger: logger.New("provider_repository"),
		parser: newParser(),
	}
}

// Create inserts the providers and their configs in one transaction, after
// check passes against the project row. Creates for one project run one at a
// time, so of concurrent creates of a provider type one wins and the others
// get ErrProviderAlreadyExists.
func (r *repository) Create(ctx context.Context, projectID string, provs []*provider.Provider, check project.CheckAPISecretFunc) error {
	r.logger.InfoContext(ctx, "creating providers", slog.String("project_id", projectID))

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := projectrepo.CheckCredentialsSerialized(tx, projectID, check); err != nil {
			return err
		}
		for _, prov := range provs {
			prov.ProjectID = projectID
			if err := r.create(tx, prov); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating providers", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) create(tx *gorm.DB, prov *provider.Provider) error {
	var existing int64
	err := tx.Model(&Provider{}).Where("project_id = ? AND type = ?", prov.ProjectID, r.parser.mapProviderTypeToDatabase[prov.Type]).Count(&existing).Error
	if err != nil {
		return err
	}
	if existing > 0 {
		return domainErrors.ErrProviderAlreadyExists
	}

	if prov.ID == "" {
		prov.ID = uuid.NewString()
	}
	if err := tx.Create(r.parser.toDatabaseProvider(prov)).Error; err != nil {
		return err
	}
	switch cfg := prov.Config.(type) {
	case *provider.CustomConfig:
		cfg.ProviderID = prov.ID
		return tx.Create(r.parser.toDatabaseCustomProvider(cfg)).Error
	case *provider.OpenfortConfig:
		cfg.ProviderID = prov.ID
		return tx.Create(r.parser.toDatabaseOpenfortProvider(cfg)).Error
	default:
		return domainErrors.ErrInvalidProviderConfig
	}
}

// checked runs write in a transaction that first passes check against the
// project row held FOR SHARE, so a credential commit or disable either waits
// for the write or lands first and refuses it.
func (r *repository) checked(ctx context.Context, projectID string, check project.CheckAPISecretFunc, write func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := projectrepo.CheckCredentials(tx, projectID, check); err != nil {
			return err
		}
		return write(tx)
	})
}

func (r *repository) GetByProjectAndType(ctx context.Context, projectID string, providerType provider.Type) (*provider.Provider, error) {
	r.logger.InfoContext(ctx, "getting provider", slog.String("project_id", projectID), slog.String("provider_type", providerType.String()))

	dbProv := Provider{}
	err := r.db.Preload("Custom").Preload("Openfort").Where("project_id = ? AND type = ?", projectID, r.parser.mapProviderTypeToDatabase[providerType]).First(&dbProv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrProviderNotFound
		}
		r.logger.ErrorContext(ctx, "error getting provider", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomainProvider(dbProv), nil
}

func (r *repository) Get(ctx context.Context, id string) (*provider.Provider, error) {
	r.logger.InfoContext(ctx, "getting provider", slog.String("provider_id", id))

	dbProv := Provider{}
	err := r.db.Preload("Custom").Preload("Openfort").Where("id = ?", id).First(&dbProv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrProviderNotFound
		}
		r.logger.ErrorContext(ctx, "error getting provider", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomainProvider(dbProv), nil
}

func (r *repository) List(ctx context.Context, projectID string) ([]*provider.Provider, error) {
	r.logger.InfoContext(ctx, "listing providers", slog.String("project_id", projectID))

	var dbProvs []Provider
	err := r.db.Where("project_id = ?", projectID).Find(&dbProvs).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error listing providers", logger.Error(err))
		return nil, err
	}

	var provs []*provider.Provider
	for _, dbProv := range dbProvs {
		provs = append(provs, r.parser.toDomainProvider(dbProv))
	}

	return provs, nil
}

func (r *repository) Delete(ctx context.Context, projectID, providerID string, check project.CheckAPISecretFunc) error {
	r.logger.InfoContext(ctx, "deleting provider", slog.String("provider_id", providerID))

	err := r.checked(ctx, projectID, check, func(tx *gorm.DB) error {
		cmd := tx.Where("project_id = ?", projectID).Delete(&Provider{ID: providerID})
		if cmd.Error != nil {
			return cmd.Error
		}
		if cmd.RowsAffected == 0 {
			return domainErrors.ErrProviderNotFound
		}
		return nil
	})
	if err != nil && !errors.Is(err, domainErrors.ErrProviderNotFound) {
		r.logger.ErrorContext(ctx, "error deleting provider", logger.Error(err))
	}
	return err
}

func (r *repository) GetCustom(ctx context.Context, providerID string) (*provider.CustomConfig, error) {
	r.logger.InfoContext(ctx, "getting custom provider", slog.String("provider_id", providerID))

	dbProv := &ProviderCustom{}
	err := r.db.Where("provider_id = ?", providerID).First(dbProv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrProviderNotFound
		}
		r.logger.ErrorContext(ctx, "error getting custom provider", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomainCustomProvider(dbProv), nil
}

func (r *repository) UpdateCustom(ctx context.Context, projectID string, prov *provider.CustomConfig, check project.CheckAPISecretFunc) error {
	r.logger.InfoContext(ctx, "updating custom provider", slog.String("provider_id", prov.ProviderID))

	updates := r.parser.toUpdateCustomProviderMap(prov)
	err := r.checked(ctx, projectID, check, func(tx *gorm.DB) error {
		return tx.Model(&ProviderCustom{}).Where("provider_id = ?", prov.ProviderID).Updates(updates).Error
	})
	if err != nil {
		r.logger.ErrorContext(ctx, "error updating custom provider", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetOpenfort(ctx context.Context, providerID string) (*provider.OpenfortConfig, error) {
	r.logger.InfoContext(ctx, "getting openfort provider", slog.String("provider_id", providerID))

	dbProv := &ProviderOpenfort{}
	err := r.db.Where("provider_id = ?", providerID).First(dbProv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrProviderNotFound
		}
		r.logger.ErrorContext(ctx, "error getting openfort provider", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomainOpenfortProvider(dbProv), nil
}

func (r *repository) UpdateOpenfort(ctx context.Context, projectID string, prov *provider.OpenfortConfig, check project.CheckAPISecretFunc) error {
	r.logger.InfoContext(ctx, "updating openfort provider", slog.String("provider_id", prov.ProviderID))

	dbProv := r.parser.toDatabaseOpenfortProvider(prov)
	err := r.checked(ctx, projectID, check, func(tx *gorm.DB) error {
		return tx.Save(dbProv).Error
	})
	if err != nil {
		r.logger.ErrorContext(ctx, "error updating openfort provider", logger.Error(err))
		return err
	}

	return nil
}
