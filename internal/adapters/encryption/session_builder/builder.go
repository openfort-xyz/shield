package sessbldr

import (
	"context"
	"encoding/json"
	"errors"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/share"
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
	requireOTPCheck        bool
}

func NewEncryptionKeyBuilder(encryptionPartsRepo repositories.EncryptionPartsRepository, projectRepository repositories.ProjectRepository, reconstructionStrategy strategies.ReconstructionStrategy, requireOTPCheck bool) builders.EncryptionKeyBuilder {
	return &sessionBuilder{
		encryptionPartsRepo:    encryptionPartsRepo,
		projectRepo:            projectRepository,
		reconstructionStrategy: reconstructionStrategy,
		requireOTPCheck:        requireOTPCheck,
	}
}

func (b *sessionBuilder) SetProjectPart(ctx context.Context, identifier string) error {
	data, err := b.encryptionPartsRepo.Get(ctx, identifier)
	if err != nil {
		if errors.Is(err, domainErrors.ErrDataInDBNotFound) {
			return domainErrors.ErrInvalidEncryptionSession
		}
		return err
	}

	var part share.EncryptionPart
	if err := json.Unmarshal([]byte(data), &part); err != nil {
		return err
	}

	if b.requireOTPCheck && !part.OTPVerified {
		return domainErrors.ErrOTPVerificationRequired
	}

	err = b.encryptionPartsRepo.Delete(ctx, identifier)
	if err != nil {
		return err
	}

	b.projectPart = part.EncPart
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

func (b *sessionBuilder) GetProjectPart(_ context.Context) string {
	return b.projectPart
}

func (b *sessionBuilder) GetDatabasePart(_ context.Context) string {
	return b.databasePart
}

func (b *sessionBuilder) Build(_ context.Context) (string, error) {
	if b.projectPart == "" {
		return "", domainErrors.ErrProjectPartRequired
	}

	if b.databasePart == "" {
		return "", domainErrors.ErrDatabasePartRequired
	}

	return b.reconstructionStrategy.Reconstruct(b.databasePart, b.projectPart)
}
