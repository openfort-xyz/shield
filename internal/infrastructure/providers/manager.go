package providers

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
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
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("provider_manager"),
	}
}

func (p *Manager) GetProvider(ctx context.Context, projectID string, providerType provider.Type) (providers.IdentityProvider, error) {
	p.logger.InfoContext(ctx, "getting provider", slog.String("provider_type", string(providerType)))

	prov, err := p.repo.GetByProjectAndType(ctx, projectID, providerType)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return nil, ErrProviderNotConfigured
		}
		p.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
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
		return newOpenfortProvider(p.config.openfortConfig, config), nil
	case provider.TypeSupabase:
		config, ok := prov.Config.(*provider.SupabaseConfig)
		if !ok {
			return nil, ErrProviderConfigMismatch
		}
		return newSupabaseProvider(p.config.supabaseConfig, config), nil
	default:
		return nil, ErrProviderNotSupported
	}
}
