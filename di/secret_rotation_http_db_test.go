package di

import (
	"net/http"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

// withSecret returns a client that authenticates with secret instead.
func (c httpClient) withSecret(secret string) httpClient {
	c.secret = secret
	return c
}

type secretRow struct {
	APISecret        string
	PendingAPISecret *string
	PendingAPIKey    *string
}

func readSecretRow(t *testing.T, apiKey string) secretRow {
	t.Helper()
	var row secretRow
	err := dbtest.RequireDB(t).Raw(`SELECT api_secret, pending_api_secret, pending_api_key FROM shld_projects WHERE api_key = ?`, apiKey).Scan(&row).Error
	if err != nil {
		t.Fatalf("read project row: %v", err)
	}
	return row
}

func TestAPISecretRotation_HTTP(t *testing.T) {
	c, key, _ := newProject(t)
	oldSecret := c.secret
	const commit = "/project/rotate-api-secret/commit"

	expect(t, "commit with nothing pending", c.do(http.MethodPost, commit, key, nil), http.StatusOK, "")

	lost := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil)
	expect(t, "prepare", lost, http.StatusOK, "")
	prepared := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil)
	expect(t, "prepare again", prepared, http.StatusOK, "")
	newSecret := prepared.str("api_secret")
	if len(newSecret) != 64 || newSecret == oldSecret {
		t.Fatalf("pending secret = %q, want a new hex-encoded 32-byte secret", newSecret)
	}

	row := readSecretRow(t, key)
	if row.PendingAPISecret == nil || *row.PendingAPISecret == newSecret || (*row.PendingAPISecret)[:4] != "$2a$" {
		t.Fatalf("stored pending secret = %v, want a bcrypt hash, not the secret", row.PendingAPISecret)
	}

	expect(t, "commit with a replaced pending secret", c.withSecret(lost.str("api_secret")).do(http.MethodPost, commit, key, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "commit with the current secret while pending", c.do(http.MethodPost, commit, key, nil), http.StatusConflict, "A_PENDING_NOT_PROVEN")
	expect(t, "pending secret on another route", c.withSecret(newSecret).do(http.MethodGet, "/project", key, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "pending secret on the key commit route", c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-key/commit", key, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "pending secret on secret prepare", c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "old secret still works before commit", c.do(http.MethodGet, "/project", key, nil), http.StatusOK, "")

	committed := c.withSecret(newSecret).do(http.MethodPost, commit, key, nil)
	expect(t, "commit with the pending secret", committed, http.StatusOK, "")
	if committed.str("status") != "committed" || committed.Body["api_secret"] != nil {
		t.Fatalf("commit body = %v, want status committed and no secret", committed.Body)
	}

	expect(t, "old secret after commit", c.do(http.MethodGet, "/project", key, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "old secret on the commit route", c.do(http.MethodPost, commit, key, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "new secret after commit", c.withSecret(newSecret).do(http.MethodGet, "/project", key, nil), http.StatusOK, "")

	retry := c.withSecret(newSecret).do(http.MethodPost, commit, key, nil)
	expect(t, "repeated commit", retry, http.StatusOK, "")
	if retry.str("status") != "current" {
		t.Errorf("repeated commit status = %q, want current", retry.str("status"))
	}

	row = readSecretRow(t, key)
	if row.PendingAPISecret != nil || row.APISecret == newSecret || row.APISecret[:4] != "$2a$" {
		t.Fatalf("row after commit = %+v, want a bcrypt hash and nothing pending", row)
	}

	c.withSecret(newSecret).do(http.MethodDelete, "/project", key, nil)
}

func TestAPISecretRotation_HTTP_Expired(t *testing.T) {
	c, key, _ := newProject(t)
	client := dbtest.RequireDB(t)
	const commit = "/project/rotate-api-secret/commit"

	pending := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil).str("api_secret")
	dbtest.Exec(t, client, `UPDATE shld_projects SET pending_api_secret_expires_at = now() - interval '1 minute' WHERE api_key = ?`, key)

	expect(t, "commit with an expired pending secret", c.withSecret(pending).do(http.MethodPost, commit, key, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "commit with the current secret after expiry", c.do(http.MethodPost, commit, key, nil), http.StatusConflict, "A_ROTATION_EXPIRED")
	expect(t, "current secret still works", c.do(http.MethodGet, "/project", key, nil), http.StatusOK, "")
}

// A key rotation and a secret rotation in flight at once each commit only
// their own credential; the secret commit must present the current key.
func TestAPISecretRotation_HTTP_InterleavedWithAPIKey(t *testing.T) {
	c, oldKey, _ := newProject(t)

	newKey := c.do(http.MethodPost, "/project/rotate-api-key/prepare", oldKey, nil).str("api_key")
	newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", oldKey, nil).str("api_secret")

	expect(t, "key commit", c.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil), http.StatusOK, "")
	if row := readSecretRow(t, newKey); row.PendingAPISecret == nil {
		t.Fatalf("key commit cleared the pending secret: %+v", row)
	}
	expect(t, "secret commit with the revoked key", c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", oldKey, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "secret commit with the new key", c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", newKey, nil), http.StatusOK, "")
	expect(t, "new key and new secret", c.withSecret(newSecret).do(http.MethodGet, "/project", newKey, nil), http.StatusOK, "")

	// The other order: the secret commits first, then the key.
	s := c.withSecret(newSecret)
	nextKey := s.do(http.MethodPost, "/project/rotate-api-key/prepare", newKey, nil).str("api_key")
	nextSecret := s.do(http.MethodPost, "/project/rotate-api-secret/prepare", newKey, nil).str("api_secret")
	expect(t, "secret commit first", s.withSecret(nextSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", newKey, nil), http.StatusOK, "")
	if row := readSecretRow(t, newKey); row.PendingAPIKey == nil {
		t.Fatalf("secret commit cleared the pending key: %+v", row)
	}
	expect(t, "key commit with the old secret", s.do(http.MethodPost, "/project/rotate-api-key/commit", nextKey, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "key commit with the new secret", s.withSecret(nextSecret).do(http.MethodPost, "/project/rotate-api-key/commit", nextKey, nil), http.StatusOK, "")
	expect(t, "both rotated", s.withSecret(nextSecret).do(http.MethodGet, "/project", nextKey, nil), http.StatusOK, "")

	s.withSecret(nextSecret).do(http.MethodDelete, "/project", nextKey, nil)
}
