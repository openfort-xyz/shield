package apptest

import (
	"context"
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/contexter"
)

// Share routes authenticate through authmdw, which puts the project from the
// API-key cache into the request context. Once 2FA is enabled, an encryption
// session registered without an OTP must be refused even while that cache
// entry still says 2FA is off.
func TestEnable2FA_CachedProjectDoesNotSkipOTP(t *testing.T) {
	tests := []struct {
		name string
		read func(apps Apps, ctx context.Context, session string) error
	}{
		{"GetShare", func(apps Apps, ctx context.Context, session string) error {
			_, err := apps.Share.GetShare(ctx, shareapp.WithEncryptionSession(session))
			return err
		}},
		{"GetKeychainShares", func(apps Apps, ctx context.Context, session string) error {
			_, err := apps.Share.GetKeychainShares(ctx, nil, shareapp.WithEncryptionSession(session))
			return err
		}},
		{"GetKeychainShares by reference", func(apps Apps, ctx context.Context, session string) error {
			_, err := apps.Share.GetKeychainShares(ctx, ptr(share.DefaultReference), shareapp.WithEncryptionSession(session))
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := New(t)
			p, part := apps.SeedEncryptedProject(t)
			apps.register(t, p, part, "wallet", "")
			session, err := apps.Project.RegisterEncryptionSession(ProjectContext(p), part, "external-user", nil, "")
			if err != nil {
				t.Fatalf("RegisterEncryptionSession: %v", err)
			}
			if _, err := apps.ProjectSvc.GetByAPIKey(context.Background(), p.APIKey); err != nil {
				t.Fatalf("GetByAPIKey before enabling 2FA: %v", err)
			}

			if err := apps.Project.Enable2FA(ProjectContext(p)); err != nil {
				t.Fatalf("Enable2FA: %v", err)
			}
			cached, err := apps.ProjectSvc.GetByAPIKey(context.Background(), p.APIKey)
			if err != nil {
				t.Fatalf("GetByAPIKey after enabling 2FA: %v", err)
			}
			ctx := contexter.WithExternalUserID(contexter.WithProject(UserContext(p), cached), "external-user")

			if err := tt.read(apps, ctx, session); !errors.Is(err, shareapp.ErrOTPVerificationRequired) {
				t.Errorf("read with a session registered before 2FA: err = %v, want %v", err, shareapp.ErrOTPVerificationRequired)
			}
		})
	}
}
