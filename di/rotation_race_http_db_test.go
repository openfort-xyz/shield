package di

import (
	"net/http"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

// queueOnProjectRow holds FOR UPDATE on the project row and its encryption
// part rows, and starts each request once the previous one is waiting behind
// that hold, then releases it. Waiters are granted the project row in arrival
// order, so the requests that lock it commit in the order given, each having
// authenticated against the state before any ran.
func queueOnProjectRow(t *testing.T, apiKey string, requests ...func() reply) []reply {
	t.Helper()
	client := dbtest.RequireDB(t)
	holder := client.Begin()
	if holder.Error != nil {
		t.Fatalf("begin: %v", holder.Error)
	}
	defer holder.Rollback()
	var holderPID int
	if err := holder.Raw(`SELECT pg_backend_pid()`).Scan(&holderPID).Error; err != nil {
		t.Fatalf("holder pid: %v", err)
	}
	if err := holder.Exec(`SELECT id FROM shld_projects WHERE api_key = ? FOR UPDATE`, apiKey).Error; err != nil {
		t.Fatalf("lock project row: %v", err)
	}
	if err := holder.Exec(`SELECT e.id FROM shld_encryption_parts e JOIN shld_projects p ON p.id = e.project_id
		WHERE p.api_key = ? FOR UPDATE OF e`, apiKey).Error; err != nil {
		t.Fatalf("lock encryption part rows: %v", err)
	}

	replies := make([]reply, len(requests))
	done := make([]chan struct{}, len(requests))
	for i, req := range requests {
		done[i] = make(chan struct{})
		go func() {
			defer close(done[i])
			replies[i] = req()
		}()
		waitForLockWaiters(t, holderPID, i+1, done[i])
	}
	if err := holder.Rollback().Error; err != nil {
		t.Fatalf("release project row: %v", err)
	}
	for _, d := range done {
		<-d
	}
	return replies
}

// waitForLockWaiters waits until want sessions are blocked by the holder,
// directly or behind another waiter queued on the same row.
func waitForLockWaiters(t *testing.T, holderPID, want int, finished <-chan struct{}) {
	t.Helper()
	client := dbtest.RequireDB(t)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-finished:
			t.Fatalf("request finished without waiting behind the held rows")
		default:
		}
		var waiting int
		err := client.Raw(`WITH RECURSIVE blocked(pid) AS (
				SELECT pid FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid))
				UNION
				SELECT a.pid FROM pg_stat_activity a JOIN blocked b ON b.pid = ANY(pg_blocking_pids(a.pid))
			) SELECT count(*) FROM blocked`, holderPID).Scan(&waiting).Error
		if err != nil {
			t.Fatalf("read lock waits: %v", err)
		}
		if waiting >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("fewer than %d requests waited behind the held rows within 10s", want)
}

// raceSecretCommit commits a new API secret while inflight, sent with the old
// secret, waits behind it: inflight authenticated before the commit landed.
func raceSecretCommit(t *testing.T, c httpClient, apiKey string, inflight func(old httpClient) reply) (string, reply) {
	t.Helper()
	newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", apiKey, nil).str("api_secret")
	got := queueOnProjectRow(t, apiKey,
		func() reply {
			return c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", apiKey, nil)
		},
		func() reply { return inflight(c) },
	)
	expect(t, "secret commit", got[0], http.StatusOK, "")
	return newSecret, got[1]
}

// A secret commit that lands while a key commit waits for the project row
// revokes the secret that key commit authenticated with.
func TestAPIKeyRotationCommit_RacingSecretCommit_RejectsRevokedSecret(t *testing.T) {
	c, oldKey, _ := newProject(t)
	oldSecret := c.secret
	newKey := c.do(http.MethodPost, "/project/rotate-api-key/prepare", oldKey, nil).str("api_key")
	newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", oldKey, nil).str("api_secret")

	got := queueOnProjectRow(t, oldKey,
		func() reply {
			return c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", oldKey, nil)
		},
		func() reply {
			return c.withSecret(oldSecret).do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil)
		},
	)

	expect(t, "secret commit", got[0], http.StatusOK, "")
	expect(t, "key commit with the secret revoked under it", got[1], http.StatusUnauthorized, "A_INVALID")
	expect(t, "old key still current", c.withSecret(newSecret).do(http.MethodGet, "/project", oldKey, nil), http.StatusOK, "")

	expect(t, "key commit with the new secret", c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil), http.StatusOK, "")
	c.withSecret(newSecret).do(http.MethodDelete, "/project", newKey, nil)
}

