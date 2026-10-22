package project

import "time"

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
// read under a lock on the project row.
type APIKeyRotation struct {
	APIKey           string
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

// CheckAPISecretFunc vets a secret prepare against the locked state: the
// credentials it authenticated with may have been rotated since.
type CheckAPISecretFunc func(state APISecretRotation) error

// CommitAPISecretRotationFunc decides whether the pending secret replaces
// the current one.
type CommitAPISecretRotationFunc func(state APISecretRotation) (swap bool, err error)
