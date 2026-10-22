package projauth

import (
	"context"
	"encoding/hex"
	"log/slog"

	"github.com/openfort-xyz/shield/internal/core/domain/authentication"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"

	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type ProjectAuthenticator struct {
	lookup            func(ctx context.Context, apiKey string) (*project.Project, error)
	apiKey, apiSecret string
	logger            *slog.Logger
	// lookupMissLevel is the log level for an unknown key: the commit route
	// tries the pending key first, so a miss there is routine.
	lookupMissLevel slog.Level
}

var _ factories.Authenticator = (*ProjectAuthenticator)(nil)

func NewProjectAuthenticator(repository repositories.ProjectRepository, apiKey, apiSecret string) factories.Authenticator {
	return &ProjectAuthenticator{
		lookup:          repository.GetByAPIKey,
		apiKey:          apiKey,
		apiSecret:       apiSecret,
		logger:          logger.New("api_key_authenticator"),
		lookupMissLevel: slog.LevelError,
	}
}

// NewPendingProjectAuthenticator authenticates with a project's unexpired
// pending API key (from a prepared rotation) and its API secret.
func NewPendingProjectAuthenticator(repository repositories.ProjectRepository, apiKey, apiSecret string) factories.Authenticator {
	return &ProjectAuthenticator{
		lookup:          repository.GetByPendingAPIKey,
		apiKey:          apiKey,
		apiSecret:       apiSecret,
		logger:          logger.New("pending_api_key_authenticator"),
		lookupMissLevel: slog.LevelDebug,
	}
}

func getAPISecretBytes(apiSecret string) []byte {
	hex32bytes, err := hex.DecodeString(apiSecret)
	if err != nil {
		// Old legacy api secrets are UUIDs and new secrets are hex-encoded 32 bytes
		return []byte(apiSecret)
	}
	return hex32bytes
}

func (a *ProjectAuthenticator) Authenticate(ctx context.Context) (*authentication.Authentication, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.lookup(ctx, a.apiKey)
	if err != nil {
		a.logger.Log(ctx, a.lookupMissLevel, "failed to authenticate api key", logger.Error(err))
		return nil, err
	}

	apiSecretBytes := getAPISecretBytes(a.apiSecret)

	err = bcrypt.CompareHashAndPassword([]byte(proj.APISecret), apiSecretBytes)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to authenticate api secret", logger.Error(err))
		return nil, err
	}

	return &authentication.Authentication{
		ProjectID: proj.ID,
	}, nil
}
