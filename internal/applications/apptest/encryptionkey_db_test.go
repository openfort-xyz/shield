package apptest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
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

// A registration racing another that has inserted its part but not yet
// committed stores nothing; its caller must learn the project already has a
// key, not get an internal error.
func TestRegisterEncryptionKey_RacingRegistrationLoserGetsAlreadyExists(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", false)
	ctx := context.Background()

	db, err := apps.DB.DB.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	inflight, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = inflight.Rollback() }()
	if _, err := inflight.ExecContext(ctx, `INSERT INTO shld_encryption_parts (id, project_id, part) VALUES (gen_random_uuid(), $1, 'part-a')`, p.ID); err != nil {
		t.Fatalf("insert in-flight part: %v", err)
	}
	var inflightPID int
	if err := inflight.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&inflightPID); err != nil {
		t.Fatalf("backend pid: %v", err)
	}

	errc := make(chan error, 1)
	go func() {
		_, err := apps.Project.RegisterEncryptionKey(ProjectContext(p))
		errc <- err
	}()
	err = commitWhenBlockedOrDone(t, apps, inflight.Commit, inflightPID, errc)

	if !errors.Is(err, projectapp.ErrEncryptionPartAlreadyExists) {
		t.Errorf("racing registration: err = %v, want %v", err, projectapp.ErrEncryptionPartAlreadyExists)
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
