package factories

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/ports/builders"
	"github.com/openfort-xyz/shield/internal/core/ports/strategies"
)

type EncryptionFactory interface {
	CreateEncryptionKeyBuilder(builderType EncryptionKeyBuilderType, projectMigrated bool, sessionPolicy EncryptionSessionPolicy) (builders.EncryptionKeyBuilder, error)
	CreateReconstructionStrategy(projectMigrated bool) strategies.ReconstructionStrategy
	CreateEncryptionStrategy(key string) strategies.EncryptionStrategy
	GrantRecoveryChange(ctx context.Context, sessionID, shareID, ciphertext string) error
	ConsumeRecoveryChange(ctx context.Context, sessionID, shareID, ciphertext string) error
}

type EncryptionSessionPolicy uint8

const (
	OTPOptional EncryptionSessionPolicy = iota
	OTPRequired
	OTPOrSignup
)

type EncryptionKeyBuilderType int8

const (
	Plain EncryptionKeyBuilderType = iota
	Session
)
