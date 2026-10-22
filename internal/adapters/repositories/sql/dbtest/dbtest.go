// Package dbtest seeds and inspects a real database for repository and
// application tests. Tests using it skip unless DB_NAME is set.
package dbtest

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
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
