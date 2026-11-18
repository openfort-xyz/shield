package apptest

import (
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// A session minted with one user's OTP must not stand in for another user's
// OTP, and the rejected attempt must not burn the session for its owner.
func TestEncryptionSession_OtherUsersOTPSessionIsRejectedAndKept(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice, bob := apps.NewUser(t, p), apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	apps.StoreShare(t, bob, part, "bob-share")
	bobsSession := apps.OTPSession(t, p, part, bob.ExternalID, "bob@example.com")

	_, err := apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(bobsSession))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("alice using bob's session: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}

	got, err := apps.Share.GetShare(bob.Context(), shareapp.WithEncryptionSession(bobsSession))
	if err != nil {
		t.Fatalf("bob using his own session after alice's attempt: %v", err)
	}
	if got.Secret != "bob-share" {
		t.Errorf("bob's share secret = %q, want %q", got.Secret, "bob-share")
	}
}

// An OTP verified in another project, even for the same external user ID,
// must not satisfy this project's OTP check.
func TestEncryptionSession_FromAnotherProjectIsRejected(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	other, _ := apps.Seed2FAProject(t)
	alice := apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	foreign := apps.OTPSession(t, other, part, alice.ExternalID, "alice@example.com")

	_, err := apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(foreign))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("session minted in another project: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}
}

// /admin/preregister authenticates the project, not a user, and registers
// shares with a session. The project binding holds there without 2FA too.
func TestEncryptionSession_FromAnotherProjectIsRejectedOnPreregister(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	other, _ := apps.SeedEncryptedProject(t)
	foreign, err := apps.Project.RegisterEncryptionSession(ProjectContext(other), part, "", nil)
	if err != nil {
		t.Fatalf("RegisterEncryptionSession: %v", err)
	}

	err = apps.Share.RegisterShare(UserContext(p), &share.Share{Secret: "wallet-share", Entropy: share.EntropyProject},
		shareapp.WithEncryptionSession(foreign))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("session minted in another project: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}
	if n := apps.ShareCount(t, p); n != 0 {
		t.Errorf("share stored with another project's session: %d share rows, want 0", n)
	}
}

func TestEncryptionSession_UnverifiedIsRejectedOn2FAProject(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice := apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	if err := apps.Project.GenerateOTP(ProjectContext(p), alice.ExternalID, true, nil, nil); err != nil {
		t.Fatalf("GenerateOTP skipping verification: %v", err)
	}
	session, err := apps.Project.RegisterEncryptionSession(ProjectContext(p), part, alice.ExternalID, nil)
	if err != nil {
		t.Fatalf("RegisterEncryptionSession: %v", err)
	}

	_, err = apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(session))
	if !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Errorf("unverified session: err = %v, want %v", err, shareapp.ErrOTPVerificationRequired)
	}
}

func TestEncryptionSession_WithoutUserWorksWithout2FA(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	alice := apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	session, err := apps.Project.RegisterEncryptionSession(ProjectContext(p), part, "", nil)
	if err != nil {
		t.Fatalf("RegisterEncryptionSession: %v", err)
	}

	got, err := apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(session))
	if err != nil {
		t.Fatalf("GetShare with a session minted without a user: %v", err)
	}
	if got.Secret != "alice-share" {
		t.Errorf("share secret = %q, want %q", got.Secret, "alice-share")
	}
}
