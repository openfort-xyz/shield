package apptest

import (
	"errors"
	"fmt"
	"maps"
	"testing"

	depsssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/deprecated_sss_reconstruction_strategy"
	sssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/random"
)

// seedLegacyProject seeds a project whose key predates key checks: a stored
// part and no check. It returns the project's part and the key. An
// unmigrated project's key is whatever the pre-migration combine makes of
// its two parts.
func seedLegacyProject(t *testing.T, apps Apps, migrated bool) (dbtest.Project, string, string) {
	t.Helper()
	var storedPart project.StoredPart
	var projectPart project.ProjectPart
	var key string
	var err error
	if migrated {
		if key, err = random.GenerateRandomString(32); err != nil {
			t.Fatalf("generate key: %v", err)
		}
		storedPart, projectPart, err = sssrec.NewSSSReconstructionStrategy().Split(key)
	} else {
		storedPart, projectPart = project.StoredPart(randomPart(t)), project.ProjectPart(randomPart(t))
		key, err = depsssrec.NewSSSReconstructionStrategy().Reconstruct(storedPart, projectPart)
	}
	if err != nil {
		t.Fatalf("legacy key: %v", err)
	}
	return dbtest.SeedProject(t, apps.DB, string(storedPart), migrated), string(projectPart), key
}

func randomPart(t *testing.T) string {
	t.Helper()
	part, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate part: %v", err)
	}
	return part
}

// insertProjectShare stores a project-entropy share encrypted under key, the
// way a write before key checks existed left it.
func insertProjectShare(t *testing.T, apps Apps, p dbtest.Project, key, secret string) string {
	t.Helper()
	ciphertext, err := cypher.Encrypt(secret, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return dbtest.InsertShare(t, apps.DB, p, "project", ciphertext)
}

func storedKeyCheck(t *testing.T, apps Apps, p dbtest.Project) *string {
	t.Helper()
	var check *string
	if err := apps.DB.Raw(`SELECT key_check FROM shld_encryption_parts WHERE project_id = ?`, p.ID).Scan(&check).Error; err != nil {
		t.Fatalf("read key check: %v", err)
	}
	return check
}

func storedPart(t *testing.T, apps Apps, p dbtest.Project) project.StoredPart {
	t.Helper()
	var part string
	if err := apps.DB.Raw(`SELECT part FROM shld_encryption_parts WHERE project_id = ? ORDER BY id LIMIT 1`, p.ID).Scan(&part).Error; err != nil {
		t.Fatalf("read stored part: %v", err)
	}
	return project.StoredPart(part)
}

func isMigrated(t *testing.T, apps Apps, p dbtest.Project) bool {
	t.Helper()
	var n int64
	if err := apps.DB.Raw(`SELECT count(*) FROM shld_shamir_migrations WHERE project_id = ? AND success`, p.ID).Scan(&n).Error; err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	return n > 0
}

// projectShareData maps each share of the project's user, soft-deleted ones
// included, to its stored data.
func projectShareData(t *testing.T, apps Apps, p dbtest.Project) map[string]string {
	t.Helper()
	var rows []struct{ ID, Data string }
	if err := apps.DB.Raw(`SELECT id, data FROM shld_shares WHERE user_id = ?`, p.UserID).Scan(&rows).Error; err != nil {
		t.Fatalf("read shares: %v", err)
	}
	data := make(map[string]string, len(rows))
	for _, row := range rows {
		data[row.ID] = row.Data
	}
	return data
}

// assertSharesDecrypt fails unless the project has want project-entropy
// shares, soft-deleted ones included, and each decrypts under the key that
// the project's recorded migration status makes of its parts: the key every
// later request reconstructs.
func assertSharesDecrypt(t *testing.T, apps Apps, p dbtest.Project, projectPart string, want int) {
	t.Helper()
	migrated := isMigrated(t, apps, p)
	reconstruct := depsssrec.NewSSSReconstructionStrategy().Reconstruct
	if migrated {
		reconstruct = sssrec.NewSSSReconstructionStrategy().Reconstruct
	}
	key, err := reconstruct(storedPart(t, apps, p), project.ProjectPart(projectPart))
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}

	var rows []struct{ ID, Data string }
	err = apps.DB.Raw(`SELECT s.id, s.data FROM shld_shares s JOIN shld_users u ON s.user_id = u.id
		WHERE u.project_id = ? AND s.entropy = 'project'`, p.ID).Scan(&rows).Error
	if err != nil {
		t.Fatalf("read shares: %v", err)
	}
	if len(rows) != want {
		t.Errorf("project shares = %d, want %d", len(rows), want)
	}
	for _, row := range rows {
		if _, err := cypher.Decrypt(row.Data, key); err != nil {
			t.Errorf("share %s does not decrypt under the project's key (migrated = %v): %v", row.ID, migrated, err)
		}
	}
}

