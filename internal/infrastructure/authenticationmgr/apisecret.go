package authenticationmgr

import (
	"context"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type apiSecret struct {
	projectRepo repositories.ProjectRepository
	logger      *slog.Logger
}

var _ authentication.APISecretAuthenticator = (*apiSecret)(nil)

func newAPISecretAuthenticator(repository repositories.ProjectRepository) authentication.APISecretAuthenticator {
	return &apiSecret{
		projectRepo: repository,
		logger:      logger.New("api_key_authenticator"),
	}
}

func (a *apiSecret) Authenticate(ctx context.Context, apiKey, apiSecret string) (string, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.projectRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api key", logger.Error(err))
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(proj.APISecret), []byte(apiSecret))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api secret", logger.Error(err))
		return "", err
	}

	return proj.ID, nil
}
