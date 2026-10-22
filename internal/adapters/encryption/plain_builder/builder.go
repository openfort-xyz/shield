package plnbldr

import (
	"context"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/pkg/cypher"

	"github.com/openfort-xyz/shield/internal/core/ports/builders"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/strategies"
)

type plainBuilder struct {
	projectPart            project.ProjectPart
	databasePart           project.StoredPart
	keyCheck               *project.KeyCheck
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
	b.projectPart = project.ProjectPart(identifier)
	return nil
}

func (b *plainBuilder) SetDatabasePart(ctx context.Context, identifier string) error {
	key, err := b.projectRepo.GetEncryptionKey(ctx, identifier)
	if err != nil {
		return err
	}

	b.databasePart = key.Part
	b.keyCheck = key.KeyCheck
	return nil
}

func (b *plainBuilder) GetProjectPart(_ context.Context) project.ProjectPart {
	return b.projectPart
}

func (b *plainBuilder) GetDatabasePart(_ context.Context) project.StoredPart {
	return b.databasePart
}

func (b *plainBuilder) Build(_ context.Context) (string, error) {
	if b.projectPart == "" {
		return "", domainErrors.ErrProjectPartRequired
	}

	if b.databasePart == "" {
		return "", domainErrors.ErrDatabasePartRequired
	}

	key, err := b.reconstructionStrategy.Reconstruct(b.databasePart, b.projectPart)
	if err != nil {
		return "", err
	}

	if b.keyCheck != nil && !cypher.VerifyKeyCheck(key, string(*b.keyCheck)) {
		return "", domainErrors.ErrInvalidEncryptionPart
	}

	return key, nil
}
