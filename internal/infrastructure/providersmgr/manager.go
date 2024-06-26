package providersmgr

import (
	"context"
	"errors"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
)

type Manager struct {
	config *Config
	repo   repositories.ProviderRepository
	logger *slog.Logger
}

func NewManager(cfg *Config, repo repositories.ProviderRepository) *Manager {
	return &Manager{
		config: cfg,
		repo:   repo,
		logger: logger.New("provider_manager"),
	}
}

func (p *Manager) GetProvider(ctx context.Context, projectID string, providerType provider.Type) (providers.IdentityProvider, error) {
	p.logger.InfoContext(ctx, "getting provider", slog.String("provider_type", string(providerType)))

	prov, err := p.repo.GetByProjectAndType(ctx, projectID, providerType)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return nil, ErrProviderNotConfigured
		}
		p.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return nil, err
	}

	switch prov.Type {
	case provider.TypeCustom:
		config, ok := prov.Config.(*provider.CustomConfig)
		if !ok {
			return nil, ErrProviderConfigMismatch
		}
		return newCustomProvider(config), nil
	case provider.TypeOpenfort:
		config, ok := prov.Config.(*provider.OpenfortConfig)
		if !ok {
			return nil, ErrProviderConfigMismatch
		}
		return newOpenfortProvider(p.config, config), nil
	default:
		return nil, ErrProviderNotSupported
	}
}
