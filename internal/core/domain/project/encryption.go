package project

import (
	"time"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"golang.org/x/crypto/bcrypt"
)

// StoredPart is the half of a project's encryption key that Shield stores.
type StoredPart string

// ProjectPart is the half of a project's encryption key the project holds.
// It is a distinct type from StoredPart because combining the two in the
// wrong order, or with a part from another split, silently yields a key
// nobody holds.
type ProjectPart string //nolint:revive // "project part" is the domain term; project.Part would be ambiguous next to StoredPart

// KeyCheck identifies an encryption key without revealing it.
type KeyCheck string

// EncryptionKey is a project's stored encryption state. KeyCheck is nil for
// keys created before key checks existed; such keys cannot be verified.
type EncryptionKey struct {
	Part     StoredPart
	KeyCheck *KeyCheck
}

// PendingEncryptionKey is a prepared, not yet committed, replacement key.
type PendingEncryptionKey struct {
	Part      StoredPart
	KeyCheck  KeyCheck
	ExpiresAt time.Time
}

// EncryptedShare is a project-entropy share's ciphertext.
type EncryptedShare struct {
	ID     string
	Secret string
}

// EncryptionKeyRotation is a project's encryption state as seen by a
// rotation, read under a lock on the project's encryption part rows.
type EncryptionKeyRotation struct {
	Current  EncryptionKey
	Pending  *PendingEncryptionKey
	Migrated bool
}

// PreparedRotation is what preparing a rotation stores: the pending key, and
// the current key's check (set for keys that predate key checks).
type PreparedRotation struct {
	Pending         PendingEncryptionKey
	CurrentKeyCheck KeyCheck
}

// CommittedRotation is what committing a rotation stores: the new current
// key and every project share re-encrypted under it.
type CommittedRotation struct {
	Key    EncryptionKey
	Shares []EncryptedShare
}

// PrepareRotationFunc decides a rotation's pending key. loadShares returns
// the project's live project-entropy shares.
type PrepareRotationFunc func(state EncryptionKeyRotation, loadShares func() ([]EncryptedShare, error)) (*PreparedRotation, error)

// CommitRotationFunc decides a rotation's outcome; nil means nothing to change.
type CommitRotationFunc func(state EncryptionKeyRotation, shares []EncryptedShare) (*CommittedRotation, error)

// APIKeyRotation is a project's API-key state as seen by a rotation commit,
// read under a lock on the project row. The secret is a bcrypt hash.
type APIKeyRotation struct {
	APIKey           string
	APISecretHash    string
	PendingAPIKey    *string
	PendingExpiresAt *time.Time
}

// CommitAPIKeyRotationFunc decides whether the pending key replaces the
// current one.
type CommitAPIKeyRotationFunc func(state APIKeyRotation) (swap bool, err error)

// APISecretRotation is a project's API-secret state as seen by a rotation
// commit, read under a lock on the project row. Secrets are bcrypt hashes.
type APISecretRotation struct {
	APIKey           string
	APISecretHash    string
	PendingHash      *string
	PendingExpiresAt *time.Time
}

// CheckAPISecretFunc vets a request's credentials against the project row
// read under lock: they may have been rotated since the request authenticated.
type CheckAPISecretFunc func(state APISecretRotation) error

// StillCurrent fails with ErrAPIKeyRevoked or ErrAPISecretRevoked when the
// key or secret a request authenticated with is no longer the project's.
func StillCurrent(apiKey, apiSecret string) CheckAPISecretFunc {
	return func(state APISecretRotation) error {
		if apiKey != state.APIKey {
			return domainErrors.ErrAPIKeyRevoked
		}
		if bcrypt.CompareHashAndPassword([]byte(state.APISecretHash), APISecretBytes(apiSecret)) != nil {
			return domainErrors.ErrAPISecretRevoked
		}
		return nil
	}
}

// CommitAPISecretRotationFunc decides whether the pending secret replaces
// the current one.
type CommitAPISecretRotationFunc func(state APISecretRotation) (swap bool, err error)
