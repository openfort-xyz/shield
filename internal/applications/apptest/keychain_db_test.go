package apptest

import (
	"context"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// A user's first requests create their keychain. A request racing another
// that has inserted the user's keychain but not yet committed must use that
// keychain, not create a second one: shares split across two keychains drop
// out of keychain reads, updates and deletes.
func TestKeychain_RacingFirstRequestsShareOneKeychain(t *testing.T) {
	tests := []struct {
		name        string
		legacyShare bool
		request     func(apps Apps, p dbtest.Project) error
	}{
		{"RegisterShare", true, func(apps Apps, p dbtest.Project) error {
			return apps.Share.RegisterShare(UserContext(p), &share.Share{Secret: "plain", Entropy: share.EntropyNone, Reference: ptr("new-" + p.ID[:8])})
		}},
		{"GetShareEncryption", false, func(apps Apps, p dbtest.Project) error {
			_, _, err := apps.Share.GetShareEncryption(UserContext(p))
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps := New(t)
			p := dbtest.SeedProject(t, apps.DB, "", true)
			if tt.legacyShare {
				id := dbtest.InsertShare(t, apps.DB, p, "none", "pre-keychain")
				dbtest.Exec(t, apps.DB, `UPDATE shld_shares SET keychain_id = NULL, reference = NULL WHERE id = ?`, id)
			}
			dbtest.Exec(t, apps.DB, `DELETE FROM shld_keychains WHERE user_id = ?`, p.UserID)
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
			if _, err := inflight.ExecContext(ctx, `INSERT INTO shld_keychains (id, user_id) VALUES ($1, $2)`, p.KeychainID, p.UserID); err != nil {
				t.Fatalf("insert in-flight keychain: %v", err)
			}
			var inflightPID int
			if err := inflight.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&inflightPID); err != nil {
				t.Fatalf("backend pid: %v", err)
			}

			errc := make(chan error, 1)
			go func() { errc <- tt.request(apps, p) }()
			_ = commitWhenBlockedOrDone(t, apps, inflight.Commit, inflightPID, errc)

			var keychains []string
			apps.DB.Raw(`SELECT id FROM shld_keychains WHERE user_id = ?`, p.UserID).Scan(&keychains)
			if len(keychains) != 1 {
				t.Fatalf("user keychains = %v, want exactly one", keychains)
			}
			var stray int64
			apps.DB.Raw(`SELECT count(*) FROM shld_shares WHERE user_id = ? AND keychain_id IS DISTINCT FROM ?`, p.UserID, keychains[0]).Scan(&stray)
			if stray != 0 {
				t.Errorf("%d of the user's shares are outside their keychain", stray)
			}
		})
	}
}

// commitWhenBlockedOrDone commits once another backend waits on pid's locks
// or the racing call has returned, and returns the racing call's error.
func commitWhenBlockedOrDone(t *testing.T, apps Apps, commit func() error, pid int, errc <-chan error) error {
	t.Helper()
	for {
		select {
		case err := <-errc:
			if cerr := commit(); cerr != nil {
				t.Fatalf("commit: %v", cerr)
			}
			return err
		default:
		}
		var blocked int64
		if err := apps.DB.Raw(`SELECT count(*) FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid))`, pid).Scan(&blocked).Error; err != nil {
			t.Fatalf("pg_stat_activity: %v", err)
		}
		if blocked > 0 {
			if err := commit(); err != nil {
				t.Fatalf("commit: %v", err)
			}
			return <-errc
		}
		time.Sleep(10 * time.Millisecond)
	}
}
