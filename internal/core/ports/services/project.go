package services

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/project"
)

type ProjectService interface {
	Create(ctx context.Context, name string) (*project.Project, error)
	SetEncryptionPart(ctx context.Context, projectID, part string) error
}
