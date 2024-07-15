package plnbldr

import (
	"context"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"

	"go.openfort.xyz/shield/internal/core/ports/builders"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
)

type plainBuilder struct {
	projectPart            string
	databasePart           string
	projectRepo            repositories.ProjectRepository
	reconstructionStrategy strategies.ReconstructionStrategy
}

func NewEncryptionKeyBuilder(repo repositories.ProjectRepository, reconstructionStrategy strategies.ReconstructionStrategy) builders.EncryptionKeyBuilder {
	return &plainBuilder{
		projectRepo:            repo,
		reconstructionStrategy: reconstructionStrategy,
	}
}

func (b *plainBuilder) SetProjectPart(_ context.Context, identifier string) error {
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

func (b *plainBuilder) GetProjectPart(ctx context.Context) string {
	return b.projectPart

}

func (b *plainBuilder) GetDatabasePart(ctx context.Context) string {
	return b.databasePart
}

func (b *plainBuilder) Build(_ context.Context) (string, error) {
	if b.projectPart == "" {
		return "", domainErrors.ErrProjectPartRequired
	}

	if b.databasePart == "" {
		return "", domainErrors.ErrDatabasePartRequired
	}

	return b.reconstructionStrategy.Reconstruct(b.databasePart, b.projectPart)
}
