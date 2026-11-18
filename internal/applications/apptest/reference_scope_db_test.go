package apptest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/contexter"
)

// An external user ID is unique only within a provider, and a custom-auth
// project picks its own, so users in different projects can share one. One
// person signing in through two of a project's providers is two internal users
// with the same external user ID, and both must see the share.
func TestGetShareByReference_ScopedToCallersProjectAndIdentity(t *testing.T) {
	const subject = "shared-subject"
	apps := New(t)
	owner := dbtest.SeedProject(t, apps.DB, "", false)
	other := dbtest.SeedProject(t, apps.DB, "", false)

	linkIdentity(t, apps, owner.ID, owner.UserID, "OPENFORT", subject)
	linkedUserID := addUser(t, apps, owner.ID)
	linkIdentity(t, apps, owner.ID, linkedUserID, "CUSTOM", subject)
	strangerUserID := addUser(t, apps, owner.ID)
	linkIdentity(t, apps, owner.ID, strangerUserID, "CUSTOM", "another-subject")
	linkIdentity(t, apps, other.ID, other.UserID, "CUSTOM", subject)

	reference := "ref-" + uuid.NewString()
	err := apps.Share.RegisterShare(UserContext(owner),
		&share.Share{Secret: "wallet-share", Entropy: share.EntropyNone, Reference: &reference})
	if err != nil {
		t.Fatalf("RegisterShare: %v", err)
	}

	tests := []struct {
		name      string
		ctx       context.Context
		wantFound bool
	}{
		{"owner", identityContext(owner.ID, owner.UserID, subject), true},
		{"same subject through another provider of the project", identityContext(owner.ID, linkedUserID, subject), true},
		{"another subject in the same project", identityContext(owner.ID, strangerUserID, "another-subject"), false},
		{"same subject in another project", identityContext(other.ID, other.UserID, subject), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := apps.Share.GetShareByReference(tt.ctx, reference)
			if tt.wantFound {
				if err != nil {
					t.Fatalf("GetShareByReference: %v", err)
				}
				if got.Secret != "wallet-share" {
					t.Errorf("secret = %q, want %q", got.Secret, "wallet-share")
				}
				return
			}
			if !errors.Is(err, shareapp.ErrShareNotFound) {
				t.Errorf("err = %v, want %v", err, shareapp.ErrShareNotFound)
			}
			if got != nil {
				t.Errorf("returned a share the caller does not own (secret %q)", got.Secret)
			}
		})
	}
}

func addUser(t *testing.T, apps Apps, projectID string) string {
	t.Helper()
	id := uuid.NewString()
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_users (id, project_id) VALUES (?, ?)`, id, projectID)
	return id
}

func linkIdentity(t *testing.T, apps Apps, projectID, userID, providerType, externalUserID string) {
	t.Helper()
	providerID := uuid.NewString()
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_providers (id, project_id, type) VALUES (?, ?, ?)`, providerID, projectID, providerType)
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_external_users (id, user_id, external_user_id, provider_id) VALUES (?, ?, ?, ?)`,
		uuid.NewString(), userID, externalUserID, providerID)
}

// identityContext is what AuthenticateUser puts in the request context.
func identityContext(projectID, userID, externalUserID string) context.Context {
	ctx := contexter.WithProjectID(context.Background(), projectID)
	ctx = contexter.WithUserID(ctx, userID)
	return contexter.WithExternalUserID(ctx, externalUserID)
}
