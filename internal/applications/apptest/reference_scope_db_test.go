package apptest

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/contexter"
)

// An external user ID is unique only within a provider, and a custom-auth
// project picks its own, so other users can carry the same one. Only the
// internal user that stored the share may read it.
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
		{"same subject through another provider of the project", identityContext(owner.ID, linkedUserID, subject), false},
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

// Every user registering without a reference gets the default one, so a project
// holds many shares under it. The lookup takes the first match by ID; the other
// project's share is given the lowest ID so a missing project filter returns it.
func TestGetShareByReference_SameReferenceAcrossUsers_ReturnsCallersShare(t *testing.T) {
	apps := New(t)
	owner := dbtest.SeedProject(t, apps.DB, "", false)
	other := dbtest.SeedProject(t, apps.DB, "", false)

	linkIdentity(t, apps, owner.ID, owner.UserID, "CUSTOM", "subject-a")
	secondUserID := addUser(t, apps, owner.ID)
	linkIdentity(t, apps, owner.ID, secondUserID, "CUSTOM", "subject-b")
	linkIdentity(t, apps, other.ID, other.UserID, "CUSTOM", "subject-a")

	callers := []struct {
		name string
		ctx  context.Context
	}{
		{"subject-a", identityContext(owner.ID, owner.UserID, "subject-a")},
		{"subject-b", identityContext(owner.ID, secondUserID, "subject-b")},
	}
	for _, c := range callers {
		err := apps.Share.RegisterShare(c.ctx, &share.Share{Secret: "share-of-" + c.name, Entropy: share.EntropyNone})
		if err != nil {
			t.Fatalf("RegisterShare as %s: %v", c.name, err)
		}
	}
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_shares (id, data, user_id, keychain_id, reference, entropy, storage_method_id) VALUES (?, ?, ?, ?, ?, 'none', 0)`,
		"00000000"+uuid.NewString()[8:], "share-of-other-project", other.UserID, other.KeychainID, share.DefaultReference)

	for _, c := range callers {
		t.Run(c.name, func(t *testing.T) {
			got, err := apps.Share.GetShareByReference(c.ctx, share.DefaultReference)
			if err != nil {
				t.Fatalf("GetShareByReference: %v", err)
			}
			if want := "share-of-" + c.name; got.Secret != want {
				t.Errorf("secret = %q, want %q", got.Secret, want)
			}
		})
	}
}

func TestRegisterShare_ReferenceIsUniquePerUser(t *testing.T) {
	apps := New(t)
	owner := dbtest.SeedProject(t, apps.DB, "", false)
	other := dbtest.SeedProject(t, apps.DB, "", false)
	sameProject := owner
	sameProject.UserID = addUser(t, apps, owner.ID)

	reference := "ref-" + uuid.NewString()
	for i, p := range []dbtest.Project{owner, owner, sameProject, other} {
		ref := reference
		err := apps.Share.RegisterShare(UserContext(p),
			&share.Share{Secret: "wallet-share", Entropy: share.EntropyNone, Reference: &ref})
		if i == 1 {
			if !errors.Is(err, shareapp.ErrShareAlreadyExists) {
				t.Fatalf("duplicate registration: got %v, want conflict", err)
			}
		} else if err != nil {
			t.Fatalf("RegisterShare #%d: %v", i+1, err)
		}
	}

	for _, p := range []dbtest.Project{owner, sameProject, other} {
		if n := apps.ShareCount(t, p); n != 1 {
			t.Errorf("shares for user %q = %d, want 1", p.UserID, n)
		}
	}
}

func TestRegisterShare_ConcurrentDefaultReferences(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", false)
	const registrations = 12
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, registrations)
	for i := range registrations {
		wg.Go(func() {
			<-start
			shr := &share.Share{Secret: "wallet-share", Entropy: share.EntropyNone}
			if i%2 == 0 {
				shr.Reference = ptr(share.DefaultReference)
			}
			results <- apps.Share.RegisterShare(UserContext(p), shr)
		})
	}
	close(start)
	wg.Wait()
	close(results)
	var successes int
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, shareapp.ErrShareAlreadyExists) {
			t.Fatalf("register: %v", err)
		}
	}
	if successes != 1 || apps.ShareCount(t, p) != 1 {
		t.Fatalf("successful registrations: %d, want 1", successes)
	}
}

func TestLegacyDuplicateShares_ReadsAgreeOnTheOriginal(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", false)
	ctx := UserContext(p)
	ref := share.DefaultReference

	// Simulate two rows written before duplicate protection. The second row
	// sorts first by ID, so it cannot accidentally pass the oldest-row check.
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_shares
		(id, data, user_id, keychain_id, reference, entropy, storage_method_id, created_at) VALUES
		('ffffffff-ffff-4fff-8fff-000000000001', 'original', ?, ?, ?, 'none', 0, now() - INTERVAL '2 hours'),
		('00000000-0000-4000-8000-000000000002', 'later', ?, ?, ?, 'none', 0, now() - INTERVAL '1 hour')`,
		p.UserID, p.KeychainID, ref, p.UserID, p.KeychainID, ref)

	got, err := apps.Share.GetShare(ctx)
	if err != nil || got.Secret != "original" {
		t.Fatalf("default read did not return the original share: %v", err)
	}
	got, err = apps.Share.GetShareByReference(ctx, ref)
	if err != nil || got.Secret != "original" {
		t.Fatalf("reference read did not return the original share: %v", err)
	}
	for _, reference := range []*string{nil, &ref} {
		shares, err := apps.Share.GetKeychainShares(ctx, reference)
		if err != nil || len(shares) != 1 || shares[0].Secret != "original" {
			t.Fatalf("keychain read did not return a single original share: %v", err)
		}
	}
	got, err = apps.Share.ExportShare(ctx, ref)
	if err != nil || got.Secret != "original" {
		t.Fatalf("export did not return the recovery share: %v", err)
	}
	if apps.ShareCount(t, p) != 2 {
		t.Fatal("reading must preserve the existing rows")
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

func TestExportShare_NormalizesLegacyReferencesAndRejectsAmbiguousOwners(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", false)
	other := dbtest.SeedProject(t, apps.DB, "", false)
	ctx := UserContext(p)
	for _, owner := range []dbtest.Project{p, other} {
		dbtest.Exec(t, apps.DB, `INSERT INTO shld_shares
            (id, data, user_id, keychain_id, reference, entropy, storage_method_id, created_at) VALUES
            (?, 'original', ?, ?, NULL, 'none', 0, now() - INTERVAL '2 hours'),
            (?, 'later', ?, ?, '', 'none', 0, now() - INTERVAL '1 hour')`,
			uuid.NewString(), owner.UserID, owner.KeychainID, uuid.NewString(), owner.UserID, owner.KeychainID)
	}
	for _, ref := range []string{"", share.DefaultReference} {
		recovered, err := apps.Share.GetShareByReference(ctx, ref)
		if err != nil {
			t.Fatal(err)
		}
		exported, err := apps.Share.ExportShare(ctx, ref)
		if err != nil || exported.ID != recovered.ID || exported.Secret != "original" {
			t.Fatalf("export and recovery disagree for %q: %v", ref, err)
		}
	}
	stranger := addUser(t, apps, p.ID)
	dbtest.Exec(t, apps.DB, `INSERT INTO shld_shares
        (id, data, user_id, reference, entropy, storage_method_id) VALUES (?, 'stranger', ?, 'default', 'none', 0)`,
		uuid.NewString(), stranger)
	exported, err := apps.Share.ExportShare(ctx, share.DefaultReference)
	if exported != nil || !errors.Is(err, shareapp.ErrAmbiguousShareReference) {
		t.Fatalf("ambiguous export returned a share: %v", err)
	}
	// Ambiguous project exports must not prevent user-scoped recovery.
	recovered, err := apps.Share.GetShareByReference(ctx, share.DefaultReference)
	if err != nil || recovered.Secret != "original" {
		t.Fatalf("recovery changed: %v", err)
	}
	dbtest.Exec(t, apps.DB, `UPDATE shld_shares SET deleted_at = now() WHERE user_id = ?`, stranger)
	exported, err = apps.Share.ExportShare(ctx, share.DefaultReference)
	if err != nil || exported.ID != recovered.ID {
		t.Fatalf("deleted share made export ambiguous: %v", err)
	}
}
