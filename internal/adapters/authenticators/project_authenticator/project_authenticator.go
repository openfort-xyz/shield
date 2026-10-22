package projauth

import (
	"context"
	"errors"
	"log/slog"
	"time"

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
	// secretHash selects the stored hash the secret is checked against.
	secretHash func(*project.Project) (string, bool)
	logger     *slog.Logger
	// lookupMissLevel and secretMissLevel are the log levels for an unknown
	// key and a wrong secret: the commit routes try the pending credential
	// first, so a miss there is routine.
	lookupMissLevel slog.Level
	secretMissLevel slog.Level
}

var errNoPendingAPISecret = errors.New("no unexpired pending API secret")

func currentSecretHash(proj *project.Project) (string, bool) {
	return proj.APISecret, true
}

func pendingSecretHash(proj *project.Project) (string, bool) {
	if proj.PendingAPISecret == nil || proj.PendingAPISecretExpiresAt == nil ||
		!time.Now().Before(*proj.PendingAPISecretExpiresAt) {
		return "", false
	}
	return *proj.PendingAPISecret, true
}

var _ factories.Authenticator = (*ProjectAuthenticator)(nil)

func NewProjectAuthenticator(repository repositories.ProjectRepository, apiKey, apiSecret string) factories.Authenticator {
	return &ProjectAuthenticator{
		lookup:          repository.GetByAPIKey,
		apiKey:          apiKey,
		apiSecret:       apiSecret,
		secretHash:      currentSecretHash,
		logger:          logger.New("api_key_authenticator"),
		lookupMissLevel: slog.LevelError,
		secretMissLevel: slog.LevelError,
	}
}

// NewPendingProjectAuthenticator authenticates with a project's unexpired
// pending API key (from a prepared rotation) and its API secret.
func NewPendingProjectAuthenticator(repository repositories.ProjectRepository, apiKey, apiSecret string) factories.Authenticator {
	return &ProjectAuthenticator{
		lookup:          repository.GetByPendingAPIKey,
		apiKey:          apiKey,
		apiSecret:       apiSecret,
		secretHash:      currentSecretHash,
		logger:          logger.New("pending_api_key_authenticator"),
		lookupMissLevel: slog.LevelDebug,
		secretMissLevel: slog.LevelError,
	}
}

// NewPendingSecretProjectAuthenticator authenticates with a project's current
// API key and its unexpired pending API secret (from a prepared rotation).
// Only the secret commit route uses it.
func NewPendingSecretProjectAuthenticator(repository repositories.ProjectRepository, apiKey, apiSecret string) factories.Authenticator {
	return &ProjectAuthenticator{
		lookup:          repository.GetByAPIKey,
		apiKey:          apiKey,
		apiSecret:       apiSecret,
		secretHash:      pendingSecretHash,
		logger:          logger.New("pending_api_secret_authenticator"),
		lookupMissLevel: slog.LevelDebug,
		secretMissLevel: slog.LevelDebug,
	}
}

func (a *ProjectAuthenticator) Authenticate(ctx context.Context) (*authentication.Authentication, error) {
	a.logger.InfoContext(ctx, "authenticating api key")

	proj, err := a.lookup(ctx, a.apiKey)
	if err != nil {
		a.logger.Log(ctx, a.lookupMissLevel, "failed to authenticate api key", logger.Error(err))
		return nil, err
	}

	hash, ok := a.secretHash(proj)
	if !ok {
		a.logger.Log(ctx, a.secretMissLevel, "failed to authenticate api secret", logger.Error(errNoPendingAPISecret))
		return nil, errNoPendingAPISecret
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), project.APISecretBytes(a.apiSecret))
	if err != nil {
		a.logger.Log(ctx, a.secretMissLevel, "failed to authenticate api secret", logger.Error(err))
		return nil, err
	}

	return &authentication.Authentication{
		ProjectID: proj.ID,
	}, nil
}
