package sessbldr

import (
	"context"
	"encoding/json"
	"errors"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/internal/core/ports/builders"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/strategies"
	"github.com/openfort-xyz/shield/pkg/cypher"
)

type sessionBuilder struct {
	projectPart            project.ProjectPart
	databasePart           project.StoredPart
	keyCheck               *project.KeyCheck
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

	b.projectPart = project.ProjectPart(part.EncPart)
	return nil
}

func (b *sessionBuilder) SetDatabasePart(ctx context.Context, identifier string) error {
	key, err := b.projectRepo.GetEncryptionKey(ctx, identifier)
	if err != nil {
		return err
	}

	b.databasePart = key.Part
	b.keyCheck = key.KeyCheck
	return nil
}

func (b *sessionBuilder) GetProjectPart(_ context.Context) project.ProjectPart {
	return b.projectPart
}

func (b *sessionBuilder) GetDatabasePart(_ context.Context) project.StoredPart {
	return b.databasePart
}

func (b *sessionBuilder) Build(_ context.Context) (string, error) {
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
