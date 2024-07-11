package encryption

import (
	aesencryptionstrategy "go.openfort.xyz/shield/internal/adapters/encryption/aes_encryption_strategy"
	plnbldr "go.openfort.xyz/shield/internal/adapters/encryption/plain_builder"
	sessbldr "go.openfort.xyz/shield/internal/adapters/encryption/session_builder"
	sssrec "go.openfort.xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/builders"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
)

type encryptionFactory struct {
	encryptionPartsRepo repositories.EncryptionPartsRepository
	projectRepo         repositories.ProjectRepository
}

func NewEncryptionFactory(encryptionPartsRepo repositories.EncryptionPartsRepository, projectRepo repositories.ProjectRepository) factories.EncryptionFactory {
	return &encryptionFactory{
		encryptionPartsRepo: encryptionPartsRepo,
		projectRepo:         projectRepo,
	}
}

func (e *encryptionFactory) CreateEncryptionKeyBuilder(builderType factories.EncryptionKeyBuilderType) (builders.EncryptionKeyBuilder, error) {
	switch builderType {
	case factories.Plain:
		return plnbldr.NewEncryptionKeyBuilder(e.projectRepo, sssrec.NewSSSReconstructionStrategy()), nil
	case factories.Session:
		return sessbldr.NewEncryptionKeyBuilder(e.encryptionPartsRepo, e.projectRepo, sssrec.NewSSSReconstructionStrategy()), nil
	}

	return nil, errors.ErrInvalidEncryptionKeyBuilderType
}

func (e *encryptionFactory) CreateReconstructionStrategy() strategies.ReconstructionStrategy {
	return sssrec.NewSSSReconstructionStrategy()
}

func (e *encryptionFactory) CreateEncryptionStrategy(key string) strategies.EncryptionStrategy {
	return aesencryptionstrategy.NewAESEncryptionStrategy(key)
}
