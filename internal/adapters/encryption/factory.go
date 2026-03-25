package encryption

import (
	aesencryptionstrategy "github.com/openfort-xyz/shield/internal/adapters/encryption/aes_encryption_strategy"
	depsssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/deprecated_sss_reconstruction_strategy"
	plnbldr "github.com/openfort-xyz/shield/internal/adapters/encryption/plain_builder"
	sessbldr "github.com/openfort-xyz/shield/internal/adapters/encryption/session_builder"
	sssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/ports/builders"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/strategies"
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
