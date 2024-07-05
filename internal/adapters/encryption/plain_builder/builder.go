package plain_builder

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/ports/builders"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/cypher"
)

type plainBuilder struct {
	projectPart  string
	databasePart string
	projectRepo  repositories.ProjectRepository
}

func NewEncryptionKeyBuilder(repo repositories.ProjectRepository) builders.EncryptionKeyBuilder {
	return &plainBuilder{
		projectRepo: repo,
	}
}

func (b *plainBuilder) SetProjectPart(ctx context.Context, identifier string) error {
	b.projectPart = identifier
	return nil
}

func (b *plainBuilder) SetDatabasePart(ctx context.Context, identifier string) error {
	part, err := b.projectRepo.GetEncryptionPart(ctx, identifier)
	if err != nil {
		return err
	}

	b.databasePart = part
	return nil
}

func (b *plainBuilder) Build(ctx context.Context) (string, error) {
	if b.projectPart == "" {
		return "", errors.New("project part is required") // TODO extract error
	}

	if b.databasePart == "" {
		return "", errors.New("database part is required") // TODO extract error
	}

	return cypher.ReconstructEncryptionKey(b.projectPart, b.databasePart)
}
