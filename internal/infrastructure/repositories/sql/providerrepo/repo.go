package providerrepo

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
	"go.openfort.xyz/shield/pkg/logger"
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

func (r *repository) Create(ctx context.Context, prov *provider.Provider) error {
	r.logger.InfoContext(ctx, "creating provider")

	if prov.ID == "" {
		prov.ID = uuid.NewString()
	}

	dbProv := r.parser.toDatabaseProvider(prov)
	err := r.db.Create(dbProv).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating provider", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetByProjectAndType(ctx context.Context, projectID string, providerType provider.Type) (*provider.Provider, error) {
	r.logger.InfoContext(ctx, "getting provider", slog.String("project_id", projectID), slog.String("provider_type", providerType.String()))

	dbProv := Provider{}
	err := r.db.Preload("Custom").Preload("Openfort").Where("project_id = ? AND type = ?", projectID, r.parser.mapProviderTypeToDatabase[providerType]).First(&dbProv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProviderNotFound
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
			return nil, domain.ErrProviderNotFound
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

func (r *repository) Delete(ctx context.Context, providerID string) error {
	r.logger.InfoContext(ctx, "deleting provider", slog.String("provider_id", providerID))

	cmd := r.db.Delete(&Provider{ID: providerID})
	if cmd.Error != nil {
		r.logger.ErrorContext(ctx, "error deleting provider", logger.Error(cmd.Error))
		return cmd.Error
	}

	if cmd.RowsAffected == 0 {
		return domain.ErrProviderNotFound
	}

	return nil
}

func (r *repository) CreateCustom(ctx context.Context, prov *provider.CustomConfig) error {
	r.logger.InfoContext(ctx, "creating custom provider", slog.String("provider_id", prov.ProviderID))

	dbProv := r.parser.toDatabaseCustomProvider(prov)
	err := r.db.Create(dbProv).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating custom provider", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetCustom(ctx context.Context, providerID string) (*provider.CustomConfig, error) {
	r.logger.InfoContext(ctx, "getting custom provider", slog.String("provider_id", providerID))

	dbProv := &ProviderCustom{}
	err := r.db.Where("provider_id = ?", providerID).First(dbProv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProviderNotFound
		}
		r.logger.ErrorContext(ctx, "error getting custom provider", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomainCustomProvider(dbProv), nil
}

func (r *repository) UpdateCustom(ctx context.Context, prov *provider.CustomConfig) error {
	r.logger.InfoContext(ctx, "updating custom provider", slog.String("provider_id", prov.ProviderID))

	updates := r.parser.toUpdateCustomProviderMap(prov)
	err := r.db.Model(&ProviderCustom{}).Where("provider_id = ?", prov.ProviderID).Updates(updates).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error updating custom provider", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) CreateOpenfort(ctx context.Context, prov *provider.OpenfortConfig) error {
	r.logger.InfoContext(ctx, "creating openfort provider", slog.String("provider_id", prov.ProviderID))

	dbProv := r.parser.toDatabaseOpenfortProvider(prov)
	err := r.db.Create(dbProv).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating openfort provider", logger.Error(err))
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
			return nil, domain.ErrProviderNotFound
		}
		r.logger.ErrorContext(ctx, "error getting openfort provider", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomainOpenfortProvider(dbProv), nil
}

func (r *repository) UpdateOpenfort(ctx context.Context, prov *provider.OpenfortConfig) error {
	r.logger.InfoContext(ctx, "updating openfort provider", slog.String("provider_id", prov.ProviderID))

	dbProv := r.parser.toDatabaseOpenfortProvider(prov)
	err := r.db.Save(dbProv).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error updating openfort provider", logger.Error(err))
		return err
	}

	return nil
}
