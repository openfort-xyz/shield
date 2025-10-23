package repositories

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/project"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *project.Project) error
	SaveProjectRateLimits(ctx context.Context, rateLimits *project.RateLimit) error
	Get(ctx context.Context, projectID string) (*project.Project, error)
	GetWithRateLimit(ctx context.Context, projectID string) (*project.WithRateLimit, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
	Delete(ctx context.Context, projectID string) error

	GetEncryptionPart(ctx context.Context, projectID string) (string, error)
	SetEncryptionPart(ctx context.Context, projectID, part string) error

	UpdateAPISecret(ctx context.Context, projectID, encryptedSecret string) error

	CreateMigration(ctx context.Context, projectID string, success bool) error
	HasSuccessfulMigration(ctx context.Context, projectID string) (bool, error)
}
