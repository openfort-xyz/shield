package apptest

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	depsssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/deprecated_sss_reconstruction_strategy"
	sssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/logger/logtest"
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

			if !isUnproven(err) {
				t.Errorf("write with a wrong part: err = %v, want encryption part unproven", err)
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

// A refused write can be a wrong part or a project whose oldest shares were
// all written with one; the operator needs to see which project.
func TestLegacyKey_UnprovenWriteLogsProjectAtErrorLevel(t *testing.T) {
	rec := logtest.Start(slog.LevelInfo)
	defer rec.Stop()
	apps := New(t)
	_, otherPart := apps.SeedEncryptedProject(t)
	p, _, key := seedLegacyProject(t, apps, true)
	insertProjectShare(t, apps, p, key, "wallet")
	rec.Reset()

	err := apps.Share.RegisterShare(UserContext(p),
		&share.Share{Secret: "new", Entropy: share.EntropyProject, Reference: ptr("new-" + p.ID[:8])},
		shareapp.WithEncryptionPart(otherPart))
	if !isUnproven(err) {
		t.Fatalf("write with a wrong part: err = %v, want encryption part unproven", err)
	}

	for _, r := range rec.Records() {
		if r.Logger == "share_repository" && r.Severity == "ERROR" && strings.Contains(r.Attr("error"), p.ID) {
			return
		}
	}
	t.Errorf("no share repository error record names project %s: %v", p.ID, rec.Records())
}

func isUnproven(err error) bool {
	return errors.Is(err, shareapp.ErrEncryptionPartUnproven) || errors.Is(err, projectapp.ErrEncryptionPartUnproven)
}

// keyWrites are the writes that encrypt a share under the project key. Each
// gets the reference of a plain share the test seeded: UpdateShare moves it to
// project entropy and EncryptProjectShares encrypts it.
var keyWrites = []struct {
	name  string
	write func(apps Apps, p dbtest.Project, part, plainRef string) error
}{
	{"RegisterShare", func(apps Apps, p dbtest.Project, part, _ string) error {
		return apps.Share.RegisterShare(UserContext(p),
			&share.Share{Secret: "new", Entropy: share.EntropyProject, Reference: ptr("new-" + p.ID[:8])},
			shareapp.WithEncryptionPart(part))
	}},
	{"UpdateShare", func(apps Apps, p dbtest.Project, part, plainRef string) error {
		_, err := apps.Share.UpdateShare(UserContext(p), &share.Share{Secret: "updated", Entropy: share.EntropyProject},
			plainRef, shareapp.WithEncryptionPart(part))
		return err
	}},
	{"EncryptProjectShares", func(apps Apps, p dbtest.Project, part, _ string) error {
		return apps.Project.EncryptProjectShares(ProjectContext(p), part)
	}},
}

func plainShareRef(t *testing.T, apps Apps, p dbtest.Project) string {
	t.Helper()
	return shareReference(t, apps, dbtest.InsertShare(t, apps.DB, p, "none", "plain-share"))
}

func sharesUnderKey(t *testing.T, apps Apps, p dbtest.Project, key string) int {
	t.Helper()
	var rows []string
	if err := apps.DB.Raw(`SELECT data FROM shld_shares WHERE user_id = ? AND entropy = 'project'`, p.UserID).Scan(&rows).Error; err != nil {
		t.Fatalf("read shares: %v", err)
	}
	n := 0
	for _, data := range rows {
		if _, err := cypher.Decrypt(data, key); err == nil {
			n++
		}
	}
	return n
}

// backdate sets when a share was created: the proof tries the oldest first.
func backdate(t *testing.T, apps Apps, shareID string, age time.Duration) {
	t.Helper()
	dbtest.Exec(t, apps.DB, `UPDATE shld_shares SET created_at = ? WHERE id = ?`, time.Now().Add(-age), shareID)
}

// strayCiphertext encrypts secret under a key the project's part does not
// rebuild, as a write with a wrong part before key checks existed left it.
func strayCiphertext(t *testing.T, secret string) string {
	t.Helper()
	ciphertext, err := cypher.Encrypt(secret, randomPart(t))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return ciphertext
}

// Only rotation, which proves a key against every project share, stores a
// check for an existing key. A share write that stored one could anchor a
// wrong part on a project with nothing to prove it against.
func TestLegacyKey_ShareWritesStoreNoKeyCheck(t *testing.T) {
	apps := New(t)
	_, otherPart := apps.SeedEncryptedProject(t)

	cases := []struct {
		name      string
		withShare bool
		rightPart bool
	}{
		{"right part, project with a share", true, true},
		{"right part, project without shares", false, true},
		{"wrong part, project without shares", false, false},
	}
	for _, w := range keyWrites {
		for _, c := range cases {
			t.Run(w.name+"/"+c.name, func(t *testing.T) {
				p, part, key := seedLegacyProject(t, apps, true)
				if c.withShare {
					insertProjectShare(t, apps, p, key, "wallet")
				}
				if !c.rightPart {
					part = otherPart
				}

				if err := w.write(apps, p, part, plainShareRef(t, apps, p)); err != nil {
					t.Fatalf("write: %v", err)
				}
				if check := storedKeyCheck(t, apps, p); check != nil {
					t.Errorf("key check stored by a share write: %q", *check)
				}
			})
		}
	}
}

// With no project share there is nothing to prove a part against, so a wrong
// part's write is stored as before key checks existed, and nothing else: the
// project's part then fails only the writes this stray share makes unproven,
// until it is deleted, and is never locked out by a stored check.
func TestLegacyKey_WrongPartFirstOnProjectWithoutShares(t *testing.T) {
	apps := New(t)
	_, otherPart := apps.SeedEncryptedProject(t)
	p, part, _ := seedLegacyProject(t, apps, true)
	ctx := UserContext(p)
	stray := "stray-" + p.ID[:8]
	wallet := "wallet-" + p.ID[:8]

	err := apps.Share.RegisterShare(ctx, &share.Share{Secret: "stray", Entropy: share.EntropyProject, Reference: &stray},
		shareapp.WithEncryptionPart(otherPart))
	if err != nil {
		t.Fatalf("first write with a wrong part on a project without shares: %v", err)
	}
	if check := storedKeyCheck(t, apps, p); check != nil {
		t.Fatalf("key check stored from a wrong part: %q", *check)
	}

	err = apps.Share.RegisterShare(ctx, &share.Share{Secret: "wallet", Entropy: share.EntropyProject, Reference: &wallet},
		shareapp.WithEncryptionPart(part))
	if !isUnproven(err) {
		t.Errorf("right part while only a stray share exists: err = %v, want encryption part unproven", err)
	}

	dbtest.Exec(t, apps.DB, `DELETE FROM shld_shares WHERE user_id = ? AND reference = ?`, p.UserID, stray)

	err = apps.Share.RegisterShare(ctx, &share.Share{Secret: "wallet", Entropy: share.EntropyProject, Reference: &wallet},
		shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("write with the right part after the stray share is deleted: %v", err)
	}
	shrs, err := apps.Share.GetKeychainShares(ctx, &wallet, shareapp.WithEncryptionPart(part))
	if err != nil || len(shrs) != 1 || shrs[0].Secret != "wallet" {
		t.Errorf("read with the right part = %v, %v; want %q", shrs, err, "wallet")
	}
	if _, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), part); err != nil {
		t.Errorf("rotation prepare with the right part: %v", err)
	}
}

