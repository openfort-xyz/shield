package session_builder

import (
	"context"
	"errors"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/builders"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
)

type sessionBuilder struct {
	projectPart            string
	databasePart           string
	encryptionPartsRepo    repositories.EncryptionPartsRepository
	projectRepo            repositories.ProjectRepository
	reconstructionStrategy strategies.ReconstructionStrategy
}

func NewEncryptionKeyBuilder(encryptionPartsRepo repositories.EncryptionPartsRepository, projectRepository repositories.ProjectRepository, reconstructionStrategy strategies.ReconstructionStrategy) builders.EncryptionKeyBuilder {
	return &sessionBuilder{
		encryptionPartsRepo:    encryptionPartsRepo,
		projectRepo:            projectRepository,
		reconstructionStrategy: reconstructionStrategy,
	}
}

func (b *sessionBuilder) SetProjectPart(ctx context.Context, identifier string) error {
	part, err := b.encryptionPartsRepo.Get(ctx, identifier)
	if err != nil {
		if errors.Is(err, domainErrors.ErrEncryptionPartNotFound) {
			return domainErrors.ErrInvalidEncryptionSession
		}
		return err
	}

	err = b.encryptionPartsRepo.Delete(ctx, identifier)
	if err != nil {
		return err
	}

	b.projectPart = part
	return nil
}

func (b *sessionBuilder) SetDatabasePart(ctx context.Context, identifier string) error {
	part, err := b.projectRepo.GetEncryptionPart(ctx, identifier)
	if err != nil {
		return err
	}

	b.databasePart = part
	return nil
}

func (b *sessionBuilder) Build(ctx context.Context) (string, error) {
	if b.projectPart == "" {
		return "", errors.New("project part is required") // TODO extract error
	}

	if b.databasePart == "" {
		return "", errors.New("database part is required") // TODO extract error
	}

	return b.reconstructionStrategy.Reconstruct(b.databasePart, b.projectPart)
}
