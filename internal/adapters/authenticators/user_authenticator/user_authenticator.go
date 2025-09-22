package usrauth

import (
	"context"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/authentication"
	"go.openfort.xyz/shield/internal/core/ports/factories"

	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/logger"
)

type UserAuthenticator struct {
	projectRepo     repositories.ProjectRepository
	userService     services.UserService
	apiKey, token   string
	identityFactory factories.Identity
	logger          *slog.Logger
}

var _ factories.Authenticator = (*UserAuthenticator)(nil)

func NewUserAuthenticator(repository repositories.ProjectRepository, userService services.UserService, apiKey, token string, identityFactory factories.Identity) factories.Authenticator {
	return &UserAuthenticator{
		projectRepo:     repository,
		userService:     userService,
		apiKey:          apiKey,
		token:           token,
		identityFactory: identityFactory,
		logger:          logger.New("api_key_authenticator"),
	}
}

func (a *UserAuthenticator) Authenticate(ctx context.Context) (*authentication.Authentication, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.projectRepo.GetByAPIKey(ctx, a.apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api key", logger.Error(err))
		return nil, err
	}

	externalUserID, err := a.identityFactory.Identify(ctx, a.token)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to identify user", logger.Error(err))
		return nil, err
	}

	usr, err := a.userService.GetOrCreate(ctx, proj.ID, externalUserID, a.identityFactory.GetProviderID())
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get or create user", logger.Error(err))
		return nil, err
	}

	return &authentication.Authentication{
		UserID:         usr.ID,
		ProjectID:      proj.ID,
		ExternalUserID: externalUserID,
	}, nil
}
