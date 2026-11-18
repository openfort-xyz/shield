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
	// Migrated is whether the key came from the current reconstruction. A
	// migration changes the key without changing the stored part, so a write
	// under a pre-migration key also aborts once the project is migrated. The
	// zero value is the pre-migration case, so a guard built without it for a
	// migrated project aborts rather than persists.
	Migrated bool
	// UnverifiedKey is the key the write encrypted under when the project
	// had no key check to verify it against. The write then persists only if
	// the key decrypts one of up to 1000 of the project's project shares, or
	// the project has none, or it matches a key check stored since
	// (sharerepo.checkGuard). Left empty for unmigrated projects: their key is
	// the stored part whatever part the caller sent, so there is no part to
	// prove.
	UnverifiedKey string
}

func (s *Share) RequiresEncryption() bool {
	return s.Entropy == EntropyProject
}

const DefaultReference = "default"
