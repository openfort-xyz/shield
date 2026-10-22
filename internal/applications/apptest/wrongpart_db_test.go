package apptest

import (
	"errors"
	"testing"

	sssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/random"
)

// Shamir combine accepts any well-formed part, so a part from another split
// yields a plausible key. Encrypting under it stores a share nobody can read.
func TestRegisterShare_WrongPartIsRejectedNotStored(t *testing.T) {
	apps := New(t)
	p, _ := apps.SeedEncryptedProject(t)
	_, otherProjectsPart := apps.SeedEncryptedProject(t)

	err := apps.Share.RegisterShare(UserContext(p),
		&share.Share{Secret: "wallet-share", Entropy: share.EntropyProject},
		shareapp.WithEncryptionPart(otherProjectsPart))

	if !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
		t.Errorf("RegisterShare with a wrong part: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
	}
	if n := apps.ShareCount(t, p); n != 0 {
		t.Errorf("share stored under a wrong key: %d share rows, want 0", n)
	}
}

// Keys created before key checks existed have no check value; they must keep
// working, verified only by AES-GCM on decrypt.
func TestGetShare_LegacyKeyWithoutCheckStillDecrypts(t *testing.T) {
	apps := New(t)
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	storedPart, projectPart, err := sssrec.NewSSSReconstructionStrategy().Split(key)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	p := dbtest.SeedProject(t, apps.DB, string(storedPart), true)
	ctx := UserContext(p)

	err = apps.Share.RegisterShare(ctx, &share.Share{Secret: "wallet-share", Entropy: share.EntropyProject},
		shareapp.WithEncryptionPart(string(projectPart)))
	if err != nil {
		t.Fatalf("RegisterShare on a legacy key: %v", err)
	}

	got, err := apps.Share.GetShare(ctx, shareapp.WithEncryptionPart(string(projectPart)))
	if err != nil {
		t.Fatalf("GetShare on a legacy key: %v", err)
	}
	if got.Secret != "wallet-share" {
		t.Errorf("GetShare secret = %q, want %q", got.Secret, "wallet-share")
	}
}
