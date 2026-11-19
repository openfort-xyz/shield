package projectsvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/projectrepo"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

// A registration racing another that has inserted its part but not yet
// committed must not insert a second part: each caller would hold the
// project half of a different key, and only the first row by id is used.
func TestService_SetEncryptionKey_RacingRegistrationStoresNoSecondKey(t *testing.T) {
	client := dbtest.RequireDB(t)
	svc := New(projectrepo.New(client), time.Minute)
	p := dbtest.SeedProject(t, client, "", true)
	ctx := context.Background()

	db, err := client.DB.DB()
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
		errc <- svc.SetEncryptionKey(ctx, p.ID, &project.EncryptionKey{Part: "part-b"}, func(project.APISecretRotation) error { return nil })
	}()
	err = commitWhenBlockedOrDone(t, client, inflight.Commit, inflightPID, errc)

	if !errors.Is(err, domainErrors.ErrEncryptionPartAlreadyExists) {
		t.Errorf("racing registration: err = %v, want %v", err, domainErrors.ErrEncryptionPartAlreadyExists)
	}
	var rows int64
	client.Raw(`SELECT count(*) FROM shld_encryption_parts WHERE project_id = ?`, p.ID).Scan(&rows)
	if rows != 1 {
		t.Errorf("encryption part rows = %d, want 1", rows)
	}
}

// commitWhenBlockedOrDone commits once another backend waits on pid's locks
// or the racing call has returned, and returns the racing call's error.
func commitWhenBlockedOrDone(t *testing.T, client *sql.Client, commit func() error, pid int, errc <-chan error) error {
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
		if err := client.Raw(`SELECT count(*) FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid))`, pid).Scan(&blocked).Error; err != nil {
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
