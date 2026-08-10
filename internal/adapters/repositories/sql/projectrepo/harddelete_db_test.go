package projectrepo

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
)

// requireDB returns a SQL client or skips the test when no database is
// explicitly configured (CI currently runs without one; locally set DB_NAME).
func requireDB(t *testing.T) *sql.Client {
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

// seedShieldProject inserts a project with the full graph of dependent rows:
// provider (+openfort child), user, external user, share WITH a passkey
// reference (the FK that blocks the DB cascade), keychain, encryption part,
// notification, rate limit and a shamir migration record.
func seedShieldProject(t *testing.T, client *sql.Client, projectID string) {
	t.Helper()

	exec := func(query string, args ...interface{}) {
		t.Helper()
		if err := client.Exec(query, args...).Error; err != nil {
			t.Fatalf("seed failed (%s): %v", query, err)
		}
	}

	providerID, userID, externalUserID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	shareID, keychainID := uuid.NewString(), uuid.NewString()

	exec(`INSERT INTO shld_projects (id, name, api_key, api_secret, enable_2fa) VALUES (?, ?, ?, ?, false)`,
		projectID, "proj-"+projectID[:8], uuid.NewString(), uuid.NewString())
	exec(`INSERT INTO shld_providers (id, project_id, type) VALUES (?, ?, 'OPENFORT')`, providerID, projectID)
	exec(`INSERT INTO shld_openfort_providers (provider_id, publishable_key) VALUES (?, ?)`, providerID, "pk_test")
	exec(`INSERT INTO shld_users (id, project_id) VALUES (?, ?)`, userID, projectID)
	exec(`INSERT INTO shld_external_users (id, user_id, external_user_id, provider_id) VALUES (?, ?, ?, ?)`,
		externalUserID, userID, "ext-"+externalUserID[:8], providerID)
	exec(`INSERT INTO shld_keychains (id, user_id) VALUES (?, ?)`, keychainID, userID)
	exec(`INSERT INTO shld_shares (id, data, user_id, keychain_id, reference, storage_method_id) VALUES (?, ?, ?, ?, ?, (SELECT min(id) FROM shld_share_storage_methods))`,
		shareID, "secret-share-data", userID, keychainID, "ref-"+shareID[:8])
	exec(`INSERT INTO shld_passkey_references (passkey_id, passkey_env, share_reference) VALUES (?, 'test-env', ?)`,
		uuid.NewString(), shareID)
	exec(`INSERT INTO shld_encryption_parts (id, project_id, part) VALUES (?, ?, ?)`, uuid.NewString(), projectID, "db-part")
	exec(`INSERT INTO shld_notifications (project_id, external_user_id, notif_type, price, sent_at) VALUES (?, ?, 'SMS', 0.1, now())`,
		projectID, externalUserID)
	exec(`INSERT INTO shld_rate_limit (project_id, email_requests_per_hour, sms_requests_per_hour) VALUES (?, 120, 2)`, projectID)
	exec(`INSERT INTO shld_shamir_migrations (id, project_id, success) VALUES (?, ?, true)`, uuid.NewString(), projectID)
}

// countShieldProjectRows counts surviving rows per table for the project.
func countShieldProjectRows(t *testing.T, client *sql.Client, projectID string) map[string]int64 {
	t.Helper()

	queries := map[string]string{
		"shld_projects":           `SELECT count(*) FROM shld_projects WHERE id = ?`,
		"shld_providers":          `SELECT count(*) FROM shld_providers WHERE project_id = ?`,
		"shld_openfort_providers": `SELECT count(*) FROM shld_openfort_providers WHERE provider_id IN (SELECT id FROM shld_providers WHERE project_id = ?)`,
		"shld_users":              `SELECT count(*) FROM shld_users WHERE project_id = ?`,
		"shld_external_users":     `SELECT count(*) FROM shld_external_users WHERE user_id IN (SELECT id FROM shld_users WHERE project_id = ?)`,
		"shld_keychains":          `SELECT count(*) FROM shld_keychains WHERE user_id IN (SELECT id FROM shld_users WHERE project_id = ?)`,
		"shld_shares":             `SELECT count(*) FROM shld_shares WHERE user_id IN (SELECT id FROM shld_users WHERE project_id = ?)`,
		"shld_passkey_references": `SELECT count(*) FROM shld_passkey_references WHERE share_reference IN (SELECT s.id FROM shld_shares s JOIN shld_users u ON s.user_id = u.id WHERE u.project_id = ?)`,
		"shld_encryption_parts":   `SELECT count(*) FROM shld_encryption_parts WHERE project_id = ?`,
		"shld_notifications":      `SELECT count(*) FROM shld_notifications WHERE project_id = ?`,
		"shld_rate_limit":         `SELECT count(*) FROM shld_rate_limit WHERE project_id = ?`,
		"shld_shamir_migrations":  `SELECT count(*) FROM shld_shamir_migrations WHERE project_id = ?`,
	}
	counts := map[string]int64{}
	for table, query := range queries {
		var n int64
		if err := client.Raw(query, projectID).Scan(&n).Error; err != nil {
			t.Fatalf("counting %s failed: %v", table, err)
		}
		counts[table] = n
	}
	return counts
}

func TestHardDelete(t *testing.T) {
	client := requireDB(t)
	ctx := context.Background()

	projectA, projectB := uuid.NewString(), uuid.NewString()
	cleanup := func(projectID string) {
		client.Exec(`DELETE FROM shld_passkey_references WHERE share_reference IN (SELECT s.id FROM shld_shares s JOIN shld_users u ON s.user_id = u.id WHERE u.project_id = ?)`, projectID)
		client.Exec(`DELETE FROM shld_shamir_migrations WHERE project_id = ?`, projectID)
		client.Exec(`DELETE FROM shld_projects WHERE id = ?`, projectID)
	}
	defer cleanup(projectA)
	defer cleanup(projectB)

	seedShieldProject(t, client, projectA)
	seedShieldProject(t, client, projectB)

	repo := New(client).(*repository)

	if err := repo.HardDelete(ctx, projectA); err != nil {
		t.Fatalf("HardDelete failed: %v", err)
	}

	for table, n := range countShieldProjectRows(t, client, projectA) {
		if n != 0 {
			t.Errorf("project A still has %d rows in %s after HardDelete", n, table)
		}
	}
	for table, n := range countShieldProjectRows(t, client, projectB) {
		if n != 1 {
			t.Errorf("project B (untouched) rows in %s: got %d, want 1", table, n)
		}
	}

	// Idempotency: hard-deleting an absent project succeeds.
	if err := repo.HardDelete(ctx, projectA); err != nil {
		t.Fatalf("second HardDelete failed: %v", err)
	}
}

// TestHardDelete_SoftDeletedRows verifies that rows GORM soft-deleted earlier
// are also physically removed by the hard delete.
func TestHardDelete_SoftDeletedRows(t *testing.T) {
	client := requireDB(t)
	ctx := context.Background()

	projectID := uuid.NewString()
	defer client.Exec(`DELETE FROM shld_projects WHERE id = ?`, projectID)

	seedShieldProject(t, client, projectID)
	// Soft-delete the project first (the register-rollback path does this).
	if err := client.Delete(&Project{}, "id = ?", projectID).Error; err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	repo := New(client).(*repository)
	if err := repo.HardDelete(ctx, projectID); err != nil {
		t.Fatalf("HardDelete of soft-deleted project failed: %v", err)
	}

	var n int64
	if err := client.Raw(`SELECT count(*) FROM shld_projects WHERE id = ?`, projectID).Scan(&n).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if n != 0 {
		t.Errorf("soft-deleted project row survived HardDelete (%d rows)", n)
	}
}
