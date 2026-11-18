package share

import "github.com/openfort-xyz/shield/internal/core/domain/project"

type Share struct {
	ID                   string
	Secret               string
	UserID               string
	KeychainID           *string
	Reference            *string
	Entropy              Entropy
	ShareStorageMethodID StorageMethodID
	EncryptionParameters *EncryptionParameters
	PasskeyReference     *PasskeyReference
	// EncryptionGuard, when set, makes persisting this share conditional on
	// the project's stored part being unchanged. Not persisted.
	EncryptionGuard *EncryptionGuard
}

// EncryptionGuard records the stored part a project-entropy write encrypted
// under, so the write aborts if a key rotation replaced it in the meantime.
type EncryptionGuard struct {
	ProjectID  string
	StoredPart project.StoredPart
	// UnverifiedKey is the key the write encrypted under when the project
	// had no key check to verify it against. The write then persists only if
	// the key is proven in the same transaction (sharerepo.proveKey). Left
	// empty for unmigrated projects: the shamir job changes their key without
	// changing the stored part, so a check stored now would reject the right
	// part after the migration.
	UnverifiedKey string
}

func (s *Share) RequiresEncryption() bool {
	return s.Entropy == EntropyProject
}

const DefaultReference = "default"