func shareReference(t *testing.T, apps Apps, shareID string) string {
	t.Helper()
	var reference string
	if err := apps.DB.Raw(`SELECT reference FROM shld_shares WHERE id = ?`, shareID).Scan(&reference).Error; err != nil {
		t.Fatalf("read reference: %v", err)
	}
	return reference
}

// A key without a check cannot be verified by Build, and Shamir combine
// accepts a part from another split. Writing under the resulting key would
// store ciphertext the project's part cannot decrypt.
func TestLegacyKey_WrongPartWritesRejected(t *testing.T) {
	apps := New(t)
	_, otherPart := apps.SeedEncryptedProject(t)

	tests := []struct {
		name  string
		write func(p dbtest.Project, projectShareID, plainShareID string) error
	}{
		{"RegisterShare", func(p dbtest.Project, _, _ string) error {
			return apps.Share.RegisterShare(UserContext(p),
				&share.Share{Secret: "new", Entropy: share.EntropyProject, Reference: ptr("new-" + p.ID[:8])},
				shareapp.WithEncryptionPart(otherPart))
		}},
		{"UpdateShare", func(p dbtest.Project, projectShareID, _ string) error {
			_, err := apps.Share.UpdateShare(UserContext(p), &share.Share{Secret: "new"},
				shareReference(t, apps, projectShareID), shareapp.WithEncryptionPart(otherPart))
			return err
		}},
		{"EncryptProjectShares", func(p dbtest.Project, _, _ string) error {
			return apps.Project.EncryptProjectShares(ProjectContext(p), otherPart)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _, key := seedLegacyProject(t, apps, true)
			projectShareID := insertProjectShare(t, apps, p, key, "wallet")
			projectShareData := dbtest.ShareData(t, apps.DB, projectShareID)
			plainShareID := dbtest.InsertShare(t, apps.DB, p, "none", "plain-share")

			err := tt.write(p, projectShareID, plainShareID)

			if !errors.Is(err, shareapp.ErrInvalidEncryptionPart) && !errors.Is(err, projectapp.ErrInvalidEncryptionPart) {
				t.Errorf("write with a wrong part: err = %v, want invalid encryption part", err)
			}
			if n := apps.ShareCount(t, p); n != 2 {
				t.Errorf("share rows = %d, want 2", n)
			}
			if got := dbtest.ShareData(t, apps.DB, projectShareID); got != projectShareData {
				t.Error("project share rewritten under a wrong key")
			}
			if got := dbtest.ShareData(t, apps.DB, plainShareID); got != "plain-share" {
				t.Errorf("plain share = %q, want it untouched", got)
			}
			if check := storedKeyCheck(t, apps, p); check != nil {
				t.Errorf("key check stored from a wrong part: %q", *check)
			}
		})
	}
}

// A write with the right part proves the key against the existing shares and
// stores its check, so later writes verify against the check.
func TestLegacyKey_CorrectPartWriteStoresKeyCheck(t *testing.T) {
	apps := New(t)
	p, part, key := seedLegacyProject(t, apps, true)
	insertProjectShare(t, apps, p, key, "wallet")

	err := apps.Share.RegisterShare(UserContext(p),
		&share.Share{Secret: "second", Entropy: share.EntropyProject, Reference: ptr("second-" + p.ID[:8])},
		shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("RegisterShare with the right part: %v", err)
	}

	want, err := cypher.KeyCheck(key)
	if err != nil {
		t.Fatalf("key check: %v", err)
	}
	if check := storedKeyCheck(t, apps, p); check == nil || *check != want {
		t.Errorf("key check after a verified write = %v, want the key's check", check)
	}
	shrs, err := apps.Share.GetKeychainShares(UserContext(p), ptr("second-"+p.ID[:8]), shareapp.WithEncryptionPart(part))
	if err != nil || len(shrs) != 1 || shrs[0].Secret != "second" {
		t.Errorf("new share under the right part = %v, %v; want %q", shrs, err, "second")
	}
}

// With no project share there is nothing to prove a part against, so the
// first write decides the key. A second first write racing it with another
// part must not leave the project with shares under two keys.
func TestLegacyKey_RacingFirstWritesDoNotMixKeys(t *testing.T) {
	parking := NewParkingShareRepo()
	apps := NewWithShareRepo(t, parking.Wrap)
	_, otherPart := apps.SeedEncryptedProject(t)
	p, part, _ := seedLegacyProject(t, apps, true)
	ctx := UserContext(p)

	parking.Armed.Store(true)
	errc := make(chan error, 1)
	go func() {
		errc <- apps.Share.RegisterShare(ctx,
			&share.Share{Secret: "wrong", Entropy: share.EntropyProject, Reference: ptr("wrong-" + p.ID[:8])},
			shareapp.WithEncryptionPart(otherPart))
	}()
	<-parking.Parked
	parking.Armed.Store(false)
	err := apps.Share.RegisterShare(ctx,
		&share.Share{Secret: "right", Entropy: share.EntropyProject, Reference: ptr("right-" + p.ID[:8])},
		shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("first write with the right part: %v", err)
	}
	close(parking.Release)

	if err := <-errc; !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
		t.Errorf("racing first write with another part: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
	}
	if n := apps.ShareCount(t, p); n != 1 {
		t.Errorf("share rows = %d, want 1", n)
	}
}

