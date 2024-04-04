package services

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/project"
)

type ProjectService interface {
	Create(ctx context.Context, name string) (*project.Project, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error)
	AddAllowedOrigin(ctx context.Context, projectID, origin string) error
	RemoveAllowedOrigin(ctx context.Context, projectID, origin string) error
	GetAllowedOrigins(ctx context.Context, projectID string) ([]string, error)
	GetEncryptionPart(ctx context.Context, projectID string) (string, error)
	SetEncryptionPart(ctx context.Context, projectID, part string) error
}
