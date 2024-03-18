package services

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/project"
)

type ProjectService interface {
	Create(ctx context.Context, name string) (*project.Project, error)
	Get(ctx context.Context, projectID string) (*project.Project, error)
}