// A share under another key, as a write with a wrong part before key checks
// existed could leave, or a share whose data is not ciphertext, must not lock
// the project's part out of writes while another share proves it.
func TestLegacyKey_UndecryptableShareDoesNotBlockRightPart(t *testing.T) {
	apps := New(t)
	undecryptable := map[string]string{
		"stray":     strayCiphertext(t, "stray"),
		"malformed": "%not-ciphertext%",
	}

	for _, w := range keyWrites {
		for name, data := range undecryptable {
			t.Run(w.name+"/"+name, func(t *testing.T) {
				p, part, key := seedLegacyProject(t, apps, true)
				backdate(t, apps, dbtest.InsertShare(t, apps.DB, p, "project", data), time.Hour)
				insertProjectShare(t, apps, p, key, "wallet")

				if err := w.write(apps, p, part, plainShareRef(t, apps, p)); err != nil {
					t.Errorf("write with the right part: %v", err)
				}
				if n := sharesUnderKey(t, apps, p, key); n != 2 {
					t.Errorf("project shares under the project's key = %d, want 2", n)
				}
			})
		}
	}
}

// With only shares the part does not decrypt, nothing proves it: the write is
// refused with its own error, never an internal one, and stores nothing.
func TestLegacyKey_OnlyUndecryptableSharesRefuseWrite(t *testing.T) {
	apps := New(t)
	undecryptable := map[string]string{
		"stray":     strayCiphertext(t, "stray"),
		"malformed": "%not-ciphertext%",
	}

	for _, w := range keyWrites {
		for name, data := range undecryptable {
			t.Run(w.name+"/"+name, func(t *testing.T) {
				p, part, _ := seedLegacyProject(t, apps, true)
				dbtest.InsertShare(t, apps.DB, p, "project", data)
				plainRef := plainShareRef(t, apps, p)
				before := projectShareData(t, apps, p)

				if err := w.write(apps, p, part, plainRef); !isUnproven(err) {
					t.Errorf("write: err = %v, want encryption part unproven", err)
				}
				if after := projectShareData(t, apps, p); !maps.Equal(after, before) {
					t.Errorf("shares changed by a refused write: %v, want %v", after, before)
				}
			})
		}
	}
}

