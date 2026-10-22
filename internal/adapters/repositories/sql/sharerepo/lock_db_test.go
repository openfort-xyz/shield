package sharerepo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

const holdWindow = 500 * time.Millisecond

// TestPartRowLockSemantics pins the Postgres behaviour the share-write guard
// relies on (READ COMMITTED):
//   - a FOR SHARE read blocked by a rotation's FOR UPDATE + in-place UPDATE
//     waits for the commit and then returns the NEW part;
//   - if the rotation deletes and re-inserts the row instead, the blocked
//     FOR SHARE read returns zero rows, so zero rows must abort a write;
//   - a plain read does not wait and returns the OLD part, so a guard
//     without a lock is unsafe.
func TestPartRowLockSemantics(t *testing.T) {
	client := dbtest.RequireDB(t)
	db, err := client.DB.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	ctx := context.Background()

	cases := []struct {
		name       string
		rotate     []string
		read       string
		wantPart   string
		wantRows   int
		wantBlocks bool
	}{
		{
			name:       "in-place update: FOR SHARE waits and sees the new part",
			rotate:     []string{`UPDATE shld_encryption_parts SET part = 'P2' WHERE project_id = $1`},
			read:       `SELECT part FROM shld_encryption_parts WHERE project_id = $1 FOR SHARE`,
			wantPart:   "P2",
			wantRows:   1,
			wantBlocks: true,
		},
		{
			name: "delete and insert: FOR SHARE sees no row",
			rotate: []string{
				`DELETE FROM shld_encryption_parts WHERE project_id = $1`,
				`INSERT INTO shld_encryption_parts (id, project_id, part) VALUES (gen_random_uuid(), $1, 'P2')`,
			},
			read:       `SELECT part FROM shld_encryption_parts WHERE project_id = $1 FOR SHARE`,
			wantRows:   0,
			wantBlocks: true,
		},
		{
			name:       "plain read: does not wait and sees the old part",
			rotate:     []string{`UPDATE shld_encryption_parts SET part = 'P2' WHERE project_id = $1`},
			read:       `SELECT part FROM shld_encryption_parts WHERE project_id = $1`,
			wantPart:   "P1",
			wantRows:   1,
			wantBlocks: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := dbtest.SeedProject(t, client, "P1", true)

			rotation, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("begin rotation: %v", err)
			}
			defer func() { _ = rotation.Rollback() }()
			if _, err := rotation.ExecContext(ctx, `SELECT part FROM shld_encryption_parts WHERE project_id = $1 FOR UPDATE`, p.ID); err != nil {
				t.Fatalf("lock: %v", err)
			}
			for _, stmt := range tc.rotate {
				if _, err := rotation.ExecContext(ctx, stmt, p.ID); err != nil {
					t.Fatalf("rotate: %v", err)
				}
			}

			type result struct {
				parts   []string
				elapsed time.Duration
				err     error
			}
			done := make(chan result, 1)
			go func() {
				start := time.Now()
				parts, err := readParts(ctx, db, tc.read, p.ID)
				done <- result{parts, time.Since(start), err}
			}()

			time.Sleep(holdWindow)
			if err := rotation.Commit(); err != nil {
				t.Fatalf("commit rotation: %v", err)
			}
			got := <-done
			if got.err != nil {
				t.Fatalf("read: %v", got.err)
			}

			if blocked := got.elapsed >= holdWindow*8/10; blocked != tc.wantBlocks {
				t.Errorf("read blocked = %v (took %v), want %v", blocked, got.elapsed, tc.wantBlocks)
			}
			if len(got.parts) != tc.wantRows {
				t.Fatalf("read returned %d rows %v, want %d", len(got.parts), got.parts, tc.wantRows)
			}
			if tc.wantRows == 1 && got.parts[0] != tc.wantPart {
				t.Errorf("read part = %q, want %q", got.parts[0], tc.wantPart)
			}
		})
	}
}

func readParts(ctx context.Context, db *sql.DB, query, projectID string) ([]string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var parts []string
	for rows.Next() {
		var part string
		if err := rows.Scan(&part); err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, rows.Err()
}
