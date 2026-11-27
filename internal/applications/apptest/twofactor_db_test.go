package apptest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/contexter"
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

// The same holds on the keychain and by-reference reads.
func TestEncryptionSession_OtherUsersOTPSessionIsRejectedAndKeptOnOtherReads(t *testing.T) {
	reads := map[string]func(apps Apps, u User, ref, session string) (string, error){
		"keychain": func(apps Apps, u User, _, session string) (string, error) {
			return onlySecret(apps.Share.GetKeychainShares(u.Context(), nil, shareapp.WithEncryptionSession(session)))
		},
		"keychain share by reference": func(apps Apps, u User, ref, session string) (string, error) {
			return onlySecret(apps.Share.GetKeychainShares(u.Context(), &ref, shareapp.WithEncryptionSession(session)))
		},
		"share by reference": func(apps Apps, u User, ref, session string) (string, error) {
			shr, err := apps.Share.GetShareByReference(u.Context(), ref, shareapp.WithEncryptionSession(session))
			if err != nil {
				return "", err
			}
			return shr.Secret, nil
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			apps := New(t)
			p, part := apps.Seed2FAProject(t)
			alice, bob := apps.NewUser(t, p), apps.NewUser(t, p)
			aliceRef := storeReferencedShare(t, apps, alice, part, "alice-share")
			bobRef := storeReferencedShare(t, apps, bob, part, "bob-share")
			bobsSession := apps.OTPSession(t, p, part, bob.ExternalID, "bob@example.com")

			if _, err := read(apps, alice, aliceRef, bobsSession); !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
				t.Errorf("alice using bob's session: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
			}

			got, err := read(apps, bob, bobRef, bobsSession)
			if err != nil {
				t.Fatalf("bob using his own session after alice's attempt: %v", err)
			}
			if got != "bob-share" {
				t.Errorf("bob's share secret = %q, want %q", got, "bob-share")
			}
		})
	}
}

// storeReferencedShare registers a project-encrypted share for the user under
// a reference no other share has, and returns the reference.
func storeReferencedShare(t *testing.T, apps Apps, u User, part, secret string) string {
	t.Helper()
	ref := "ref-" + u.ExternalID
	err := apps.Share.RegisterShare(u.Context(), &share.Share{Secret: secret, Entropy: share.EntropyProject, Reference: &ref},
		shareapp.WithEncryptionPart(part), shareapp.AsPreregistration())
	if err != nil {
		t.Fatalf("RegisterShare: %v", err)
	}
	return ref
}

func onlySecret(shrs []*share.Share, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if len(shrs) != 1 {
		return "", fmt.Errorf("got %d shares, want 1", len(shrs))
	}
	return shrs[0].Secret, nil
}

