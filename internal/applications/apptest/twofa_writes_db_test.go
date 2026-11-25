package apptest

import (
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// With 2FA on, writing a project-entropy share needs the same OTP-verified,
// user-bound session as reading one. Otherwise a stolen token plus the raw
// developer part, or anyone else's session, overwrites the victim's share.
func TestTwoFactor_ShareWritesNeedTheCallersOTPSession(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice, bob := apps.NewUser(t, p), apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	bobsSession := apps.OTPSession(t, p, part, bob.ExternalID, "bob@example.com")
	planted := &share.Share{Secret: "planted", Entropy: share.EntropyProject}

	if _, err := apps.Share.UpdateShare(alice.Context(), planted, "", shareapp.WithEncryptionPart(part)); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Errorf("update with raw part: err = %v, want %v", err, shareapp.ErrOTPVerificationRequired)
	}
	if _, err := apps.Share.UpdateShare(alice.Context(), planted, "", shareapp.WithEncryptionSession(bobsSession)); !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("update with bob's session: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}
	carol := apps.NewUser(t, p)
	err := apps.Share.RegisterShare(carol.Context(), &share.Share{Secret: "planted", Entropy: share.EntropyProject}, shareapp.WithEncryptionPart(part))
	if !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Errorf("register with raw part: err = %v, want %v", err, shareapp.ErrOTPVerificationRequired)
	}

	alicesSession := apps.OTPSession(t, p, part, alice.ExternalID, "alice@example.com")
	got, err := apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(alicesSession))
	if err != nil {
		t.Fatalf("alice reading her share: %v", err)
	}
	if got.Secret != "alice-share" {
		t.Errorf("alice's share secret = %q, want %q", got.Secret, "alice-share")
	}
}
