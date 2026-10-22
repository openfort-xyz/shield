package sharerepo

import (
	"context"
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

func TestGuardedWrites(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	ctx := context.Background()

	tests := []struct {
		name       string
		storedPart project.StoredPart
		wantErr    error
	}{
		{"current part persists", "P1", nil},
		{"replaced part aborts", "P0", domainErrors.ErrEncryptionPartChanged},
	}

	for _, tt := range tests {
		t.Run("Create/"+tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", true)
			shr := &share.Share{Secret: "ciphertext", UserID: p.UserID, KeychainID: &p.KeychainID, Entropy: share.EntropyProject,
				EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart}}

			err := repo.Create(ctx, shr)

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
			p := dbtest.SeedProject(t, client, "P1", true)
			id := dbtest.InsertShare(t, client, p, "project", "old")

			err := repo.Update(ctx, &share.Share{ID: id, Secret: "new", UserID: p.UserID, Entropy: share.EntropyProject,
				EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart}})

			assertGuarded(t, client, id, err, tt.wantErr)
		})

		t.Run("BulkUpdate/"+tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", true)
			id := dbtest.InsertShare(t, client, p, "project", "old")

			err := repo.BulkUpdate(ctx, []*share.Share{{ID: id, Secret: "new", UserID: p.UserID, Entropy: share.EntropyProject,
				EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart}}})

			assertGuarded(t, client, id, err, tt.wantErr)
		})

		t.Run("UpdateProjectEncryption/"+tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", true)
			id := dbtest.InsertShare(t, client, p, "none", "old")

			err := repo.UpdateProjectEncryption(ctx, id, "new",
				&share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart})

			assertGuarded(t, client, id, err, tt.wantErr)
		})
	}

	t.Run("missing part row aborts", func(t *testing.T) {
		p := dbtest.SeedProject(t, client, "", true)
		err := repo.Create(ctx, &share.Share{Secret: "c", UserID: p.UserID, KeychainID: &p.KeychainID, Entropy: share.EntropyProject,
			EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: "P1"}})
		if !errors.Is(err, domainErrors.ErrEncryptionPartChanged) {
			t.Errorf("Create without a part row: err = %v, want %v", err, domainErrors.ErrEncryptionPartChanged)
		}
	})
}

// assertGuarded checks the write's error and that the share's data changed
// only when the write was allowed.
func assertGuarded(t *testing.T, client *sql.Client, shareID string, err, wantErr error) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	want := "new"
	if wantErr != nil {
		want = "old"
	}
	if got := dbtest.ShareData(t, client, shareID); got != want {
		t.Errorf("share data = %q, want %q", got, want)
	}
}

// Readers use the project's first part row (by id); the guard must compare
// the same row, or a project with duplicate rows can never write.
func TestGuardedWrites_DuplicatePartRows(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	p := dbtest.SeedProject(t, client, "", true)
	dbtest.Exec(t, client, `INSERT INTO shld_encryption_parts (id, project_id, part) VALUES ('00000000-0000-0000-0000-000000000001', ?, 'FIRST')`, p.ID)
	dbtest.Exec(t, client, `INSERT INTO shld_encryption_parts (id, project_id, part) VALUES ('ffffffff-ffff-ffff-ffff-ffffffffffff', ?, 'SECOND')`, p.ID)

	err := repo.Create(context.Background(), &share.Share{Secret: "c", UserID: p.UserID, KeychainID: &p.KeychainID, Entropy: share.EntropyProject,
		EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: "FIRST"}})
	if err != nil {
		t.Errorf("guarded write with the part readers use: %v", err)
	}
}
