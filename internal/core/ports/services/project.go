package services

import (
	"context"
	"time"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

type ProjectService interface {
	Create(ctx context.Context, name string, enable2fa bool) (*project.Project, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
	SaveProjectRateLimits(ctx context.Context, projectID string, smsRateLimit int64, emaiRateLimit int64) error
	SetEncryptionKey(ctx context.Context, projectID string, key *project.EncryptionKey) error
	// PrepareAPIKeyRotation stores a new pending API key and returns it.
	// PrepareAPIKeyRotation fails with ErrAPIKeyRevoked when presentedKey is no
	// longer the project's current key (a commit landed after authentication).
	PrepareAPIKeyRotation(ctx context.Context, projectID, presentedKey string, expiresAt time.Time) (string, error)
	// CommitAPIKeyRotation makes the pending key current when presentedKey is
	// that pending key, and reports false when presentedKey is already the
	// current key with nothing pending (a retry after a lost response).
	CommitAPIKeyRotation(ctx context.Context, projectID, presentedKey string) (currentKey string, committed bool, err error)
}
