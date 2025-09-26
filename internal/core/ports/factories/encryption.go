package factories

import (
	"go.openfort.xyz/shield/internal/core/ports/builders"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
)

type EncryptionFactory interface {
	CreateEncryptionKeyBuilder(builderType EncryptionKeyBuilderType, projectMigrated bool, otpRequired bool) (builders.EncryptionKeyBuilder, error)
	CreateReconstructionStrategy(projectMigrated bool) strategies.ReconstructionStrategy
	CreateEncryptionStrategy(key string) strategies.EncryptionStrategy
}

type EncryptionKeyBuilderType int8

const (
	Plain EncryptionKeyBuilderType = iota
	Session
)
