package di

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestShareReads_BoundToAuthenticatedUser_HTTP(t *testing.T) {
	suffix := "-" + uuid.NewString()
	fakeOpenfortAPI(t, suffix)
	p := newUserProject(t)
	_, otherPEM := newSigner(t)
	expect(t, "second custom provider", p.do(http.MethodPost, "/project/providers", p.apiKey,
		map[string]any{"providers": map[string]any{"custom": map[string]any{"pem": otherPEM, "key_type": "rsa"}}}),
		http.StatusConflict, "PV_EXISTS")
	expect(t, "add openfort provider", p.do(http.MethodPost, "/project/providers", p.apiKey,
		map[string]any{"providers": map[string]any{"openfort": map[string]any{"publishable_key": "pk_test_" + uuid.NewString()}}}),
		http.StatusOK, "")

	asPlayer := func(method, path, token string, body any, headers ...string) reply {
		return p.do(method, path, "", body, append([]string{
			"X-API-Key", p.apiKey, "X-Auth-Provider", "openfort", "Authorization", "Bearer " + token,
			"X-Openfort-Provider", "firebase", "X-Openfort-Token-Type", "idToken",
		}, headers...)...)
	}
	sharedSubject := "pla_alice" + suffix
	part := []string{"X-Encryption-Part", p.part}

	expect(t, "alice registers her share", asPlayer(http.MethodPost, "/shares", "alice",
		map[string]any{"secret": "alice-secret", "entropy": "project", "encryption_part": p.part}), http.StatusCreated, "")

	for _, path := range []string{"/shares/default", "/shares"} {
		got := p.asUser(t, http.MethodGet, path, sharedSubject, nil, part...)
		if got.Status != http.StatusNotFound || got.str("secret") == "alice-secret" {
			t.Errorf("GET %s as the custom-provider user with alice's subject: got %d %v, want 404",
				path, got.Status, got.Body)
		}
	}

	attempts := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/keychain?reference=default", nil},
		{http.MethodPut, "/shares", map[string]any{"secret": "overwritten", "entropy": "project", "reference": "default", "encryption_part": p.part}},
		{http.MethodDelete, "/shares/default", nil},
		{http.MethodDelete, "/shares", nil},
	}
	for _, a := range attempts {
		got := p.asUser(t, a.method, a.path, sharedSubject, a.body, part...)
		if got.Status != http.StatusNotFound {
			t.Errorf("%s %s as the custom-provider user with alice's subject: got %d %v, want 404", a.method, a.path, got.Status, got.Body)
		}
	}

	for _, path := range []string{"/shares/default", "/shares"} {
		got := asPlayer(http.MethodGet, path, "alice", nil, part...)
		expect(t, "GET "+path+" as alice", got, http.StatusOK, "")
		if got.str("secret") != "alice-secret" {
			t.Errorf("GET %s as alice: secret = %q, want alice-secret", path, got.str("secret"))
		}
	}
}
