package apptest

import (
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// A write reconstructs the key from the stored part it read, then persists.
// If a rotation commits in between, persisting would store ciphertext under
// the replaced key, which no current part can decrypt.
func TestRegisterShare_DuringRotation_IsNotPersisted(t *testing.T) {
	parking := NewParkingShareRepo()
	apps := NewWithShareRepo(t, parking.Wrap)
	p, part := apps.SeedEncryptedProject(t)

	parking.Armed.Store(true)
	errc := make(chan error, 1)
	go func() {
		errc <- apps.Share.RegisterShare(UserContext(p),
			&share.Share{Secret: "wallet-share", Entropy: share.EntropyProject},
			shareapp.WithEncryptionPart(part))
	}()
	<-parking.Parked
	newPart := apps.RotateOutOfBand(t, p, part)
	close(parking.Release)

	if err := <-errc; !errors.Is(err, shareapp.ErrEncryptionPartChanged) {
		t.Errorf("RegisterShare racing a rotation: err = %v, want %v", err, shareapp.ErrEncryptionPartChanged)
	}
	if n := apps.ShareCount(t, p); n != 0 {
		_, err := apps.Share.GetShare(UserContext(p), shareapp.WithEncryptionPart(newPart))
		t.Errorf("share written during rotation was persisted (%d rows); GetShare with the new part: %v", n, err)
	}
}

func TestUpdateShare_DuringRotation_IsNotPersisted(t *testing.T) {
	parking := NewParkingShareRepo()
	apps := NewWithShareRepo(t, parking.Wrap)
	p, part := apps.SeedEncryptedProject(t)
	ctx := UserContext(p)

	err := apps.Share.RegisterShare(ctx, &share.Share{Secret: "original", Entropy: share.EntropyProject},
		shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("RegisterShare: %v", err)
	}

	parking.Armed.Store(true)
	errc := make(chan error, 1)
	go func() {
		_, err := apps.Share.UpdateShare(ctx, &share.Share{Secret: "updated"}, share.DefaultReference,
			shareapp.WithEncryptionPart(part))
		errc <- err
	}()
	<-parking.Parked
	newPart := apps.RotateOutOfBand(t, p, part)
	close(parking.Release)

	if err := <-errc; !errors.Is(err, shareapp.ErrEncryptionPartChanged) {
		t.Errorf("UpdateShare racing a rotation: err = %v, want %v", err, shareapp.ErrEncryptionPartChanged)
	}
	got, err := apps.Share.GetShare(ctx, shareapp.WithEncryptionPart(newPart))
	if err != nil {
		t.Fatalf("share does not decrypt under the new part after a racing update: %v", err)
	}
	if got.Secret != "original" {
		t.Errorf("secret = %q, want %q", got.Secret, "original")
	}
}

// Moving a pre-keychain share into a keychain rewrites the share row. If it
// wrote back the ciphertext it read, a rotation committing in between would
// be undone for that share.
func TestKeychainMigration_DuringRotation_KeepsRotatedCiphertext(t *testing.T) {
	parking := NewParkingShareRepo()
	apps := NewWithShareRepo(t, parking.Wrap)
	p, part := apps.SeedEncryptedProject(t)
	ctx := UserContext(p)

	err := apps.Share.RegisterShare(ctx, &share.Share{Secret: "original", Entropy: share.EntropyProject},
		shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("RegisterShare: %v", err)
	}
	// Make it a pre-keychain share.
	if err := apps.DB.Exec(`UPDATE shld_shares SET keychain_id = NULL, reference = NULL WHERE user_id = ?`, p.UserID).Error; err != nil {
		t.Fatalf("detach keychain: %v", err)
	}
	if err := apps.DB.Exec(`DELETE FROM shld_keychains WHERE user_id = ?`, p.UserID).Error; err != nil {
		t.Fatalf("delete keychain: %v", err)
	}

	parking.Armed.Store(true)
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The migration runs first and parks in Update; the outcome of this
		// second registration does not matter.
		_ = apps.Share.RegisterShare(ctx, &share.Share{Secret: "second", Entropy: share.EntropyProject},
			shareapp.WithEncryptionPart(part))
	}()
	<-parking.Parked
	newPart := apps.RotateOutOfBand(t, p, part)
	parking.Armed.Store(false)
	close(parking.Release)
	<-done

	got, err := apps.Share.GetShare(ctx, shareapp.WithEncryptionPart(newPart))
	if err != nil {
		t.Fatalf("migrated share does not decrypt under the new part: %v", err)
	}
	if got.Secret != "original" {
		t.Errorf("secret = %q, want %q", got.Secret, "original")
	}
}
