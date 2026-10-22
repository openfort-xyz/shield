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
}

func (s *Share) RequiresEncryption() bool {
	return s.Entropy == EntropyProject
}

const DefaultReference = "default"
