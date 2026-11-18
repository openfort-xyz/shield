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
	// under a pre-migration key also aborts if the project has a successful
	// migration row when the write takes its part-row lock. The
	// zero value is the pre-migration case, so a guard built without it for a
	// migrated project aborts rather than persists.
	Migrated bool
	// Key is the key the write encrypted under. A project can gain a key
	// check after the key was rebuilt without one (rotation prepare); the
	// write then persists only if Key matches it.
	Key string
}

func (s *Share) RequiresEncryption() bool {
	return s.Entropy == EntropyProject
}

const DefaultReference = "default"