// An unmigrated project's key changes when it is migrated, with the same
// stored part. A check stored for the pre-migration key would then reject the
// project's part.
func TestLegacyKey_UnmigratedWriteStoresNoKeyCheck(t *testing.T) {
	apps := New(t)
	p, part, _ := seedLegacyProject(t, apps, false)

	err := apps.Share.RegisterShare(UserContext(p),
		&share.Share{Secret: "wallet", Entropy: share.EntropyProject}, shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("RegisterShare on an unmigrated project: %v", err)
	}

	if check := storedKeyCheck(t, apps, p); check != nil {
		t.Errorf("key check stored for an unmigrated key: %q", *check)
	}
}

// The deprecated combine puts the stored part at x = 0 and interpolates at 0,
// so an unmigrated project's key is its stored part and a read succeeds with
// any project part. A migration started from such a read re-encrypts every
// share under a key derived from a part nobody proved, and the project's own
// part stops working.
func TestLegacyKey_WrongPartReadLeavesSharesUnchanged(t *testing.T) {
	apps := New(t)
	p, part, key := seedLegacyProject(t, apps, false)
	reference := shareReference(t, apps, insertProjectShare(t, apps, p, key, "wallet"))
	before := projectShareData(t, apps, p)

	_, _ = apps.Share.GetKeychainShares(UserContext(p), &reference, shareapp.WithEncryptionPart(randomPart(t)))

	if n := migrationRows(t, apps, p); n != 0 {
		t.Errorf("migration records after a wrong-part read = %d, want 0", n)
	}
	if after := projectShareData(t, apps, p); !maps.Equal(after, before) {
		t.Error("a read with a wrong part rewrote the project's shares")
	}
	assertSharesDecrypt(t, apps, p, part, 1)
	shrs, err := apps.Share.GetKeychainShares(UserContext(p), &reference, shareapp.WithEncryptionPart(part))
	if err != nil || len(shrs) != 1 || shrs[0].Secret != "wallet" {
		t.Errorf("read with the project's part after a wrong-part read = %v, %v; want %q", shrs, err, "wallet")
	}
}

// Share reads and writes leave a project's migration status as it is: an
// unmigrated project keeps working under its pre-migration key, and a
// migrated one under its key.
func TestProjectKey_ReadsAndWritesKeepMigrationStatus(t *testing.T) {
	for _, migrated := range []bool{false, true} {
		t.Run(fmt.Sprintf("migrated=%v", migrated), func(t *testing.T) {
			apps := New(t)
			p, part, key := seedLegacyProject(t, apps, migrated)
			ctx := UserContext(p)
			withPart := shareapp.WithEncryptionPart(part)
			existing := shareReference(t, apps, insertProjectShare(t, apps, p, key, "wallet"))
			second := "second-" + p.ID[:8]
			rowsBefore := migrationRows(t, apps, p)

			if shrs, err := apps.Share.GetKeychainShares(ctx, &existing, withPart); err != nil || len(shrs) != 1 || shrs[0].Secret != "wallet" {
				t.Fatalf("read = %v, %v; want %q", shrs, err, "wallet")
			}
			err := apps.Share.RegisterShare(ctx, &share.Share{Secret: "second", Entropy: share.EntropyProject, Reference: &second}, withPart)
			if err != nil {
				t.Fatalf("RegisterShare: %v", err)
			}
			if _, err := apps.Share.UpdateShare(ctx, &share.Share{Secret: "updated"}, existing, withPart); err != nil {
				t.Fatalf("UpdateShare: %v", err)
			}

			for reference, want := range map[string]string{existing: "updated", second: "second"} {
				shrs, err := apps.Share.GetKeychainShares(ctx, &reference, withPart)
				if err != nil || len(shrs) != 1 || shrs[0].Secret != want {
					t.Errorf("read %s = %v, %v; want %q", reference, shrs, err, want)
				}
			}
			if n := migrationRows(t, apps, p); n != rowsBefore {
				t.Errorf("migration records = %d, want %d", n, rowsBefore)
			}
			if got := isMigrated(t, apps, p); got != migrated {
				t.Errorf("migrated = %v, want %v", got, migrated)
			}
			assertSharesDecrypt(t, apps, p, part, 2)
		})
	}
}

func migrationRows(t *testing.T, apps Apps, p dbtest.Project) int64 {
	t.Helper()
	var n int64
	if err := apps.DB.Raw(`SELECT count(*) FROM shld_shamir_migrations WHERE project_id = ?`, p.ID).Scan(&n).Error; err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	return n
}
