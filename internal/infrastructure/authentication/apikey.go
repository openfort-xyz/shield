package authentication

import (
	"context"
	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type apiKey struct {
	projectRepo repositories.ProjectRepository
	logger      *slog.Logger
}

var _ authentication.APIKeyAuthenticator = (*apiKey)(nil)

func newAPIKeyAuthenticator(repository repositories.ProjectRepository) authentication.APIKeyAuthenticator {
	return &apiKey{
		projectRepo: repository,
		logger:      slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("api_key_authenticator"),
	}
}

func (a *apiKey) Authenticate(ctx context.Context, apiKey string) (string, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.projectRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api key", slog.String("error", err.Error()))
		return "", err
	}

	return proj.ID, nil
}
