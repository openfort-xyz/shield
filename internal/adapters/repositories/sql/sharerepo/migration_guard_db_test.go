package sharerepo

import (
	"context"
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// A migration changes an unmigrated project's key without changing its stored
// part, so the part comparison alone lets through a write encrypted under the
// pre-migration key after the project migrated. No request rebuilds that key
// any more.
func TestGuardedWrites_PreMigrationKey(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	ctx := context.Background()
	preMigration := func(p dbtest.Project) *share.EncryptionGuard {
		return &share.EncryptionGuard{ProjectID: p.ID, StoredPart: "P1"}
	}

	tests := []struct {
		name     string
		migrated bool
		wantErr  error
	}{
		{"unmigrated project persists", false, nil},
		{"project migrated since aborts", true, domainErrors.ErrEncryptionPartChanged},
	}

	for _, tt := range tests {
		t.Run("Create/"+tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", tt.migrated)
			err := repo.Create(ctx, &share.Share{Secret: "new", UserID: p.UserID, KeychainID: &p.KeychainID,
				Entropy: share.EntropyProject, EncryptionGuard: preMigration(p)})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create: err = %v, want %v", err, tt.wantErr)
			}
			var n int64
			client.Raw(`SELECT count(*) FROM shld_shares WHERE user_id = ?`, p.UserID).Scan(&n)
			if want := map[bool]int64{true: 0, false: 1}[tt.wantErr != nil]; n != want {
				t.Errorf("share rows = %d, want %d", n, want)
			}
		})

		t.Run("Update/"+tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", tt.migrated)
			id := dbtest.InsertShare(t, client, p, "project", "old")

			err := repo.Update(ctx, &share.Share{ID: id, Secret: "new", UserID: p.UserID, Entropy: share.EntropyProject,
				EncryptionGuard: preMigration(p)})

			assertGuarded(t, client, id, err, tt.wantErr)
		})

		t.Run("UpdateProjectEncryption/"+tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", tt.migrated)
			id := dbtest.InsertShare(t, client, p, "none", "old")

			err := repo.UpdateProjectEncryption(ctx, id, "old", "new", preMigration(p))

			assertGuarded(t, client, id, err, tt.wantErr)
		})
	}
}

// A migration that records itself while holding FOR UPDATE on the part row
// cannot interleave with a pre-migration write: a write that reaches its
// guard lock meanwhile waits, then sees the record and aborts.
func TestGuardedWrites_PreMigrationKeyWaitsForMigration(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	db, err := client.DB.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	ctx := context.Background()
	p := dbtest.SeedProject(t, client, "P1", false)

	migration, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin migration: %v", err)
	}
	defer func() { _ = migration.Rollback() }()
	var pid int
	if err := migration.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatalf("backend pid: %v", err)
	}
	if _, err := migration.ExecContext(ctx, `SELECT id FROM shld_encryption_parts WHERE project_id = $1 FOR UPDATE`, p.ID); err != nil {
		t.Fatalf("lock part: %v", err)
	}
	if _, err := migration.ExecContext(ctx, `INSERT INTO shld_shamir_migrations (id, project_id, success) VALUES (gen_random_uuid(), $1, true)`, p.ID); err != nil {
		t.Fatalf("record migration: %v", err)
	}

	finished := make(chan struct{})
	var writeErr error
	go func() {
		defer close(finished)
		writeErr = repo.Create(ctx, &share.Share{Secret: "new", UserID: p.UserID, KeychainID: &p.KeychainID,
			Entropy: share.EntropyProject, EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: "P1"}})
	}()
	if !dbtest.WaitForLockWait(t, client, pid, finished) {
		t.Fatal("the write did not wait for the migration's part lock")
	}
	if err := migration.Commit(); err != nil {
		t.Fatalf("commit migration: %v", err)
	}
	<-finished

	if !errors.Is(writeErr, domainErrors.ErrEncryptionPartChanged) {
		t.Errorf("write waiting on a migration: err = %v, want %v", writeErr, domainErrors.ErrEncryptionPartChanged)
	}
	var n int64
	client.Raw(`SELECT count(*) FROM shld_shares WHERE user_id = ?`, p.UserID).Scan(&n)
	if n != 0 {
		t.Errorf("share rows = %d, want 0", n)
	}
}
