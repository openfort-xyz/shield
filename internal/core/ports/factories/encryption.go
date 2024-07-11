package factories

import (
	"go.openfort.xyz/shield/internal/core/ports/builders"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
)

type EncryptionFactory interface {
	CreateEncryptionKeyBuilder(builderType EncryptionKeyBuilderType) (builders.EncryptionKeyBuilder, error)
	CreateReconstructionStrategy() strategies.ReconstructionStrategy
	CreateEncryptionStrategy(key string) strategies.EncryptionStrategy
}

type EncryptionKeyBuilderType int8

const (
	Plain EncryptionKeyBuilderType = iota
	Session
)