// The proof tries at most the project's 1000 oldest project shares, so a
// write costs at most that many decryptions however many shares there are.
func TestLegacyKey_ProofTriesOldestShares(t *testing.T) {
	apps := New(t)
	tests := []struct {
		name     string
		position int
		want     error
	}{
		{"decrypting share is the 1000th oldest", 1000, nil},
		{"decrypting share is the 1001st oldest", 1001, shareapp.ErrEncryptionPartUnproven},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, part, key := seedLegacyProject(t, apps, true)
			dbtest.Exec(t, apps.DB, `INSERT INTO shld_shares (id, data, user_id, keychain_id, reference, entropy, storage_method_id, created_at)
				SELECT gen_random_uuid(), ?, ?, ?, 'bulk-' || g, 'project', 0, timestamp '2020-01-01' + g * interval '1 second'
				FROM generate_series(1, 1001) g`, strayCiphertext(t, "stray"), p.UserID, p.KeychainID)
			ciphertext, err := cypher.Encrypt("wallet", key)
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}
			dbtest.Exec(t, apps.DB, `UPDATE shld_shares SET data = ? WHERE user_id = ? AND reference = ?`,
				ciphertext, p.UserID, fmt.Sprintf("bulk-%d", tt.position))

			err = apps.Share.RegisterShare(UserContext(p),
				&share.Share{Secret: "new", Entropy: share.EntropyProject, Reference: ptr("new-" + p.ID[:8])},
				shareapp.WithEncryptionPart(part))

			if !errors.Is(err, tt.want) {
				t.Errorf("RegisterShare: err = %v, want %v", err, tt.want)
			}
		})
	}
}

