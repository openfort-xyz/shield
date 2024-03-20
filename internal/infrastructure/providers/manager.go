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

	switch providerType {
	case provider.TypeCustom:
		config, err := p.repo.GetCustom(ctx, projectID)
		if err != nil {
			if errors.Is(err, domain.ErrProviderNotFound) {
				return nil, ErrProviderNotConfigured
			}
			p.logger.ErrorContext(ctx, "failed to get custom provider", slog.String("error", err.Error()))
			return nil, err
		}
		return newCustomProvider(config), nil
	case provider.TypeOpenfort:
		config, err := p.repo.GetOpenfort(ctx, projectID)
		if err != nil {
			if errors.Is(err, domain.ErrProviderNotFound) {
				return nil, ErrProviderNotConfigured
			}
			p.logger.ErrorContext(ctx, "failed to get openfort provider", slog.String("error", err.Error()))
			return nil, err
		}
		return newOpenfortProvider(p.config.openfortConfig, config), nil
	case provider.TypeSupabase:
		config, err := p.repo.GetSupabase(ctx, projectID)
		if err != nil {
			if errors.Is(err, domain.ErrProviderNotFound) {
				return nil, ErrProviderNotConfigured
			}
			p.logger.ErrorContext(ctx, "failed to get supabase provider", slog.String("error", err.Error()))
			return nil, err
		}
		return newSupabaseProvider(p.config.supabaseConfig, config), nil
	default:
		return nil, ErrProviderNotSupported
	}
}