// A key commit that lands while a secret commit waits for the project row
// revokes the key that secret commit authenticated with.
func TestAPISecretRotationCommit_RacingKeyCommit_RejectsRevokedKey(t *testing.T) {
	c, oldKey, _ := newProject(t)
	newKey := c.do(http.MethodPost, "/project/rotate-api-key/prepare", oldKey, nil).str("api_key")
	newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", oldKey, nil).str("api_secret")

	got := queueOnProjectRow(t, oldKey,
		func() reply {
			return c.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil)
		},
		func() reply {
			return c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", oldKey, nil)
		},
	)

	expect(t, "key commit", got[0], http.StatusOK, "")
	expect(t, "secret commit with the key revoked under it", got[1], http.StatusUnauthorized, "A_INVALID")
	expect(t, "old secret still current", c.do(http.MethodGet, "/project", newKey, nil), http.StatusOK, "")

	expect(t, "secret commit with the new key", c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", newKey, nil), http.StatusOK, "")
	c.withSecret(newSecret).do(http.MethodDelete, "/project", newKey, nil)
}

func TestEncryptionKeyRotationPrepare_RacingSecretCommit_RejectsRevokedSecret(t *testing.T) {
	c, key, part := newProject(t)
	newSecret, got := raceSecretCommit(t, c, key, func(old httpClient) reply {
		return old.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part})
	})
	expect(t, "prepare with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")

	c = c.withSecret(newSecret)
	expect(t, "commit finds nothing pending", c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key,
		map[string]any{"encryption_part": part, "new_encryption_part": part}), http.StatusConflict, "EC_NO_PENDING")
	newPart := c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part}).str("encryption_part")
	committed := c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key, map[string]any{"encryption_part": part, "new_encryption_part": newPart})
	expect(t, "rotation with the new secret", committed, http.StatusOK, "")
	c.do(http.MethodDelete, "/project", key, nil)
}

func TestEncryptionKeyRotationCommit_RacingSecretCommit_RejectsRevokedSecret(t *testing.T) {
	c, key, part := newProject(t)
	newPart := c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part}).str("encryption_part")
	body := map[string]any{"encryption_part": part, "new_encryption_part": newPart}
	newSecret, got := raceSecretCommit(t, c, key, func(old httpClient) reply {
		return old.do(http.MethodPost, "/project/rotate-encryption-key/commit", key, body)
	})
	expect(t, "commit with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")

	c = c.withSecret(newSecret)
	expect(t, "old part still current", c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key,
		map[string]any{"encryption_part": newPart}), http.StatusBadRequest, "EC_INVALID")
	newPart = c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part}).str("encryption_part")
	committed := c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key, map[string]any{"encryption_part": part, "new_encryption_part": newPart})
	expect(t, "commit with the new secret", committed, http.StatusOK, "")
	if committed.str("status") != "committed" {
		t.Fatalf("commit with the new secret: %v, want committed", committed.Body)
	}
	c.do(http.MethodDelete, "/project", key, nil)
}

func TestAPIKeyRotationPrepare_RacingSecretCommit_RejectsRevokedSecret(t *testing.T) {
	c, key, _ := newProject(t)
	newSecret, got := raceSecretCommit(t, c, key, func(old httpClient) reply {
		return old.do(http.MethodPost, "/project/rotate-api-key/prepare", key, nil)
	})
	expect(t, "prepare with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")

	var pending *string
	dbtest.RequireDB(t).Raw(`SELECT pending_api_key FROM shld_projects WHERE api_key = ?`, key).Scan(&pending)
	if pending != nil {
		t.Fatalf("refused prepare stored a pending key")
	}
	c = c.withSecret(newSecret)
	newKey := c.do(http.MethodPost, "/project/rotate-api-key/prepare", key, nil).str("api_key")
	expect(t, "key rotation with the new secret", c.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil), http.StatusOK, "")
	c.do(http.MethodDelete, "/project", newKey, nil)
}

func TestDeleteProject_RacingSecretCommit_RejectsRevokedSecret(t *testing.T) {
	c, key, _ := newProject(t)
	newSecret, got := raceSecretCommit(t, c, key, func(old httpClient) reply {
		return old.do(http.MethodDelete, "/project", key, nil)
	})
	expect(t, "delete with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")

	c = c.withSecret(newSecret)
	expect(t, "project survives", c.do(http.MethodGet, "/project", key, nil), http.StatusOK, "")
	expect(t, "delete with the new secret", c.do(http.MethodDelete, "/project", key, nil), http.StatusNoContent, "")
	expect(t, "project gone", c.do(http.MethodGet, "/project", key, nil), http.StatusUnauthorized, "A_INVALID")
}
