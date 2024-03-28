package repositories

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/project"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *project.Project) error
	Get(ctx context.Context, projectID string) (*project.Project, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
	AddAllowedOrigin(ctx context.Context, projectID, origin string) error
	RemoveAllowedOrigin(ctx context.Context, projectID, origin string) error
	GetAllowedOrigins(ctx context.Context, projectID string) ([]string, error)
	GetAllowedOriginsByAPIKey(ctx context.Context, apiKey string) ([]string, error)
}
