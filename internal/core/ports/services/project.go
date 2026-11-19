package services

import (
	"context"
	"time"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

type ProjectService interface {
	Create(ctx context.Context, name string, enable2fa bool) (*project.Project, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
	// SetDisabled disables or re-enables the project with apiKey and evicts
	// it from this process's API-key cache. It returns ErrProjectNotFound
	// when no project has that key.
	SetDisabled(ctx context.Context, apiKey string, disabled bool) error
	SaveProjectRateLimits(ctx context.Context, projectID string, smsRateLimit int64, emaiRateLimit int64) error
	// SetEncryptionKey stores the project's encryption key once check passes
	// under the project row lock.
	SetEncryptionKey(ctx context.Context, projectID string, key *project.EncryptionKey, check project.CheckAPISecretFunc) error
	// PrepareAPIKeyRotation stores a new pending API key and returns it.
	// It fails with ErrAPIKeyRevoked or ErrAPISecretRevoked when the presented
	// key or secret is no longer current (a commit landed after authentication).
	PrepareAPIKeyRotation(ctx context.Context, projectID, presentedKey, presentedSecret string, expiresAt time.Time) (string, error)
	// CommitAPIKeyRotation makes the pending key current when presentedKey is
	// that pending key, and reports false when presentedKey is already the
	// current key with nothing pending (a retry after a lost response). It
	// fails with ErrAPISecretRevoked when presentedSecret is no longer current.
	CommitAPIKeyRotation(ctx context.Context, projectID, presentedKey, presentedSecret string) (currentKey string, committed bool, err error)
	// PrepareAPISecretRotation stores the hash of a new pending API secret
	// and returns the secret. It fails with ErrAPIKeyRevoked or
	// ErrAPISecretRevoked when the presented key or secret is no longer
	// current (a commit landed after authentication).
	PrepareAPISecretRotation(ctx context.Context, projectID, presentedKey, presentedSecret string, expiresAt time.Time) (string, error)
	// CommitAPISecretRotation makes the pending secret current when
	// presentedSecret is that pending secret, and reports false when it is
	// already the current secret with nothing pending (a retry after a lost
	// response).
	CommitAPISecretRotation(ctx context.Context, projectID, presentedKey, presentedSecret string) (committed bool, err error)
}
