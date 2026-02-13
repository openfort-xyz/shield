package identity

import (
	"context"
	"errors"
	"log/slog"

	cstmidty "go.openfort.xyz/shield/internal/adapters/authenticators/identity/custom_identity"
	ofidty "go.openfort.xyz/shield/internal/adapters/authenticators/identity/openfort_identity"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"

	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
)

type identityFactory struct {
	config *ofidty.Config
	repo   repositories.ProviderRepository
	logger *slog.Logger
}

func NewIdentityFactory(cfg *ofidty.Config, repo repositories.ProviderRepository) factories.IdentityFactory {
	return &identityFactory{
		config: cfg,
		repo:   repo,
		logger: logger.New("provider_manager"),
	}
}

func (p *identityFactory) CreateCustomIdentity(ctx context.Context, projectID string) (factories.Identity, error) {
	prov, err := p.repo.GetByProjectAndType(ctx, projectID, provider.TypeCustom)
	if err != nil {
		if errors.Is(err, domainErrors.ErrProviderNotFound) {
			return nil, domainErrors.ErrProviderNotConfigured
		}
		p.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return nil, err
	}

	config, ok := prov.Config.(*provider.CustomConfig)
	if !ok {
		return nil, domainErrors.ErrProviderConfigMismatch
	}

	return cstmidty.NewCustomIdentityFactory(config), nil
}

func (p *identityFactory) CreateOpenfortIdentity(ctx context.Context, projectID string, authenticationProvider, tokenType *string) (factories.Identity, error) {
	prov, err := p.repo.GetByProjectAndType(ctx, projectID, provider.TypeOpenfort)
	if err != nil {
		if errors.Is(err, domainErrors.ErrProviderNotFound) {
			return nil, domainErrors.ErrProviderNotConfigured
		}
		p.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return nil, err
	}

	config, ok := prov.Config.(*provider.OpenfortConfig)
	if !ok {
		return nil, domainErrors.ErrProviderConfigMismatch
	}

	return ofidty.NewOpenfortIdentityFactory(p.config, config, authenticationProvider, tokenType), nil
}
