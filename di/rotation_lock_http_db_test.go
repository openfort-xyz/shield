package di

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

// gatedBody is a request body whose sender stalls after the headers until
// release is closed.
type gatedBody struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	r       io.Reader
}

func (g *gatedBody) Read(p []byte) (int, error) {
	g.once.Do(func() { close(g.started) })
	<-g.release
	return g.r.Read(p)
}

// stallAcross sends a request whose body stalls once the middleware has
// authenticated it, runs event to completion, then lets the body through.
func stallAcross(t *testing.T, payload any, send func(body io.Reader) reply, event func()) reply {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}
	body := &gatedBody{started: make(chan struct{}), release: make(chan struct{}), r: bytes.NewReader(raw)}
	var got reply
	done := make(chan struct{})
	go func() {
		defer close(done)
		got = send(body)
	}()
	select {
	case <-body.started:
	case <-done:
		t.Fatalf("request finished without reading its body: %d %v", got.Status, got.Body)
	case <-time.After(10 * time.Second):
		t.Fatalf("handler never read the body")
	}

	event()
	close(body.release)
	<-done
	return got
}

// stallAcrossSecretCommit stalls a request across a completed API secret
// commit and returns the new secret.
func stallAcrossSecretCommit(t *testing.T, c httpClient, apiKey, method, path string, payload any) (string, reply) {
	t.Helper()
	var newSecret string
	got := stallAcross(t, payload, func(body io.Reader) reply { return c.do(method, path, apiKey, body) }, func() {
		newSecret = commitNewSecret(t, c, apiKey)
	})
	return newSecret, got
}

func commitNewSecret(t *testing.T, c httpClient, apiKey string) string {
	t.Helper()
	newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", apiKey, nil).str("api_secret")
	expect(t, "secret commit", c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", apiKey, nil), http.StatusOK, "")
	expect(t, "old secret refused", c.do(http.MethodGet, "/project", apiKey, nil), http.StatusUnauthorized, "A_INVALID")
	return newSecret
}

func customProvider(pem string) map[string]any {
	return map[string]any{"providers": map[string]any{"custom": map[string]any{"pem": pem, "key_type": "rsa"}}}
}

func providerCount(t *testing.T, c httpClient, apiKey string) int {
	t.Helper()
	res := c.do(http.MethodGet, "/project/providers", apiKey, nil)
	expect(t, "list providers", res, http.StatusOK, "")
	providers, _ := res.Body["providers"].([]any)
	return len(providers)
}

func TestUpdateProvider_StalledAcrossSecretCommit_RejectsRevokedSecret(t *testing.T) {
	p := newUserProject(t)
	path := "/project/providers/" + p.providerID
	before := p.do(http.MethodGet, path, p.apiKey, nil).str("pem")
	_, otherPEM := newSigner(t)

	newSecret, got := stallAcrossSecretCommit(t, p.httpClient, p.apiKey, http.MethodPut, path, map[string]any{"pem": otherPEM, "key_type": "rsa"})
	expect(t, "provider update with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")

	owner := p.withSecret(newSecret)
	if owner.do(http.MethodGet, path, p.apiKey, nil).str("pem") != before {
		t.Fatalf("refused update changed the provider key")
	}
	expect(t, "update with the new secret", owner.do(http.MethodPut, path, p.apiKey, map[string]any{"pem": otherPEM, "key_type": "rsa"}), http.StatusOK, "")
	if owner.do(http.MethodGet, path, p.apiKey, nil).str("pem") != otherPEM {
		t.Fatalf("update with the new secret did not change the provider key")
	}
	owner.do(http.MethodDelete, "/project", p.apiKey, nil)
}

func TestAddProviders_StalledAcrossSecretCommit_RejectsRevokedSecret(t *testing.T) {
	c, key, _ := newProject(t)
	_, pem := newSigner(t)

	newSecret, got := stallAcrossSecretCommit(t, c, key, http.MethodPost, "/project/providers", customProvider(pem))
	expect(t, "provider add with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")

	owner := c.withSecret(newSecret)
	if n := providerCount(t, owner, key); n != 0 {
		t.Fatalf("refused add left %d providers", n)
	}
	expect(t, "add with the new secret", owner.do(http.MethodPost, "/project/providers", key, customProvider(pem)), http.StatusOK, "")
	owner.do(http.MethodDelete, "/project", key, nil)
}

func TestProviderWrites_RacingSecretCommit_RejectsRevokedSecret(t *testing.T) {
	_, otherPEM := newSigner(t)
	writes := map[string]func(old httpClient, key, providerID string) reply{
		"add": func(old httpClient, key, _ string) reply {
			return old.do(http.MethodPost, "/project/providers", key,
				map[string]any{"providers": map[string]any{"openfort": map[string]any{"publishable_key": "pk_" + uuid.NewString()}}})
		},
		"update": func(old httpClient, key, providerID string) reply {
			return old.do(http.MethodPut, "/project/providers/"+providerID, key, map[string]any{"pem": otherPEM, "key_type": "rsa"})
		},
		"delete": func(old httpClient, key, providerID string) reply {
			return old.do(http.MethodDelete, "/project/providers/"+providerID, key, nil)
		},
	}
	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			p := newUserProject(t)
			path := "/project/providers/" + p.providerID
			before := p.do(http.MethodGet, path, p.apiKey, nil).str("pem")

			newSecret, got := raceSecretCommit(t, p.httpClient, p.apiKey, func(old httpClient) reply {
				return write(old, p.apiKey, p.providerID)
			})
			expect(t, name+" with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")

			owner := p.withSecret(newSecret)
			if n := providerCount(t, owner, p.apiKey); n != 1 {
				t.Fatalf("refused %s left %d providers, want 1", name, n)
			}
			if owner.do(http.MethodGet, path, p.apiKey, nil).str("pem") != before {
				t.Fatalf("refused %s changed the provider key", name)
			}
			expect(t, name+" with the new secret", write(owner, p.apiKey, p.providerID), http.StatusOK, "")
			owner.do(http.MethodDelete, "/project", p.apiKey, nil)
		})
	}
}

