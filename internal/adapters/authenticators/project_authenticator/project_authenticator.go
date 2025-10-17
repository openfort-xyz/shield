package projauth

import (
	"context"
	"encoding/hex"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/authentication"
	"go.openfort.xyz/shield/internal/core/ports/factories"

	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type ProjectAuthenticator struct {
	projectRepo       repositories.ProjectRepository
	apiKey, apiSecret string
	logger            *slog.Logger
}

var _ factories.Authenticator = (*ProjectAuthenticator)(nil)

func NewProjectAuthenticator(repository repositories.ProjectRepository, apiKey, apiSecret string) factories.Authenticator {
	return &ProjectAuthenticator{
		projectRepo: repository,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		logger:      logger.New("api_key_authenticator"),
	}
}

func getApiSecretBytes(apiSecret string) []byte {
	hex32bytes, err := hex.DecodeString(apiSecret)
	if err != nil {
		// Old legacy api secrets are UUIDs and new secrets are hex-encoded 32 bytes
		return []byte(apiSecret)
	}
	return hex32bytes
}

func (a *ProjectAuthenticator) Authenticate(ctx context.Context) (*authentication.Authentication, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.projectRepo.GetByAPIKey(ctx, a.apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api key", logger.Error(err))
		return nil, err
	}

	apiSecretBytes := getApiSecretBytes(a.apiSecret)

	err = bcrypt.CompareHashAndPassword([]byte(proj.APISecret), apiSecretBytes)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api secret", logger.Error(err))
		return nil, err
	}

	return &authentication.Authentication{
		ProjectID: proj.ID,
	}, nil
}
