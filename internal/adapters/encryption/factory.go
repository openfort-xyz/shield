package encryption

import (
	"errors"
	"go.openfort.xyz/shield/internal/adapters/encryption/aes_encryption_strategy"
	"go.openfort.xyz/shield/internal/adapters/encryption/plain_builder"
	"go.openfort.xyz/shield/internal/adapters/encryption/session_builder"
	"go.openfort.xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
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
		return plain_builder.NewEncryptionKeyBuilder(e.projectRepo, sss_reconstruction_strategy.NewSSSReconstructionStrategy()), nil
	case factories.Session:
		return session_builder.NewEncryptionKeyBuilder(e.encryptionPartsRepo, e.projectRepo, sss_reconstruction_strategy.NewSSSReconstructionStrategy()), nil
	}

	return nil, errors.New("invalid builder type") //TODO extract error
}

func (e *encryptionFactory) CreateReconstructionStrategy() strategies.ReconstructionStrategy {
	return sss_reconstruction_strategy.NewSSSReconstructionStrategy()
}

func (e *encryptionFactory) CreateEncryptionStrategy(key string) strategies.EncryptionStrategy {
	return aes_encryption_strategy.NewAESEncryptionStrategy(key)
}
