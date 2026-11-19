package di

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

type stallEvent struct {
	name string
	run  func(t *testing.T, p userProject)
	want func(t *testing.T, step string, got reply)
}

func expectRevoked(t *testing.T, step string, got reply) {
	t.Helper()
	expect(t, step, got, http.StatusUnauthorized, "A_INVALID")
}

func expectRefused(t *testing.T, step string, got reply) {
	t.Helper()
	if got.Status < 400 {
		t.Fatalf("%s: got %d %v, want an error", step, got.Status, got.Body)
	}
}

func commitNewKey(t *testing.T, c httpClient, apiKey string) string {
	t.Helper()
	newKey := c.do(http.MethodPost, "/project/rotate-api-key/prepare", apiKey, nil).str("api_key")
	expect(t, "key commit", c.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil), http.StatusOK, "")
	return newKey
}

// stallEvents are the changes that must refuse a request which authenticated
// before them and sent its body after them. A nil run is the control: the
// stalled request with nothing in between must succeed.
func stallEvents() []stallEvent {
	return []stallEvent{
		{name: "nothing"},
		{"secret commit", func(t *testing.T, p userProject) {
			owner := p.withSecret(commitNewSecret(t, p.httpClient, p.apiKey))
			t.Cleanup(func() { owner.do(http.MethodDelete, "/project", p.apiKey, nil) })
		}, expectRevoked},
		{"key commit", func(t *testing.T, p userProject) {
			newKey := commitNewKey(t, p.httpClient, p.apiKey)
			t.Cleanup(func() { p.do(http.MethodDelete, "/project", newKey, nil) })
		}, expectRevoked},
		{"disable", func(t *testing.T, p userProject) {
			expect(t, "disable", p.setDisabled("disable", p.apiKey), http.StatusNoContent, "")
		}, expectDisabled},
		{"delete", func(t *testing.T, p userProject) {
			expect(t, "delete", p.do(http.MethodDelete, "/project", p.apiKey, nil), http.StatusNoContent, "")
		}, expectRevoked},
	}
}

type stalledRoute struct {
	name string
	ok   int
	// userRoute routes authenticate with the API key and a user token; the
	// API secret plays no part.
	userRoute bool
	// prepare runs before the stall and returns the body and the sender.
	prepare func(t *testing.T, p userProject) (any, func(body io.Reader) reply)
}

func createUser(t *testing.T, p userProject, externalUserID string) string {
	t.Helper()
	res := p.do(http.MethodPost, "/user", p.apiKey, map[string]any{"external_user_id": externalUserID, "provider_id": p.providerID})
	expect(t, "create user", res, http.StatusCreated, "")
	return res.str("user_id")
}

func secretRoute(p userProject, path string, headers ...string) func(body io.Reader) reply {
	return func(body io.Reader) reply { return p.do(http.MethodPost, path, p.apiKey, body, headers...) }
}

func plainShare() map[string]any {
	return map[string]any{"secret": "planted-share", "entropy": "user", "salt": "s", "iterations": 1000, "length": 32, "digest": "SHA-256"}
}

func stalledRoutes() []stalledRoute {
	return []stalledRoute{
		{name: "share import", ok: http.StatusCreated, prepare: func(t *testing.T, p userProject) (any, func(io.Reader) reply) {
			userID := createUser(t, p, "user-"+uuid.NewString())
			body := plainShare()
			body["user_id"] = userID
			return body, secretRoute(p, "/shares/migration/import")
		}},
		{name: "preregister", ok: http.StatusCreated, prepare: func(_ *testing.T, p userProject) (any, func(io.Reader) reply) {
			return plainShare(), secretRoute(p, "/admin/preregister", "X-User-ID", "user-"+uuid.NewString(), "X-Auth-Provider", "custom")
		}},
		{name: "create user", ok: http.StatusCreated, prepare: func(_ *testing.T, p userProject) (any, func(io.Reader) reply) {
			return map[string]any{"external_user_id": "user-" + uuid.NewString(), "provider_id": p.providerID}, secretRoute(p, "/user")
		}},
		{name: "encrypt", ok: http.StatusOK, prepare: func(_ *testing.T, p userProject) (any, func(io.Reader) reply) {
			return map[string]any{"encryption_part": p.part}, secretRoute(p, "/project/encrypt")
		}},
		{name: "encryption session", ok: http.StatusOK, prepare: func(_ *testing.T, p userProject) (any, func(io.Reader) reply) {
			return map[string]any{"encryption_part": p.part, "user_id": "user-" + uuid.NewString()},
				secretRoute(p, "/project/encryption-session")
		}},
		{name: "encryption lookup", ok: http.StatusOK, prepare: func(_ *testing.T, p userProject) (any, func(io.Reader) reply) {
			return map[string]any{"references": []string{"ref-" + uuid.NewString()}}, secretRoute(p, "/shares/encryption/reference/bulk")
		}},
		{name: "enable 2FA", ok: http.StatusOK, prepare: func(_ *testing.T, p userProject) (any, func(io.Reader) reply) {
			return map[string]any{}, secretRoute(p, "/project/enable-2fa")
		}},
		{name: "user share", ok: http.StatusCreated, userRoute: true, prepare: func(t *testing.T, p userProject) (any, func(io.Reader) reply) {
			token := p.token(t, "user-"+uuid.NewString())
			return plainShare(), func(body io.Reader) reply {
				return p.do(http.MethodPost, "/shares", "", body,
					"X-API-Key", p.apiKey, "X-Auth-Provider", "custom", "Authorization", "Bearer "+token)
			}
		}},
	}
}

