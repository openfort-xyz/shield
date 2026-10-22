package repositories

import (
	"context"
	"time"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *project.Project) error
	SaveProjectRateLimits(ctx context.Context, rateLimits *project.RateLimit) error
	Get(ctx context.Context, projectID string) (*project.Project, error)
	GetWithRateLimit(ctx context.Context, projectID string) (*project.WithRateLimit, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
	Delete(ctx context.Context, projectID string) error
	// HardDelete permanently removes the project row and, through the DB's
	// ON DELETE CASCADE constraints, all its providers, users, external
	// users, shares, keychains, encryption parts, notifications and rate
	// limits. It first removes the two rows the cascade cannot reach:
	// shld_passkey_references (plain FK to shld_shares) and
	// shld_shamir_migrations (no FK). Idempotent.
	HardDelete(ctx context.Context, projectID string) error

	// GetEncryptionKey returns ErrEncryptionPartNotFound when the project has
	// no encryption key.
	GetEncryptionKey(ctx context.Context, projectID string) (*project.EncryptionKey, error)
	SetEncryptionKey(ctx context.Context, projectID string, key *project.EncryptionKey) error

	// PrepareEncryptionKeyRotation runs prepare in a transaction holding
	// FOR UPDATE on the project's encryption part rows and stores its result.
	// It returns ErrEncryptionPartNotFound when the project has no key, and
	// prepare's error unchanged.
	PrepareEncryptionKeyRotation(ctx context.Context, projectID string, prepare project.PrepareRotationFunc) error
	// CommitEncryptionKeyRotation runs commit in a transaction holding
	// FOR UPDATE on the project's encryption part rows, then re-encrypts the
	// shares, replaces the part in place and clears the pending key. It
	// returns ErrEncryptionPartNotFound when the project has no key, and
	// commit's error unchanged.
	CommitEncryptionKeyRotation(ctx context.Context, projectID string, commit project.CommitRotationFunc) error

	// SetPendingAPIKey stores a pending key only while currentKey is still the
	// project's API key, else returns ErrAPIKeyRevoked.
	SetPendingAPIKey(ctx context.Context, projectID, currentKey, apiKey string, expiresAt time.Time) error
	// GetByPendingAPIKey returns the project whose unexpired pending API key
	// is apiKey, or ErrProjectNotFound.
	GetByPendingAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
	// CommitAPIKeyRotation runs commit under FOR UPDATE on the project row and,
	// when it returns true, makes the pending key current.
	CommitAPIKeyRotation(ctx context.Context, projectID string, commit project.CommitAPIKeyRotationFunc) error
	// SetPendingAPISecret stores the hash of a pending secret only while
	// currentKey is still the project's API key, else returns ErrAPIKeyRevoked.
	SetPendingAPISecret(ctx context.Context, projectID, currentKey, secretHash string, expiresAt time.Time) error
	// CommitAPISecretRotation runs commit under FOR UPDATE on the project row
	// and, when it returns true, makes the pending secret current.
	CommitAPISecretRotation(ctx context.Context, projectID string, commit project.CommitAPISecretRotationFunc) error

	Update2FA(ctx context.Context, projectID string, enable2FA bool) error

	CreateMigration(ctx context.Context, projectID string, success bool) error
	HasSuccessfulMigration(ctx context.Context, projectID string) (bool, error)
}
