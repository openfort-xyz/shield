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
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/random"
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
				EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart, Migrated: true}}

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
				EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart, Migrated: true}})

			assertGuarded(t, client, id, err, tt.wantErr)
		})

		t.Run("UpdateProjectEncryption/"+tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", true)
			id := dbtest.InsertShare(t, client, p, "none", "old")

			err := repo.UpdateProjectEncryption(ctx, id, "old", "new",
				&share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart, Migrated: true})

			assertGuarded(t, client, id, err, tt.wantErr)
		})
	}

	t.Run("missing part row aborts", func(t *testing.T) {
		p := dbtest.SeedProject(t, client, "", true)
		err := repo.Create(ctx, &share.Share{Secret: "c", UserID: p.UserID, KeychainID: &p.KeychainID, Entropy: share.EntropyProject,
			EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: "P1", Migrated: true}})
		if !errors.Is(err, domainErrors.ErrEncryptionPartChanged) {
			t.Errorf("Create without a part row: err = %v, want %v", err, domainErrors.ErrEncryptionPartChanged)
		}
	})
}

// The sample proof of a key without a key check runs before the write's
// transaction, so the transaction decides with the part row locked: a part
// replaced since reports the rotation, and a key check stored since decides
// over the sample.
func TestGuardedWrites_UnverifiedKey(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	otherKey, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	stray, err := cypher.Encrypt("stray", otherKey)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	tests := []struct {
		name       string
		storedPart project.StoredPart
		checkOf    string
		strayShare bool
		wantErr    error
	}{
		{"no share and no check persists", "P1", "", false, nil},
		{"no decrypting share is unproven", "P1", "", true, domainErrors.ErrEncryptionPartUnproven},
		{"replaced part reports the rotation first", "P0", "", true, domainErrors.ErrEncryptionPartChanged},
		{"check stored since rejects another key", "P1", otherKey, false, domainErrors.ErrInvalidEncryptionPart},
		{"check stored since accepts the key over the sample", "P1", key, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", true)
			if tt.checkOf != "" {
				check, err := cypher.KeyCheck(tt.checkOf)
				if err != nil {
					t.Fatalf("key check: %v", err)
				}
				dbtest.Exec(t, client, `UPDATE shld_encryption_parts SET key_check = ? WHERE project_id = ?`, check, p.ID)
			}
			if tt.strayShare {
				dbtest.InsertShare(t, client, p, "project", stray)
			}

			err := repo.Create(context.Background(), &share.Share{Secret: "c", UserID: p.UserID, KeychainID: &p.KeychainID,
				Entropy: share.EntropyProject, Reference: ptr("new"),
				EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: tt.storedPart, Migrated: true, UnverifiedKey: key}})

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create: err = %v, want %v", err, tt.wantErr)
			}
			var n int64
			client.Raw(`SELECT count(*) FROM shld_shares WHERE user_id = ? AND reference = 'new'`, p.UserID).Scan(&n)
			if want := map[bool]int64{true: 0, false: 1}[tt.wantErr != nil]; n != want {
				t.Errorf("written share rows = %d, want %d", n, want)
			}
		})
	}
}

// The proof's database work follows its sample, not the project: on a project
// with five times the sample, a key that decrypts the first shares read stops
// after a few rows, and one that decrypts none reads about the sample.
func TestGuardedWrites_UnverifiedKeyProofReadsBounded(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	otherKey, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	ciphertext, err := cypher.Encrypt("wallet", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	p := dbtest.SeedProject(t, client, "P1", true)
	projectShares := 5 * proofSampleSize
	dbtest.Exec(t, client, `INSERT INTO shld_shares (id, data, user_id, keychain_id, reference, entropy, storage_method_id)
		SELECT gen_random_uuid(), ?, ?, ?, 'bulk-' || gen_random_uuid(), 'project', 0 FROM generate_series(1, ?)`,
		ciphertext, p.UserID, p.KeychainID, projectShares)
	dbtest.Exec(t, client, `ANALYZE shld_users`)
	dbtest.Exec(t, client, `ANALYZE shld_shares`)

	tests := []struct {
		name             string
		key              string
		minRows, maxRows int
		wantErr          error
	}{
		{"key decrypts the first shares read", key, 1, 100, nil},
		{"key decrypts none", otherKey, proofSampleSize, 2 * proofSampleSize, domainErrors.ErrEncryptionPartUnproven},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			reads := dbtest.ShareReads(t, client, func() {
				err = repo.Create(context.Background(), &share.Share{Secret: "c", UserID: p.UserID, KeychainID: &p.KeychainID,
					Entropy: share.EntropyProject, Reference: ptr("new-" + tt.key[:8]),
					EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: "P1", Migrated: true, UnverifiedKey: tt.key}})
			})

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create: err = %v, want %v", err, tt.wantErr)
			}
			rows := 0
			for _, n := range reads {
				rows += n
			}
			if rows < tt.minRows || rows > tt.maxRows {
				t.Errorf("proof read %d share rows (%v) of a project with %d, want %d to %d",
					rows, reads, projectShares, tt.minRows, tt.maxRows)
			}
		})
	}
}

func ptr(s string) *string { return &s }

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
		EncryptionGuard: &share.EncryptionGuard{ProjectID: p.ID, StoredPart: "FIRST", Migrated: true}})
	if err != nil {
		t.Errorf("guarded write with the part readers use: %v", err)
	}
}
