package authenticationmgr

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providersmgr"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
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
		logger:          slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("api_key_authenticator"),
	}
}

func (a *user) Authenticate(ctx context.Context, apiKey, token string, providerType provider.Type, opts ...authentication.CustomOption) (string, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.projectRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api key", slog.String("error", err.Error()))
		return "", err
	}

	prov, err := a.providerManager.GetProvider(ctx, proj.ID, providerType)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return "", err
	}

	opt := make(authentication.CustomOptions)
	for _, o := range opts {
		o(&opt)
	}

	var providerCustomOptions []providers.CustomOption
	if ofProvider, ok := opt[authentication.CustomOptionOpenfortProvider]; ok {
		providerCustomOptions = append(providerCustomOptions, providers.WithCustomOption(providers.CustomOptionOpenfortProvider, ofProvider))
	}
	if ofTokenType, ok := opt[authentication.CustomOptionOpenfortTokenType]; ok {
		providerCustomOptions = append(providerCustomOptions, providers.WithCustomOption(providers.CustomOptionOpenfortTokenType, ofTokenType))
	}

	externalUserID, err := prov.Identify(ctx, token, providerCustomOptions...)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to identify user", slog.String("error", err.Error()))
		return "", err
	}

	usr, err := a.userService.GetByExternal(ctx, externalUserID, prov.GetProviderID())
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) && !errors.Is(err, domain.ErrExternalUserNotFound) {
			a.logger.ErrorContext(ctx, "failed to get user by external", slog.String("error", err.Error()))
			return "", err
		}
		usr, err = a.userService.Create(ctx, proj.ID)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to create user", slog.String("error", err.Error()))
			return "", err
		}

		_, err = a.userService.CreateExternal(ctx, proj.ID, usr.ID, externalUserID, prov.GetProviderID())
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to create external user", slog.String("error", err.Error()))
			return "", err
		}
	}

	return usr.ID, nil
}