// A request that authenticated, then held its body while credentials were
// rotated or the project was disabled or deleted, is judged by the state
// after its body arrived.
func TestStalledBody_CheckedAfterBodyIsRead(t *testing.T) {
	t.Setenv("SHIELD_SERVICE_TOKEN", testServiceToken)
	server := newClient(t)
	for _, route := range stalledRoutes() {
		for _, event := range stallEvents() {
			if route.userRoute && event.name == "secret commit" {
				continue
			}
			t.Run(route.name+" across "+event.name, func(t *testing.T) {
				t.Parallel()
				p := newUserProjectOn(t, server)
				payload, send := route.prepare(t, p)
				got := stallAcross(t, payload, send, func() {
					if event.run != nil {
						event.run(t, p)
					}
				})
				if event.run == nil {
					expect(t, route.name+" with a slow body", got, route.ok, "")
					return
				}
				want := event.want
				if route.userRoute && event.name == "delete" {
					// Delete does not evict the API-key cache the user routes
					// read, so the write fails on the deleted rows instead.
					want = expectRefused
				}
				want(t, route.name+" stalled across "+event.name, got)
			})
		}
	}
}

// projectState is what a refused write must leave unchanged.
func projectState(t *testing.T, c httpClient, apiKey string) string {
	t.Helper()
	list := c.do(http.MethodGet, "/project/providers", apiKey, nil)
	expect(t, "list providers", list, http.StatusOK, "")
	providers, _ := list.Body["providers"].([]any)
	var details []map[string]any
	for _, prov := range providers {
		id, _ := prov.(map[string]any)["provider_id"].(string)
		detail := c.do(http.MethodGet, "/project/providers/"+id, apiKey, nil)
		expect(t, "get provider", detail, http.StatusOK, "")
		details = append(details, detail.Body)
	}
	var row struct {
		Enable2FA bool
		Pending   int
	}
	err := dbtest.RequireDB(t).Raw(`SELECT p.enable_2fa,
			(SELECT count(*) FROM shld_encryption_parts e WHERE e.project_id = p.id AND e.pending_part IS NOT NULL) AS pending
		FROM shld_projects p WHERE p.api_key = ?`, apiKey).Scan(&row).Error
	if err != nil {
		t.Fatalf("read project row: %v", err)
	}
	raw, err := json.Marshal(map[string]any{"providers": details, "enable_2fa": row.Enable2FA, "pending_parts": row.Pending})
	if err != nil {
		t.Fatalf("encode state: %v", err)
	}
	return string(raw)
}

func addOpenfort(t *testing.T, c httpClient, apiKey string) string {
	t.Helper()
	res := c.do(http.MethodPost, "/project/providers", apiKey,
		map[string]any{"providers": map[string]any{"openfort": map[string]any{"publishable_key": "pk_" + uuid.NewString()}}})
	expect(t, "add openfort provider", res, http.StatusOK, "")
	providers, _ := res.Body["providers"].([]any)
	id, _ := providers[0].(map[string]any)["provider_id"].(string)
	return id
}

type queuedWrite struct {
	name string
	// prepare runs before the race and returns the write, sent with the
	// given client and API key.
	prepare func(t *testing.T, p userProject) func(c httpClient, apiKey string) reply
}

