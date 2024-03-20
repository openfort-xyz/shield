package authentication

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providers"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type user struct {
	projectRepo     repositories.ProjectRepository
	providerManager *providers.Manager
	userService     services.UserService
	logger          *slog.Logger
}

var _ authentication.UserAuthenticator = (*user)(nil)

func newUserAuthenticator(repository repositories.ProjectRepository, providerManager *providers.Manager, userService services.UserService) authentication.UserAuthenticator {
	return &user{
		projectRepo:     repository,
		providerManager: providerManager,
		userService:     userService,
		logger:          slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("api_key_authenticator"),
	}
}

func (a *user) Authenticate(ctx context.Context, apiKey string, token string, providerType provider.Type) (string, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.projectRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api key", slog.String("error", err.Error()))
		return "", err
	}

	prov, err := a.providerManager.GetProvider(ctx, proj.ID, providerType)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
	}

	externalUserID, err := prov.Identify(ctx, token)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to identify user", slog.String("error", err.Error()))
		return "", err
	}

	usr, err := a.userService.GetByExternal(ctx, externalUserID, prov.GetProviderID())
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get user by external", slog.String("error", err.Error()))
		return "", err
	}

	return usr.ID, nil
}
