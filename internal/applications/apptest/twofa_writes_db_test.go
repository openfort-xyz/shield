package apptest

import (
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// On 2FA projects, share writes take the caller's OTP-verified session, as
// share reads do.
func TestTwoFactor_ShareWritesUseTheCallersOTPSession(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice, bob := apps.NewUser(t, p), apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	bobsSession := apps.OTPSession(t, p, part, bob.ExternalID, "bob@example.com")
	updated := &share.Share{Secret: "updated", Entropy: share.EntropyProject}

	if _, err := apps.Share.UpdateShare(alice.Context(), updated, share.DefaultReference, shareapp.WithEncryptionPart(part)); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Errorf("update with raw part: err = %v, want %v", err, shareapp.ErrOTPVerificationRequired)
	}
	if _, err := apps.Share.UpdateShare(alice.Context(), updated, share.DefaultReference, shareapp.WithEncryptionSession(bobsSession)); !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("update with bob's session: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}
	carol := apps.NewUser(t, p)
	err := apps.Share.RegisterShare(carol.Context(), &share.Share{Secret: "updated", Entropy: share.EntropyProject}, shareapp.WithEncryptionPart(part))
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

func TestTwoFactor_SignupSessionOnlyRegistersShares(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	user := apps.NewUser(t, p)
	apps.StoreShare(t, user, part, "existing-share")

	projectCtx := ProjectContext(p)
	if err := apps.Project.GenerateOTP(projectCtx, user.ExternalID, true, nil, nil); err != nil {
		t.Fatalf("request signup session: %v", err)
	}
	session, err := apps.Project.RegisterEncryptionSession(projectCtx, part, user.ExternalID, nil)
	if err != nil {
		t.Fatalf("create signup session: %v", err)
	}

	ctx := user.Context()
	withSession := shareapp.WithEncryptionSession(session)
	if _, err := apps.Share.GetShare(ctx, withSession); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Fatalf("read with signup session: got %v, want %v", err, shareapp.ErrOTPVerificationRequired)
	}
	if _, err := apps.Share.UpdateShare(ctx, &share.Share{Secret: "updated", Entropy: share.EntropyProject}, share.DefaultReference, withSession); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Fatalf("update with signup session: got %v, want %v", err, shareapp.ErrOTPVerificationRequired)
	}

	ref := "new-wallet"
	if err := apps.Share.RegisterShare(ctx, &share.Share{Secret: "new-share", Entropy: share.EntropyProject, Reference: &ref}, withSession); err != nil {
		t.Fatalf("register with signup session: %v", err)
	}
	ref = "another-wallet"
	if err := apps.Share.RegisterShare(ctx, &share.Share{Secret: "another-share", Entropy: share.EntropyProject, Reference: &ref}, withSession); !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Fatalf("reuse signup session: got %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}

	verified := apps.OTPSession(t, p, part, user.ExternalID, "user@example.com")
	got, err := apps.Share.GetShareByReference(ctx, "new-wallet", shareapp.WithEncryptionSession(verified))
	if err != nil || got.Secret != "new-share" {
		t.Fatalf("read newly registered share: got %v, err %v", got, err)
	}
}
