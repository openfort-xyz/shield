package services

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/project"
)

type ProjectService interface {
	Create(ctx context.Context, name string, enable2fa bool) (*project.Project, error)
	SetEncryptionPart(ctx context.Context, projectID, part string) error
}
