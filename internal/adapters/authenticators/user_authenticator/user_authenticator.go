package usrauth

import (
	"context"
	"log/slog"

	"github.com/openfort-xyz/shield/internal/core/domain/authentication"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"

	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/logger"
)

type UserAuthenticator struct {
	userService     services.UserService
	project         *project.Project
	token           string
	identityFactory factories.Identity
	logger          *slog.Logger
}

var _ factories.Authenticator = (*UserAuthenticator)(nil)

func NewUserAuthenticator(userService services.UserService, proj *project.Project, token string, identityFactory factories.Identity) factories.Authenticator {
	return &UserAuthenticator{
		userService:     userService,
		project:         proj,
		token:           token,
		identityFactory: identityFactory,
		logger:          logger.New("api_key_authenticator"),
	}
}

func (a *UserAuthenticator) Authenticate(ctx context.Context) (*authentication.Authentication, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	externalUserID, err := a.identityFactory.Identify(ctx, a.token)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to identify user", logger.Error(err))
		return nil, err
	}

	usr, err := a.userService.GetOrCreate(ctx, a.project.ID, externalUserID, a.identityFactory.GetProviderID())
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get or create user", logger.Error(err))
		return nil, err
	}

	return &authentication.Authentication{
		UserID:         usr.ID,
		ProjectID:      a.project.ID,
		ExternalUserID: externalUserID,
	}, nil
}
