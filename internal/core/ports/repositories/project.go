package repositories

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/project"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrProjectExists   = errors.New("project exists")
)

type ProjectRepository interface {
	Create(ctx context.Context, project *project.Project) error
	Get(ctx context.Context, projectID string) (*project.Project, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
}
