package apptest

import (
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/otp"
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

// External user IDs are only unique within a project.
func TestOTP_SameUserIDInTwoProjectsKeepsSeparateCodes(t *testing.T) {
	apps := New(t)
	p1, part1 := apps.Seed2FAProject(t)
	p2, part2 := apps.Seed2FAProject(t)
	u := apps.NewUser(t, p1)
	code1 := apps.RequestEmailOTP(t, p1, u.ExternalID, "user@example.com")
	code2 := apps.RequestEmailOTP(t, p2, u.ExternalID, "user@example.com")

	if _, err := apps.Project.RegisterEncryptionSession(ProjectContext(p1), part1, u.ExternalID, &code1); err != nil {
		t.Errorf("first project's code after the second project issued one: %v", err)
	}
	if _, err := apps.Project.RegisterEncryptionSession(ProjectContext(p2), part2, u.ExternalID, &code2); err != nil {
		t.Errorf("second project's code: %v", err)
	}
}

func TestOTP_FailedAttemptsInOneProjectDoNotLockOutAnother(t *testing.T) {
	apps := New(t)
	p1, _ := apps.Seed2FAProject(t)
	p2, part2 := apps.Seed2FAProject(t)
	u := apps.NewUser(t, p1)
	email, wrong := "user@example.com", "wrong-code"
	for range otp.DefaultSecurityConfig.MaxUserOnboardAttempts {
		apps.RequestEmailOTP(t, p2, u.ExternalID, email)
		for range otp.DefaultSecurityConfig.MaxFailedAttempts {
			_, _ = apps.Project.RegisterEncryptionSession(ProjectContext(p2), part2, u.ExternalID, &wrong)
		}
	}
	if err := apps.Project.GenerateOTP(ProjectContext(p2), u.ExternalID, false, &email, nil); !errors.Is(err, projectapp.ErrOTPRateLimitExceeded) {
		t.Fatalf("precondition: OTP in the project with the failures: err = %v, want %v", err, projectapp.ErrOTPRateLimitExceeded)
	}

	if err := apps.Project.GenerateOTP(ProjectContext(p1), u.ExternalID, false, &email, nil); err != nil {
		t.Errorf("OTP in another project: %v", err)
	}
}

// OTPs and sessions share one store. User IDs are caller-supplied, so an OTP
// request or check for a user ID equal to a session ID must not reach it.
func TestOTP_UserIDEqualToSessionIDLeavesSessionUsable(t *testing.T) {
	bogus := "000000000"
	actions := map[string]func(apps Apps, p dbtest.Project, part, session string){
		"request": func(apps Apps, p dbtest.Project, _, session string) {
			_ = apps.Project.GenerateOTP(ProjectContext(p), session, true, nil, nil)
		},
		"verify": func(apps Apps, p dbtest.Project, part, session string) {
			_, _ = apps.Project.RegisterEncryptionSession(ProjectContext(p), part, session, &bogus)
		},
	}
	for name, act := range actions {
		t.Run(name, func(t *testing.T) {
			apps := New(t)
			p, part := apps.Seed2FAProject(t)
			alice := apps.NewUser(t, p)
			apps.StoreShare(t, alice, part, "alice-share")
			session := apps.OTPSession(t, p, part, alice.ExternalID, "alice@example.com")

			act(apps, p, part, session)

			got, err := apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(session))
			if err != nil {
				t.Fatalf("GetShare with the session afterwards: %v", err)
			}
			if got.Secret != "alice-share" {
				t.Errorf("share secret = %q, want %q", got.Secret, "alice-share")
			}
		})
	}
}

func TestEncryptionSession_LookupByUserIDLeavesPendingOTPUsable(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice := apps.NewUser(t, p)
	code := apps.RequestEmailOTP(t, p, alice.ExternalID, "alice@example.com")

	err := apps.Share.RegisterShare(alice.Context(), &share.Share{Secret: "wallet-share", Entropy: share.EntropyProject},
		shareapp.WithEncryptionSession(alice.ExternalID))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("session ID equal to a user ID: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}

	if _, err := apps.Project.RegisterEncryptionSession(ProjectContext(p), part, alice.ExternalID, &code); err != nil {
		t.Errorf("pending OTP afterwards: %v", err)
	}
}
