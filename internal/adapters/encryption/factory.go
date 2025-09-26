package encryption

import (
	aesencryptionstrategy "go.openfort.xyz/shield/internal/adapters/encryption/aes_encryption_strategy"
	depsssrec "go.openfort.xyz/shield/internal/adapters/encryption/deprecated_sss_reconstruction_strategy"
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

func (e *encryptionFactory) CreateEncryptionKeyBuilder(builderType factories.EncryptionKeyBuilderType, projectMigrated bool, otpRequired bool) (builders.EncryptionKeyBuilder, error) {
	var reconstructionStrategy strategies.ReconstructionStrategy
	if projectMigrated {
		reconstructionStrategy = sssrec.NewSSSReconstructionStrategy()
	} else {
		reconstructionStrategy = depsssrec.NewSSSReconstructionStrategy()
	}
	switch builderType {
	case factories.Plain:
		return plnbldr.NewEncryptionKeyBuilder(e.projectRepo, reconstructionStrategy), nil
	case factories.Session:
		return sessbldr.NewEncryptionKeyBuilder(e.encryptionPartsRepo, e.projectRepo, reconstructionStrategy, otpRequired), nil
	}

	return nil, errors.ErrInvalidEncryptionKeyBuilderType
}

func (e *encryptionFactory) CreateReconstructionStrategy(projectMigrated bool) strategies.ReconstructionStrategy {
	if projectMigrated {
		return sssrec.NewSSSReconstructionStrategy()
	}
	return depsssrec.NewSSSReconstructionStrategy()
}

func (e *encryptionFactory) CreateEncryptionStrategy(key string) strategies.EncryptionStrategy {
	return aesencryptionstrategy.NewAESEncryptionStrategy(key)
}
