package apptest

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/pkg/contexter"

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
	if err := apps.Project.GenerateOTP(projectCtx, user.ExternalID, true, nil, nil, ""); err != nil {
		t.Fatalf("request signup session: %v", err)
	}
	session, err := apps.Project.RegisterEncryptionSession(projectCtx, part, user.ExternalID, nil, "")
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

func TestTwoFactor_RecoveryChangeRequiresVerifiedReadOfTheSameShare(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	user := apps.NewUser(t, p)
	reference := storeReferencedShare(t, apps, user, part, "original")
	legacyID := uuid.NewString()
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_shares (id, user_id, entropy, data, storage_method_id) VALUES (?, ?, 'USER', 'legacy', 0)`, legacyID, user.ID)
	previous, err := apps.ShareRepo.GetByReferenceForProjectAndUserID(user.Context(), reference, p.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	otherReference := "other-wallet"
	if err := apps.Share.RegisterShare(user.Context(), &share.Share{Secret: "other-share", Entropy: share.EntropyProject, Reference: &otherReference},
		shareapp.WithEncryptionPart(part), shareapp.AsPreregistration()); err != nil {
		t.Fatal(err)
	}
	ctx := user.Context()
	newShare := func() *share.Share {
		return &share.Share{Secret: "client-encrypted", Entropy: share.EntropyUser}
	}

	if _, err := apps.Share.UpdateShare(ctx, newShare(), reference); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Fatalf("changing protected recovery without approval: %v", err)
	}
	session := apps.OTPSession(t, p, part, user.ExternalID, "user@example.com")
	if _, err := apps.Share.GetShareByReference(ctx, reference, shareapp.WithEncryptionSession(session)); err != nil {
		t.Fatalf("read share with OTP: %v", err)
	}
	if _, err := apps.Share.GetShareByReference(ctx, reference, shareapp.WithEncryptionSession(session)); !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Fatalf("read again with consumed session: %v", err)
	}
	change := shareapp.WithRecoveryChangeSession(session)
	if _, err := apps.Share.UpdateShare(ctx, newShare(), otherReference, change); !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Fatalf("change a different share with the approval: %v", err)
	}
	if _, err := apps.Share.UpdateShare(ctx, newShare(), reference, change); err != nil {
		t.Fatalf("change recovery after verified read: %v", err)
	}
	if _, err := apps.Share.UpdateShare(ctx, newShare(), otherReference, change); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
		t.Fatalf("reuse recovery change approval: %v", err)
	}
	previous.ExpectedUpdate = &share.UpdateState{Entropy: previous.Entropy, Secret: previous.Secret}
	if err := apps.ShareRepo.Update(ctx, previous); !errors.Is(err, domainErrors.ErrShareChanged) {
		t.Fatalf("stale share update: %v", err)
	}
	got, err := apps.Share.GetShareByReference(ctx, reference)
	if err != nil || got.Entropy != share.EntropyUser || got.Secret != "client-encrypted" {
		t.Fatalf("read updated user-entropy share: %v", err)
	}
	if got := dbtest.ShareData(t, apps.DB, legacyID); got != "legacy" {
		t.Fatal("explicitly referenced update changed the legacy default share")
	}
}

func TestTwoFactor_BindsCodesAndSessionsToTheConfiguredProvider(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	user := apps.NewUser(t, p)
	apps.StoreShare(t, user, part, "original")
	otherProvider := uuid.NewString()
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_providers (id, project_id, type) VALUES (?, ?, 'OPENFORT')`, otherProvider, p.ID)
	projectCtx := ProjectContext(p)
	email := "user@example.com"

	if err := apps.Project.GenerateOTP(projectCtx, user.ExternalID, false, &email, nil, ""); !errors.Is(err, projectapp.ErrNoProviderSpecified) {
		t.Fatalf("ambiguous provider: %v", err)
	}
	if err := apps.Project.GenerateOTP(projectCtx, user.ExternalID, false, &email, nil, "custom"); err != nil {
		t.Fatalf("request OTP: %v", err)
	}
	code := apps.Notifier.Sent(email)[0]
	if _, err := apps.Project.RegisterEncryptionSession(projectCtx, part, user.ExternalID, &code, "openfort"); !errors.Is(err, projectapp.ErrOTPRequired) {
		t.Fatalf("code with different provider: %v", err)
	}
	session, err := apps.Project.RegisterEncryptionSession(projectCtx, part, user.ExternalID, &code, "custom")
	if err != nil {
		t.Fatalf("mint verified session: %v", err)
	}
	otherProviderCtx := contexter.WithProviderID(user.Context(), otherProvider)
	if _, err := apps.Share.GetShare(otherProviderCtx, shareapp.WithEncryptionSession(session)); !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Fatalf("session with different provider: %v", err)
	}
	if _, err := apps.Share.GetShare(user.Context(), shareapp.WithEncryptionSession(session)); err != nil {
		t.Fatalf("session with matching provider after rejected use: %v", err)
	}
}