func queuedWrites(otherPEM string) []queuedWrite {
	return []queuedWrite{
		{"enable 2FA", func(*testing.T, userProject) func(httpClient, string) reply {
			return func(c httpClient, key string) reply { return c.do(http.MethodPost, "/project/enable-2fa", key, nil) }
		}},
		{"openfort provider update", func(t *testing.T, p userProject) func(httpClient, string) reply {
			id := addOpenfort(t, p.httpClient, p.apiKey)
			return func(c httpClient, key string) reply {
				return c.do(http.MethodPut, "/project/providers/"+id, key, map[string]any{"publishable_key": "pk_" + uuid.NewString()})
			}
		}},
		{"cookie-only provider update", func(_ *testing.T, p userProject) func(httpClient, string) reply {
			return func(c httpClient, key string) reply {
				return c.do(http.MethodPut, "/project/providers/"+p.providerID, key, map[string]any{"cookie_field_name": "session"})
			}
		}},
		{"PEM provider update", func(_ *testing.T, p userProject) func(httpClient, string) reply {
			return func(c httpClient, key string) reply {
				return c.do(http.MethodPut, "/project/providers/"+p.providerID, key, map[string]any{"pem": otherPEM, "key_type": "rsa"})
			}
		}},
		{"provider add", func(*testing.T, userProject) func(httpClient, string) reply {
			return func(c httpClient, key string) reply {
				return c.do(http.MethodPost, "/project/providers", key,
					map[string]any{"providers": map[string]any{"openfort": map[string]any{"publishable_key": "pk_" + uuid.NewString()}}})
			}
		}},
		{"provider delete", func(_ *testing.T, p userProject) func(httpClient, string) reply {
			return func(c httpClient, key string) reply {
				return c.do(http.MethodDelete, "/project/providers/"+p.providerID, key, nil)
			}
		}},
		{"encryption key prepare", func(_ *testing.T, p userProject) func(httpClient, string) reply {
			return func(c httpClient, key string) reply {
				return c.do(http.MethodPost, "/project/rotate-encryption-key/prepare", key, map[string]any{"encryption_part": p.part})
			}
		}},
	}
}

// A write that authenticated before a credential commit and waits for the
// project row behind it is refused and changes nothing.
func TestProjectWrites_QueuedBehindCredentialCommit_RejectRevokedCredential(t *testing.T) {
	commits := map[string]func(t *testing.T, p userProject, write func(httpClient, string) reply) (reply, httpClient, string){
		"secret": func(t *testing.T, p userProject, write func(httpClient, string) reply) (reply, httpClient, string) {
			newSecret, got := raceSecretCommit(t, p.httpClient, p.apiKey, func(old httpClient) reply { return write(old, p.apiKey) })
			return got, p.withSecret(newSecret), p.apiKey
		},
		"key": func(t *testing.T, p userProject, write func(httpClient, string) reply) (reply, httpClient, string) {
			newKey := p.do(http.MethodPost, "/project/rotate-api-key/prepare", p.apiKey, nil).str("api_key")
			got := queueOnProjectRow(t, p.apiKey,
				func() reply { return p.do(http.MethodPost, "/project/rotate-api-key/commit", newKey, nil) },
				func() reply { return write(p.httpClient, p.apiKey) },
			)
			expect(t, "key commit", got[0], http.StatusOK, "")
			return got[1], p.httpClient, newKey
		},
	}
	_, otherPEM := newSigner(t)
	server := newClient(t)
	for _, w := range queuedWrites(otherPEM) {
		for commit, race := range commits {
			t.Run(w.name+" behind a "+commit+" commit", func(t *testing.T) {
				t.Parallel()
				p := newUserProjectOn(t, server)
				write := w.prepare(t, p)
				before := projectState(t, p.httpClient, p.apiKey)

				got, owner, key := race(t, p, write)
				t.Cleanup(func() { owner.do(http.MethodDelete, "/project", key, nil) })
				expectRevoked(t, w.name+" with the "+commit+" revoked under it", got)
				if after := projectState(t, owner, key); after != before {
					t.Fatalf("refused %s changed the project:\nbefore %s\nafter  %s", w.name, before, after)
				}
				if res := write(owner, key); res.Status >= 300 {
					t.Fatalf("%s with the new %s: %d %v", w.name, commit, res.Status, res.Body)
				}
			})
		}
	}
}