func TestRegisterEncryptionKey_RacingSecretCommit_RejectsRevokedSecret(t *testing.T) {
	dbtest.RequireDB(t)
	srv, err := ProvideRESTServer()
	if err != nil {
		t.Fatalf("ProvideRESTServer: %v", err)
	}
	c := httpClient{t: t, handler: srv.Handler()}
	res := c.do(http.MethodPost, "/register", "", map[string]any{"name": "rotation-" + uuid.NewString()[:8]})
	expect(t, "register without a key", res, http.StatusCreated, "")
	c.secret = res.str("api_secret")
	key := res.str("api_key")

	newSecret, got := raceSecretCommit(t, c, key, func(old httpClient) reply {
		return old.do(http.MethodPost, "/project/encryption-key", key, nil)
	})
	expect(t, "key registration with the secret revoked under it", got, http.StatusUnauthorized, "A_INVALID")
	if got.str("encryption_part") != "" {
		t.Fatalf("refused registration returned a key part")
	}

	owner := c.withSecret(newSecret)
	registered := owner.do(http.MethodPost, "/project/encryption-key", key, nil)
	expect(t, "key registration with the new secret", registered, http.StatusOK, "")
	if registered.str("encryption_part") == "" {
		t.Fatalf("registration with the new secret returned no key part")
	}
	owner.do(http.MethodDelete, "/project", key, nil)
}

// A disable that lands while project writes wait for the row refuses them
// as the middleware would have.
func TestProjectWrites_RacingDisable_RefuseDisabledProject(t *testing.T) {
	t.Setenv("SHIELD_SERVICE_TOKEN", testServiceToken)
	c, key, part := newProject(t)
	newKey := c.do(http.MethodPost, "/project/rotate-api-key/prepare", key, nil).str("api_key")
	newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil).str("api_secret")
	_, pem := newSigner(t)

	got := queueOnProjectRow(t, key,
		func() reply { return c.setDisabled("disable", key) },
		func() reply {
			return c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part})
		},
		func() reply { return c.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil) },
		func() reply {
			return c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", key, nil)
		},
		func() reply { return c.do(http.MethodPost, "/project/providers", key, customProvider(pem)) },
	)
	expect(t, "disable", got[0], http.StatusNoContent, "")
	for i, step := range []string{"encryption prepare", "API key commit", "API secret commit", "provider add"} {
		expectDisabled(t, step+" queued behind the disable", got[i+1])
	}

	expect(t, "enable", c.setDisabled("enable", key), http.StatusNoContent, "")
	expect(t, "key and secret unchanged", c.do(http.MethodGet, "/project", key, nil), http.StatusOK, "")
	if n := providerCount(t, c, key); n != 0 {
		t.Fatalf("refused provider add left %d providers", n)
	}
	expect(t, "nothing pending", c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key,
		map[string]any{"encryption_part": part, "new_encryption_part": part}), http.StatusConflict, "EC_NO_PENDING")
}

func TestEncryptionKeyRotation_RacingDelete_ReturnsNotFound(t *testing.T) {
	c, key, part := newProject(t)
	newPart := c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part}).str("encryption_part")

	got := queueOnProjectRow(t, key,
		func() reply { return c.do(http.MethodDelete, "/project", key, nil) },
		func() reply {
			return c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": part})
		},
		func() reply {
			return c.do(http.MethodPost, "/project/rotate-encryption-key/commit", key,
				map[string]any{"encryption_part": part, "new_encryption_part": newPart})
		},
	)
	expect(t, "delete", got[0], http.StatusNoContent, "")
	expect(t, "prepare queued behind the delete", got[1], http.StatusNotFound, "PJ_NOT_FOUND")
	expect(t, "commit queued behind the delete", got[2], http.StatusNotFound, "PJ_NOT_FOUND")
}
