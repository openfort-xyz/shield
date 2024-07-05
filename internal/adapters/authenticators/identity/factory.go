package identity

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/adapters/authenticators/identity/custom_identity"
	"go.openfort.xyz/shield/internal/adapters/authenticators/identity/openfort_identity"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
)

type identityFactory struct {
	config *openfort_identity.Config
	repo   repositories.ProviderRepository
	logger *slog.Logger
}

func NewIdentityFactory(cfg *openfort_identity.Config, repo repositories.ProviderRepository) factories.IdentityFactory {
	return &identityFactory{
		config: cfg,
		repo:   repo,
		logger: logger.New("provider_manager"),
	}
}

func (p *identityFactory) CreateCustomIdentity(ctx context.Context, apiKey string) (factories.Identity, error) {
	prov, err := p.repo.GetByAPIKeyAndType(ctx, apiKey, provider.TypeCustom)
	if err != nil {
		if errors.Is(err, domainErrors.ErrProjectNotFound) {
			return nil, domainErrors.ErrProviderNotConfigured
		}
		p.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return nil, err
	}

	config, ok := prov.Config.(*provider.CustomConfig)
	if !ok {
		return nil, domainErrors.ErrProviderConfigMismatch
	}

	return custom_identity.NewCustomIdentityFactory(config), nil
}

func (p *identityFactory) CreateOpenfortIdentity(ctx context.Context, apiKey string, authenticationProvider, tokenType *string) (factories.Identity, error) {
	prov, err := p.repo.GetByAPIKeyAndType(ctx, apiKey, provider.TypeOpenfort)
	if err != nil {
		if errors.Is(err, domainErrors.ErrProjectNotFound) {
			return nil, domainErrors.ErrProviderNotConfigured
		}
		p.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return nil, err
	}

	config, ok := prov.Config.(*provider.OpenfortConfig)
	if !ok {
		return nil, domainErrors.ErrProviderConfigMismatch
	}

	return openfort_identity.NewOpenfortIdentityFactory(p.config, config, authenticationProvider, tokenType), nil
}