// A request that writes several provider rows either applies them all or,
// refused, none: a credential commit cannot land between them.
func TestProviderMultiWrites_AreAtomic(t *testing.T) {
	_, pem := newSigner(t)
	t.Run("update with JWK and PEM", func(t *testing.T) {
		p := newUserProject(t)
		path := "/project/providers/" + p.providerID
		before := projectState(t, p.httpClient, p.apiKey)
		newSecret := p.do(http.MethodPost, "/project/rotate-api-secret/prepare", p.apiKey, nil).str("api_secret")
		got := queueOnProjectRow(t, p.apiKey,
			func() reply {
				return p.do(http.MethodPut, path, p.apiKey, map[string]any{"jwk": "https://example.com/.well-known/jwks.json", "pem": pem, "key_type": "rsa"})
			},
			func() reply {
				return p.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", p.apiKey, nil)
			},
		)
		owner := p.withSecret(newSecret)
		t.Cleanup(func() { owner.do(http.MethodDelete, "/project", p.apiKey, nil) })
		expect(t, "secret commit", got[1], http.StatusOK, "")
		after := owner.do(http.MethodGet, path, p.apiKey, nil)
		switch got[0].Status {
		case http.StatusUnauthorized:
			if projectState(t, owner, p.apiKey) != before {
				t.Fatalf("refused update changed the provider: %v", after.Body)
			}
		case http.StatusOK:
			if after.str("pem") != pem || after.str("jwk") != "" {
				t.Fatalf("accepted update left %v, want the new PEM and no JWK", after.Body)
			}
		default:
			t.Fatalf("update: %d %v", got[0].Status, got[0].Body)
		}
	})
	t.Run("add openfort and custom", func(t *testing.T) {
		c, key, _ := newProject(t)
		newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil).str("api_secret")
		got := queueOnProjectRow(t, key,
			func() reply {
				return c.do(http.MethodPost, "/project/providers", key, map[string]any{"providers": map[string]any{
					"openfort": map[string]any{"publishable_key": "pk_" + uuid.NewString()},
					"custom":   map[string]any{"pem": pem, "key_type": "rsa"},
				}})
			},
			func() reply {
				return c.withSecret(newSecret).do(http.MethodPost, "/project/rotate-api-secret/commit", key, nil)
			},
		)
		owner := c.withSecret(newSecret)
		t.Cleanup(func() { owner.do(http.MethodDelete, "/project", key, nil) })
		expect(t, "secret commit", got[1], http.StatusOK, "")
		want := map[int]int{http.StatusUnauthorized: 0, http.StatusOK: 2}
		n, known := want[got[0].Status]
		if !known {
			t.Fatalf("add: %d %v", got[0].Status, got[0].Body)
		}
		if have := providerCount(t, owner, key); have != n {
			t.Fatalf("add answered %d and left %d providers, want %d", got[0].Status, have, n)
		}
	})
}

// Two adds of the same provider type for one project: one wins.
func TestAddProviders_Concurrent_OneWins(t *testing.T) {
	_, pem := newSigner(t)
	bodies := map[string]func() map[string]any{
		"openfort": func() map[string]any {
			return map[string]any{"providers": map[string]any{"openfort": map[string]any{"publishable_key": "pk_" + uuid.NewString()}}}
		},
		"custom": func() map[string]any { return customProvider(pem) },
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			c, key, _ := newProject(t)
			add := func() reply { return c.do(http.MethodPost, "/project/providers", key, body()) }
			got := queueOnProjectRow(t, key, add, add)
			statuses := map[int]int{got[0].Status: 1}
			statuses[got[1].Status]++
			if statuses[http.StatusOK] != 1 || statuses[http.StatusConflict] != 1 {
				t.Fatalf("concurrent adds answered %d %v and %d %v, want one 200 and one 409", got[0].Status, got[0].Body, got[1].Status, got[1].Body)
			}
			if n := providerCount(t, c, key); n != 1 {
				t.Fatalf("concurrent adds left %d providers, want 1", n)
			}
		})
	}
}

// Bodies near the cap still reach the handler intact.
func TestLargeAllowedBody_ReachesHandler(t *testing.T) {
	p := newUserProject(t)
	pad := func(payload any) io.Reader {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode body: %v", err)
		}
		return strings.NewReader(string(raw[:len(raw)-1]) + strings.Repeat(" ", 256<<10-len(raw)) + "}")
	}
	expect(t, "large encryption lookup", p.do(http.MethodPost, "/shares/encryption/reference/bulk", p.apiKey,
		pad(map[string]any{"references": []string{"ref"}})), http.StatusOK, "")
	expect(t, "large user share", p.asUser(t, http.MethodPost, "/shares", "user-"+uuid.NewString(), pad(plainShare())), http.StatusCreated, "")
}

// Wrong credentials are answered before the declared body arrives. The
// declared body is over the cap so the server does not wait to drain it.
func TestBadCredentials_AnsweredBeforeBody(t *testing.T) {
	p := newUserProject(t)
	srv := httptest.NewServer(p.handler)
	defer srv.Close()

	requests := map[string]string{
		"wrong secret": fmt.Sprintf("POST /shares/migration/import HTTP/1.1\r\nHost: shield\r\nX-API-Key: %s\r\nX-API-Secret: %s\r\n",
			p.apiKey, strings.Repeat("0", 64)),
		"bad user token": fmt.Sprintf("POST /shares HTTP/1.1\r\nHost: shield\r\nX-API-Key: %s\r\nX-Auth-Provider: custom\r\nAuthorization: Bearer bad\r\n",
			p.apiKey),
	}
	for name, head := range requests {
		t.Run(name, func(t *testing.T) {
			conn, err := net.Dial("tcp", srv.Listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, err := fmt.Fprintf(conn, "%sContent-Length: %d\r\n\r\n", head, 256<<10+1); err != nil {
				t.Fatal(err)
			}
			res, err := http.ReadResponse(bufio.NewReader(conn), nil)
			if err != nil {
				t.Fatalf("no response while the declared body is unsent: %v", err)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", res.StatusCode)
			}
		})
	}
}
