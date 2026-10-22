package di

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

// httpClient drives the real router (routes, middleware, CORS) in-process.
type httpClient struct {
	t       *testing.T
	handler http.Handler
	secret  string
}

type reply struct {
	Status int
	Body   map[string]any
}

func (c httpClient) do(method, path, apiKey string, body any, headers ...string) reply {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			c.t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
		req.Header.Set("X-API-Secret", c.secret)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)

	out := reply{Status: rec.Code, Body: map[string]any{}}
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out.Body)
	}
	return out
}

func (r reply) str(key string) string {
	s, _ := r.Body[key].(string)
	return s
}

// newProject registers a project over HTTP and returns a client for it with
// its API key and encryption part. The project is hard-deleted afterwards.
func newProject(t *testing.T) (httpClient, string, string) {
	t.Helper()
	dbtest.RequireDB(t)
	srv, err := ProvideRESTServer()
	if err != nil {
		t.Fatalf("ProvideRESTServer: %v", err)
	}
	c := httpClient{t: t, handler: srv.Handler()}

	res := c.do(http.MethodPost, "/register", "", map[string]any{"name": "rotation-" + uuid.NewString()[:8], "generate_encryption_key": true})
	if res.Status != http.StatusCreated {
		t.Fatalf("register: %d %v", res.Status, res.Body)
	}
	c.secret = res.str("api_secret")
	apiKey := res.str("api_key")

	// Tests that rotate the key delete the project themselves.
	t.Cleanup(func() { c.do(http.MethodDelete, "/project", apiKey, nil) })
	return c, apiKey, res.str("encryption_part")
}

func expect(t *testing.T, step string, got reply, status int, code string) {
	t.Helper()
	if got.Status != status || (code != "" && got.str("code") != code) {
		t.Fatalf("%s: got %d %v, want %d %s", step, got.Status, got.Body, status, code)
	}
}

func TestAPIKeyRotation_HTTP(t *testing.T) {
	c, oldKey, _ := newProject(t)

	// Prime the project-service cache with the old key on a cached route.
	// The request then fails on the auth provider, after the key lookup.
	const invalidKey = "Invalid API key or API secret"
	primed := c.do(http.MethodGet, "/shares", oldKey, nil, "X-Auth-Provider", "custom")
	if primed.str("message") == invalidKey {
		t.Fatalf("old key rejected before rotation: %v", primed.Body)
	}

	expect(t, "commit with nothing pending", c.do(http.MethodPost, "/project/rotate-api-key/commit", oldKey, nil), http.StatusOK, "")

	lost := c.do(http.MethodPost, "/project/rotate-api-key/prepare", oldKey, nil)
	expect(t, "prepare", lost, http.StatusOK, "")
	prepared := c.do(http.MethodPost, "/project/rotate-api-key/prepare", oldKey, nil)
	expect(t, "prepare again", prepared, http.StatusOK, "")
	newKey := prepared.str("api_key")

	expect(t, "commit with a replaced pending key", c.do(http.MethodPost, "/project/rotate-api-key/commit", lost.str("api_key"), nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "commit with the current key while pending", c.do(http.MethodPost, "/project/rotate-api-key/commit", oldKey, nil), http.StatusConflict, "A_PENDING_NOT_PROVEN")
	expect(t, "pending key on another route", c.do(http.MethodGet, "/project", newKey, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "old key still works before commit", c.do(http.MethodGet, "/project", oldKey, nil), http.StatusOK, "")

	committed := c.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil)
	expect(t, "commit with the pending key", committed, http.StatusOK, "")
	if committed.str("status") != "committed" || committed.str("api_key") != newKey {
		t.Fatalf("commit body = %v, want status committed and the new key", committed.Body)
	}

	expect(t, "old key after commit", c.do(http.MethodGet, "/project", oldKey, nil), http.StatusUnauthorized, "A_INVALID")
	if cached := c.do(http.MethodGet, "/shares", oldKey, nil, "X-Auth-Provider", "custom"); cached.str("message") != invalidKey {
		t.Fatalf("old key on a cached route after commit: %v, want %q (cache not evicted)", cached.Body, invalidKey)
	}
	expect(t, "new key after commit", c.do(http.MethodGet, "/project", newKey, nil), http.StatusOK, "")

	retry := c.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil)
	expect(t, "repeated commit", retry, http.StatusOK, "")
	if retry.str("status") != "current" {
		t.Errorf("repeated commit status = %q, want current", retry.str("status"))
	}

	c.do(http.MethodDelete, "/project", newKey, nil)
}

func TestAPIKeyRotation_HTTP_Expired(t *testing.T) {
	c, key, _ := newProject(t)
	client := dbtest.RequireDB(t)

	pending := c.do(http.MethodPost, "/project/rotate-api-key/prepare", key, nil).str("api_key")
	dbtest.Exec(t, client, `UPDATE shld_projects SET pending_api_key_expires_at = now() - interval '1 minute' WHERE api_key = ?`, key)

	expect(t, "commit with an expired pending key", c.do(http.MethodPost, "/project/rotate-api-key/commit", pending, nil), http.StatusUnauthorized, "A_INVALID")
	expect(t, "commit with the current key after expiry", c.do(http.MethodPost, "/project/rotate-api-key/commit", key, nil), http.StatusConflict, "A_ROTATION_EXPIRED")
	expect(t, "current key still works", c.do(http.MethodGet, "/project", key, nil), http.StatusOK, "")
}

func TestEncryptionKeyRotation_HTTP(t *testing.T) {
	c, key, part := newProject(t)

	expect(t, "prepare without a part", c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{}), http.StatusBadRequest, "")
	expect(t, "commit without prepare", c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key,
		map[string]any{"encryption_part": part, "new_encryption_part": part}), http.StatusConflict, "EC_NO_PENDING")

	prepared := c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part})
	expect(t, "prepare", prepared, http.StatusOK, "")
	newPart := prepared.str("encryption_part")

	body := map[string]any{"encryption_part": part, "new_encryption_part": newPart}
	first := c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key, body)
	expect(t, "commit", first, http.StatusOK, "")
	again := c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key, body)
	expect(t, "repeated commit", again, http.StatusOK, "")
	if first.str("status") != "committed" || again.str("status") != "already_committed" {
		t.Errorf("statuses = %q then %q, want committed then already_committed", first.str("status"), again.str("status"))
	}

	expect(t, "prepare with the old part", c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key,
		map[string]any{"encryption_part": part}), http.StatusBadRequest, "EC_INVALID")
}