// Presenting a session in another project must not burn it for its own.
func TestEncryptionSession_RejectedInAnotherProjectStaysUsableInItsOwn(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	other, otherPart := apps.SeedEncryptedProject(t)
	alice, bob := apps.NewUser(t, p), apps.NewUser(t, other)
	apps.StoreShare(t, alice, part, "alice-share")
	apps.StoreShare(t, bob, otherPart, "bob-share")
	bobsSession, err := apps.Project.RegisterEncryptionSession(ProjectContext(other), otherPart, bob.ExternalID, nil)
	if err != nil {
		t.Fatalf("RegisterEncryptionSession: %v", err)
	}

	_, err = apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(bobsSession))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("session minted in another project: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}

	got, err := apps.Share.GetShare(bob.Context(), shareapp.WithEncryptionSession(bobsSession))
	if err != nil {
		t.Fatalf("bob using his session in its own project afterwards: %v", err)
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

// The Openfort API names a third-party user by two IDs: the one shares are
// stored under, and the one SDKs hand the project backend for OTP sessions.
func TestEncryptionSession_MintedForAltUserIDIsAccepted(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice, bob := apps.NewUser(t, p), apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	apps.StoreShare(t, bob, part, "bob-share")
	alicesAltID := "usr_" + alice.ExternalID
	t.Cleanup(func() {
		if err := apps.DB.Exec(`DELETE FROM shld_user_contacts WHERE external_user_id = ?`, alicesAltID).Error; err != nil {
			t.Errorf("cleanup user contacts: %v", err)
		}
	})
	session := apps.OTPSession(t, p, part, alicesAltID, "alice@example.com")

	bobWithOwnAltID := contexter.WithAltExternalUserID(bob.Context(), "usr_"+bob.ExternalID)
	_, err := apps.Share.GetShare(bobWithOwnAltID, shareapp.WithEncryptionSession(session))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("bob using alice's session: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}
	_, err = apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionSession(session))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionSession) {
		t.Errorf("alice without her alt ID: err = %v, want %v", err, shareapp.ErrInvalidEncryptionSession)
	}

	aliceWithAltID := contexter.WithAltExternalUserID(alice.Context(), alicesAltID)
	got, err := apps.Share.GetShare(aliceWithAltID, shareapp.WithEncryptionSession(session))
	if err != nil {
		t.Fatalf("alice using the session minted for her alt ID: %v", err)
	}
	if got.Secret != "alice-share" {
		t.Errorf("share secret = %q, want %q", got.Secret, "alice-share")
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

func TestEncryptionPart_IsRejectedOnReadsOf2FAProject(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice := apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")
	withPart := shareapp.WithEncryptionPart(part)

	reads := map[string]func() error{
		"GetShare": func() error {
			_, err := apps.Share.GetShare(alice.Context(), withPart)
			return err
		},
		"GetKeychainShares": func() error {
			_, err := apps.Share.GetKeychainShares(alice.Context(), nil, withPart)
			return err
		},
	}
	for name, read := range reads {
		if err := read(); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
			t.Errorf("%s with a raw part: err = %v, want %v", name, err, shareapp.ErrOTPVerificationRequired)
		}
	}
}

func TestEncryptionPart_WorksOnReadsWithout2FA(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	alice := apps.NewUser(t, p)
	apps.StoreShare(t, alice, part, "alice-share")

	got, err := apps.Share.GetShare(alice.Context(), shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("GetShare with a raw part: %v", err)
	}
	if got.Secret != "alice-share" {
		t.Errorf("share secret = %q, want %q", got.Secret, "alice-share")
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

// The first contact an OTP goes to binds the user. A request naming any other
// contact must get no OTP: none delivered, and none replacing the pending one.
func TestOTP_MismatchedContactGetsNoOTP(t *testing.T) {
	email, phone := "attacker@example.com", "+346000000000"
	cases := []struct {
		name         string
		email, phone *string
		to           string
	}{
		{"another email", &email, nil, email},
		{"a phone instead of the bound email", nil, &phone, phone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apps := New(t)
			p, part := apps.Seed2FAProject(t)
			alice := apps.NewUser(t, p)
			code := apps.RequestEmailOTP(t, p, alice.ExternalID, "alice@example.com")

			err := apps.Project.GenerateOTP(ProjectContext(p), alice.ExternalID, false, tc.email, tc.phone)
			if !errors.Is(err, projectapp.ErrUserContactInformationMismatch) {
				t.Errorf("GenerateOTP to %s: err = %v, want %v", tc.to, err, projectapp.ErrUserContactInformationMismatch)
			}
			if sent := apps.Notifier.Sent(tc.to); len(sent) != 0 {
				t.Errorf("%d messages delivered to %s, want 0", len(sent), tc.to)
			}
			if _, err := apps.Project.RegisterEncryptionSession(ProjectContext(p), part, alice.ExternalID, &code); err != nil {
				t.Errorf("alice's pending OTP after the rejected request: %v", err)
			}
		})
	}
}

var errProviderRejected = errors.New("provider rejected the destination")

func byEmail(to string) (email, phone *string) { return &to, nil }
func byPhone(to string) (email, phone *string) { return nil, &to }

// A first OTP request whose delivery fails, such as to a mistyped phone number
// the provider rejects, must not bind its contact: a retry with the corrected
// contact gets an OTP there.
func TestOTP_FailedFirstDeliveryLeavesContactUnbound(t *testing.T) {
	cases := []struct {
		name, typo, fixed string
		via               func(to string) (email, phone *string)
	}{
		{"email", "alice@exmaple.com", "alice@example.com", byEmail},
		{"phone", "+346000000001", "+346000000002", byPhone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apps := New(t)
			p, part := apps.Seed2FAProject(t)
			alice := apps.NewUser(t, p)
			ctx := ProjectContext(p)

			apps.Notifier.Fail = errProviderRejected
			email, phone := tc.via(tc.typo)
			if err := apps.Project.GenerateOTP(ctx, alice.ExternalID, false, email, phone); !errors.Is(err, errProviderRejected) {
				t.Fatalf("GenerateOTP to %s: err = %v, want %v", tc.typo, err, errProviderRejected)
			}
			apps.Notifier.Fail = nil

			email, phone = tc.via(tc.fixed)
			if err := apps.Project.GenerateOTP(ctx, alice.ExternalID, false, email, phone); err != nil {
				t.Fatalf("GenerateOTP to the corrected contact %s: %v", tc.fixed, err)
			}
			sent := apps.Notifier.Sent(tc.fixed)
			if len(sent) != 1 {
				t.Fatalf("%d messages delivered to %s, want 1", len(sent), tc.fixed)
			}
			if _, err := apps.Project.RegisterEncryptionSession(ctx, part, alice.ExternalID, &sent[0]); err != nil {
				t.Errorf("OTP delivered to the corrected contact: %v", err)
			}
		})
	}
}

// A request whose delivery failed issued no OTP: the code must not verify even
// if the provider sent it before erroring.
func TestOTP_FailedDeliveryLeavesNoPendingOTP(t *testing.T) {
	apps := New(t)
	p, part := apps.Seed2FAProject(t)
	alice := apps.NewUser(t, p)
	email := "alice@example.com"

	apps.Notifier.Fail = errProviderRejected
	if err := apps.Project.GenerateOTP(ProjectContext(p), alice.ExternalID, false, &email, nil); !errors.Is(err, errProviderRejected) {
		t.Fatalf("GenerateOTP: err = %v, want %v", err, errProviderRejected)
	}
	sent := apps.Notifier.Sent(email)
	if len(sent) != 1 {
		t.Fatalf("%d messages handed to the provider for %s, want 1", len(sent), email)
	}

	_, err := apps.Project.RegisterEncryptionSession(ProjectContext(p), part, alice.ExternalID, &sent[0])
	if !errors.Is(err, projectapp.ErrOTPRequired) {
		t.Errorf("code from the failed request: err = %v, want %v", err, projectapp.ErrOTPRequired)
	}
}

// A failed delivery to the bound contact must leave the binding in place.
func TestOTP_FailedDeliveryKeepsExistingContact(t *testing.T) {
	apps := New(t)
	p, _ := apps.Seed2FAProject(t)
	alice := apps.NewUser(t, p)
	email, other := "alice@example.com", "attacker@example.com"
	apps.RequestEmailOTP(t, p, alice.ExternalID, email)

	apps.Notifier.Fail = errProviderRejected
	if err := apps.Project.GenerateOTP(ProjectContext(p), alice.ExternalID, false, &email, nil); !errors.Is(err, errProviderRejected) {
		t.Fatalf("GenerateOTP to the bound contact: err = %v, want %v", err, errProviderRejected)
	}
	apps.Notifier.Fail = nil

	err := apps.Project.GenerateOTP(ProjectContext(p), alice.ExternalID, false, &other, nil)
	if !errors.Is(err, projectapp.ErrUserContactInformationMismatch) {
		t.Errorf("GenerateOTP to %s: err = %v, want %v", other, err, projectapp.ErrUserContactInformationMismatch)
	}
	if sent := apps.Notifier.Sent(other); len(sent) != 0 {
		t.Errorf("%d messages delivered to %s, want 0", len(sent), other)
	}
}
