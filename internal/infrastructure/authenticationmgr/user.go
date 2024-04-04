package authenticationmgr

import (
	"context"
	"errors"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providersmgr"
	"go.openfort.xyz/shield/pkg/logger"
)

type user struct {
	projectRepo     repositories.ProjectRepository
	providerManager *providersmgr.Manager
	userService     services.UserService
	logger          *slog.Logger
}

var _ authentication.UserAuthenticator = (*user)(nil)

func newUserAuthenticator(repository repositories.ProjectRepository, providerManager *providersmgr.Manager, userService services.UserService) authentication.UserAuthenticator {
	return &user{
		projectRepo:     repository,
		providerManager: providerManager,
		userService:     userService,
		logger:          logger.New("api_key_authenticator"),
	}
}

func (a *user) Authenticate(ctx context.Context, apiKey, token string, providerType provider.Type, opts ...authentication.CustomOption) (string, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.projectRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api key", logger.Error(err))
		return "", err
	}

	prov, err := a.providerManager.GetProvider(ctx, proj.ID, providerType)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return "", err
	}

	var opt authentication.CustomOptions
	for _, o := range opts {
		o(&opt)
	}

	var providerCustomOptions []providers.CustomOption
	if opt.OpenfortProvider != nil {
		providerCustomOptions = append(providerCustomOptions, providers.WithOpenfortProvider(*opt.OpenfortProvider))
	}
	if opt.OpenfortTokenType != nil {
		providerCustomOptions = append(providerCustomOptions, providers.WithOpenfortTokenType(*opt.OpenfortTokenType))
	}

	externalUserID, err := prov.Identify(ctx, token, providerCustomOptions...)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to identify user", logger.Error(err))
		return "", err
	}

	usr, err := a.userService.GetByExternal(ctx, externalUserID, prov.GetProviderID())
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) && !errors.Is(err, domain.ErrExternalUserNotFound) {
			a.logger.ErrorContext(ctx, "failed to get user by external", logger.Error(err))
			return "", err
		}
		usr, err = a.userService.Create(ctx, proj.ID)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to create user", logger.Error(err))
			return "", err
		}

		_, err = a.userService.CreateExternal(ctx, proj.ID, usr.ID, externalUserID, prov.GetProviderID())
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to create external user", logger.Error(err))
			return "", err
		}
	}

	return usr.ID, nil
}
