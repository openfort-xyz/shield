package apptest

import (
	"strings"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// A registered key is split with the current reconstruction, so the project
// must read as migrated from the moment its part is stored. Storing the part
// alone leaves a project whose part every request rebuilds wrongly, whose
// half was never returned, and which cannot register again.
func TestRegisterEncryptionKey_FailedStatusWriteStoresNoKey(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", false)
	restore := failSuccessfulMigrations(t, apps, p)

	if _, err := apps.Project.RegisterEncryptionKey(ProjectContext(p)); err == nil {
		t.Fatal("registration succeeded although recording the migration failed")
	}
	var parts int64
	if err := apps.DB.Raw(`SELECT count(*) FROM shld_encryption_parts WHERE project_id = ?`, p.ID).Scan(&parts).Error; err != nil {
		t.Fatalf("count parts: %v", err)
	}
	if parts != 0 {
		t.Errorf("part rows after a failed registration = %d, want 0", parts)
	}

	restore()
	part, err := apps.Project.RegisterEncryptionKey(ProjectContext(p))
	if err != nil {
		t.Fatalf("registration after the failure: %v", err)
	}
	reference := "wallet-" + p.ID[:8]
	err = apps.Share.RegisterShare(UserContext(p),
		&share.Share{Secret: "wallet", Entropy: share.EntropyProject, Reference: &reference}, shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("RegisterShare with the registered part: %v", err)
	}
	shrs, err := apps.Share.GetKeychainShares(UserContext(p), &reference, shareapp.WithEncryptionPart(part))
	if err != nil || len(shrs) != 1 || shrs[0].Secret != "wallet" {
		t.Errorf("read with the registered part = %v, %v; want %q", shrs, err, "wallet")
	}
}

// failSuccessfulMigrations makes recording a successful migration of the
// project fail, as a crash at that point would, until restore is called.
func failSuccessfulMigrations(t *testing.T, apps Apps, p dbtest.Project) (restore func()) {
	t.Helper()
	name := "shld_test_fail_migration_" + strings.ReplaceAll(p.ID, "-", "")[:12]
	dbtest.Exec(t, apps.DB, `CREATE FUNCTION `+name+`() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.success AND NEW.project_id = TG_ARGV[0] THEN
				RAISE EXCEPTION 'injected failure recording a migration';
			END IF;
			RETURN NEW;
		END $$`)
	dbtest.Exec(t, apps.DB, `CREATE TRIGGER `+name+` BEFORE INSERT ON shld_shamir_migrations
		FOR EACH ROW EXECUTE FUNCTION `+name+`('`+p.ID+`')`)

	dropped := false
	restore = func() {
		if dropped {
			return
		}
		dropped = true
		dbtest.Exec(t, apps.DB, `DROP TRIGGER `+name+` ON shld_shamir_migrations`)
		dbtest.Exec(t, apps.DB, `DROP FUNCTION `+name+`()`)
	}
	t.Cleanup(restore)
	return restore
}