// Soft-deleted shares stay restorable and rotation re-encrypts them, so they
// prove a part as live ones do: deleting a project's only share does not
// make it a project without shares, where any part is accepted.
func TestLegacyKey_SoftDeletedShareProvesPart(t *testing.T) {
	apps := New(t)
	_, otherPart := apps.SeedEncryptedProject(t)
	p, part, key := seedLegacyProject(t, apps, true)
	id := insertProjectShare(t, apps, p, key, "wallet")
	dbtest.Exec(t, apps.DB, `UPDATE shld_shares SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	ctx := UserContext(p)

	err := apps.Share.RegisterShare(ctx, &share.Share{Secret: "new", Entropy: share.EntropyProject, Reference: ptr("wrong-" + p.ID[:8])},
		shareapp.WithEncryptionPart(otherPart))
	if !isUnproven(err) {
		t.Errorf("wrong part with only a soft-deleted share: err = %v, want encryption part unproven", err)
	}
	err = apps.Share.RegisterShare(ctx, &share.Share{Secret: "new", Entropy: share.EntropyProject, Reference: ptr("right-" + p.ID[:8])},
		shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Errorf("right part with only a soft-deleted share: %v", err)
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

// A write that reconstructed an unmigrated project's key and persists after a
// migration (such as one a replica of an earlier release starts) would leave
// its share under a key no request rebuilds any more.
func TestLegacyKey_WriteRacingMigrationIsNotPersisted(t *testing.T) {
	tests := []struct {
		name  string
		write func(apps Apps, p dbtest.Project, part, existing string) error
	}{
		{"RegisterShare", func(apps Apps, p dbtest.Project, part, _ string) error {
			return apps.Share.RegisterShare(UserContext(p),
				&share.Share{Secret: "new", Entropy: share.EntropyProject, Reference: ptr("new-" + p.ID[:8])},
				shareapp.WithEncryptionPart(part))
		}},
		{"UpdateShare", func(apps Apps, p dbtest.Project, part, existing string) error {
			_, err := apps.Share.UpdateShare(UserContext(p), &share.Share{Secret: "new"}, existing, shareapp.WithEncryptionPart(part))
			return err
		}},
		{"EncryptProjectShares", func(apps Apps, p dbtest.Project, part, _ string) error {
			return apps.Project.EncryptProjectShares(ProjectContext(p), part)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parking := NewParkingShareRepo()
			apps := NewWithShareRepo(t, parking.Wrap)
			p, part, key := seedLegacyProject(t, apps, false)
			existing := shareReference(t, apps, insertProjectShare(t, apps, p, key, "existing"))
			plainID := dbtest.InsertShare(t, apps.DB, p, "none", "plain-share")

			parking.Armed.Store(true)
			errc := make(chan error, 1)
			go func() { errc <- tt.write(apps, p, part, existing) }()
			<-parking.Parked
			parking.Armed.Store(false)
			migrateOutOfBand(t, apps, p, part)
			close(parking.Release)

			if err := <-errc; !errors.Is(err, shareapp.ErrEncryptionPartChanged) && !errors.Is(err, projectapp.ErrEncryptionPartChanged) {
				t.Errorf("write racing a migration: err = %v, want encryption part changed", err)
			}
			assertSharesDecrypt(t, apps, p, part, 1)
			if got := dbtest.ShareData(t, apps.DB, plainID); got != "plain-share" {
				t.Errorf("plain share = %q, want it untouched", got)
			}
		})
	}
}

// migrateOutOfBand migrates the project in one transaction holding FOR UPDATE
// on its part row: every project share, soft-deleted ones included, moves
// from the pre-migration key to the current one, and the migration is
// recorded.
func migrateOutOfBand(t *testing.T, apps Apps, p dbtest.Project, projectPart string) {
	t.Helper()
	err := apps.DB.Transaction(func(tx *gorm.DB) error {
		var stored string
		if err := tx.Raw(`SELECT part FROM shld_encryption_parts WHERE project_id = ? ORDER BY id LIMIT 1 FOR UPDATE`, p.ID).Scan(&stored).Error; err != nil {
			return err
		}
		oldKey, err := depsssrec.NewSSSReconstructionStrategy().Reconstruct(project.StoredPart(stored), project.ProjectPart(projectPart))
		if err != nil {
			return err
		}
		newKey, err := sssrec.NewSSSReconstructionStrategy().Reconstruct(project.StoredPart(stored), project.ProjectPart(projectPart))
		if err != nil {
			return err
		}
		var rows []struct{ ID, Data string }
		if err := tx.Raw(`SELECT s.id, s.data FROM shld_shares s JOIN shld_users u ON s.user_id = u.id
			WHERE u.project_id = ? AND s.entropy = 'project' FOR UPDATE OF s`, p.ID).Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			plain, err := cypher.Decrypt(row.Data, oldKey)
			if err != nil {
				return err
			}
			data, err := cypher.Encrypt(plain, newKey)
			if err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE shld_shares SET data = ? WHERE id = ?`, data, row.ID).Error; err != nil {
				return err
			}
		}
		return tx.Exec(`INSERT INTO shld_shamir_migrations (id, project_id, success) VALUES (?, ?, true)`, uuid.NewString(), p.ID).Error
	})
	if err != nil {
		t.Fatalf("out-of-band migration: %v", err)
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
