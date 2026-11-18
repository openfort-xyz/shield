// Package dbtest seeds and inspects a real database for repository and
// application tests. Tests using it skip unless DB_NAME is set.
package dbtest

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"gorm.io/gorm"
)

// RequireDB returns a SQL client or skips the test when no database is
// explicitly configured (CI currently runs without one; locally set DB_NAME).
func RequireDB(t *testing.T) *sql.Client {
	t.Helper()
	if os.Getenv("DB_NAME") == "" {
		t.Skip("DB_NAME not set; skipping database-backed test")
	}
	cfg, err := sql.GetConfigFromEnv()
	if err != nil {
		t.Fatalf("failed to read DB config: %v", err)
	}
	client, err := sql.New(cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() {
		db, err := client.DB.DB()
		if err != nil {
			t.Errorf("close database: %v", err)
			return
		}
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return client
}

// Exec runs a seed statement and fails the test on error.
func Exec(t *testing.T, client *sql.Client, query string, args ...any) {
	t.Helper()
	if err := client.Exec(query, args...).Error; err != nil {
		t.Fatalf("seed failed (%s): %v", query, err)
	}
}

// Project is a seeded project with one user and that user's keychain.
type Project struct {
	ID         string
	APIKey     string
	UserID     string
	KeychainID string
}

// SeedProject inserts a project, a user and a keychain. When storedPart is
// non-empty it also inserts the project's stored encryption part, and when
// migrated is true a successful shamir migration record. The project and
// everything under it are removed when the test ends.
func SeedProject(t *testing.T, client *sql.Client, storedPart string, migrated bool) Project {
	t.Helper()

	p := Project{ID: uuid.NewString(), APIKey: uuid.NewString(), UserID: uuid.NewString(), KeychainID: uuid.NewString()}
	Exec(t, client, `INSERT INTO shld_projects (id, name, api_key, api_secret, enable_2fa) VALUES (?, ?, ?, ?, false)`,
		p.ID, "proj-"+p.ID[:8], p.APIKey, uuid.NewString())
	Exec(t, client, `INSERT INTO shld_users (id, project_id) VALUES (?, ?)`, p.UserID, p.ID)
	Exec(t, client, `INSERT INTO shld_keychains (id, user_id) VALUES (?, ?)`, p.KeychainID, p.UserID)
	if storedPart != "" {
		Exec(t, client, `INSERT INTO shld_encryption_parts (id, project_id, part) VALUES (?, ?, ?)`, uuid.NewString(), p.ID, storedPart)
	}
	if migrated {
		Exec(t, client, `INSERT INTO shld_shamir_migrations (id, project_id, success) VALUES (?, ?, true)`, uuid.NewString(), p.ID)
	}

	t.Cleanup(func() { cleanup(t, client, p.ID) })
	return p
}

// InsertShare inserts a share for the project's user and returns its ID.
func InsertShare(t *testing.T, client *sql.Client, p Project, entropy, data string) string {
	t.Helper()
	id := uuid.NewString()
	Exec(t, client, `INSERT INTO shld_shares (id, data, user_id, keychain_id, reference, entropy, storage_method_id) VALUES (?, ?, ?, ?, ?, ?, 0)`,
		id, data, p.UserID, p.KeychainID, "ref-"+id[:8], entropy)
	return id
}

// ShareData reads a share's stored data column, including soft-deleted rows.
func ShareData(t *testing.T, client *sql.Client, shareID string) string {
	t.Helper()
	var data string
	if err := client.Raw(`SELECT data FROM shld_shares WHERE id = ?`, shareID).Scan(&data).Error; err != nil {
		t.Fatalf("read share %s: %v", shareID, err)
	}
	return data
}

// ShareReads runs fn and returns, for each raw query it ran on client that
// scans shld_shares, how many of its rows the query read, counted by rerunning
// the query under EXPLAIN ANALYZE.
func ShareReads(t *testing.T, client *sql.Client, fn func()) []int {
	t.Helper()
	var queries []string
	recording := true
	err := client.Callback().Row().After("gorm:row").Register("dbtest:record:"+uuid.NewString(), func(db *gorm.DB) {
		if recording {
			queries = append(queries, db.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
		}
	})
	if err != nil {
		t.Fatalf("record queries: %v", err)
	}
	fn()
	recording = false

	var reads []int
	for _, query := range queries {
		var plan string
		if err := client.Raw("EXPLAIN (ANALYZE, FORMAT JSON) " + query).Scan(&plan).Error; err != nil {
			t.Fatalf("explain %s: %v", query, err)
		}
		var explained []struct{ Plan planNode }
		if err := json.Unmarshal([]byte(plan), &explained); err != nil || len(explained) != 1 {
			t.Fatalf("parse plan of %s: %v", query, err)
		}
		if rows, scans := explained[0].Plan.shareRows(); scans {
			reads = append(reads, int(rows))
		}
	}
	return reads
}

type planNode struct {
	Relation string     `json:"Relation Name"`
	Rows     float64    `json:"Actual Rows"`
	Loops    float64    `json:"Actual Loops"`
	Filtered float64    `json:"Rows Removed by Filter"`
	Plans    []planNode `json:"Plans"`
}

// shareRows sums the shld_shares rows the node and its children read, and
// reports whether any of them scans the table. EXPLAIN gives rows per loop.
func (n planNode) shareRows() (float64, bool) {
	var rows float64
	scans := n.Relation == "shld_shares"
	if scans {
		rows = (n.Rows + n.Filtered) * n.Loops
	}
	for _, child := range n.Plans {
		childRows, childScans := child.shareRows()
		rows += childRows
		scans = scans || childScans
	}
	return rows, scans
}

// WaitForLockWait returns true once a session is waiting for a lock held by
// backend pid, or false once finished is closed first.
func WaitForLockWait(t *testing.T, client *sql.Client, pid int, finished <-chan struct{}) bool {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-finished:
			return false
		default:
		}
		var waiting int64
		if err := client.Raw(`SELECT count(*) FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid))`, pid).Scan(&waiting).Error; err != nil {
			t.Fatalf("read lock waits: %v", err)
		}
		if waiting > 0 {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("nothing waited for backend %d within 10s", pid)
	return false
}

func cleanup(t *testing.T, client *sql.Client, projectID string) {
	t.Helper()
	stmts := []string{
		`DELETE FROM shld_passkey_references WHERE share_reference IN (SELECT s.id FROM shld_shares s JOIN shld_users u ON s.user_id = u.id WHERE u.project_id = ?)`,
		`DELETE FROM shld_shares WHERE user_id IN (SELECT id FROM shld_users WHERE project_id = ?)`,
		`DELETE FROM shld_keychains WHERE user_id IN (SELECT id FROM shld_users WHERE project_id = ?)`,
		`DELETE FROM shld_users WHERE project_id = ?`,
		`DELETE FROM shld_encryption_parts WHERE project_id = ?`,
		`DELETE FROM shld_shamir_migrations WHERE project_id = ?`,
		`DELETE FROM shld_projects WHERE id = ?`,
	}
	for _, stmt := range stmts {
		if err := client.Exec(stmt, projectID).Error; err != nil {
			t.Errorf("cleanup failed (%s): %v", stmt, err)
		}
	}
}
