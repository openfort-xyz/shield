package jwk

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func TestCheckTenantURL(t *testing.T) {
	t.Setenv(AllowedHostsEnv, "authservice:3000, idp.internal")

	tests := []struct {
		url     string
		allowed bool
	}{
		{"https://auth.example.com/.well-known/jwks.json", true},
		{"http://auth.example.com/.well-known/jwks.json", false},
		{"https://169.254.169.254/latest/meta-data", false},
		{"https://10.0.0.5/jwks.json", false},
		{"https://[::1]/jwks.json", false},
		{"https://[fd00::1]/jwks.json", false},
		{"https://user:pass@auth.example.com/jwks.json", false},
		{"file:///etc/passwd", false},
		{"gopher://auth.example.com/", false},
		{"/relative/jwks.json", false},
		{"http://authservice:3000/.well-known/jwks.json", true},
		{"http://authservice:3001/.well-known/jwks.json", false},
		{"http://idp.internal/jwks.json", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := CheckTenantURL(tt.url)
			if tt.allowed && err != nil {
				t.Errorf("CheckTenantURL rejected an allowed URL: %v", err)
			}
			if !tt.allowed && !errors.Is(err, ErrDisallowedJWKURL) {
				t.Errorf("CheckTenantURL = %v, want %v", err, ErrDisallowedJWKURL)
			}
		})
	}
}

func TestIsPublic(t *testing.T) {
	tests := map[string]bool{
		"8.8.8.8":            true,
		"2606:4700::1111":    true,
		"127.0.0.1":          false,
		"10.1.2.3":           false,
		"172.16.0.1":         false,
		"192.168.1.1":        false,
		"169.254.169.254":    false,
		"100.64.0.1":         false,
		"0.1.2.3":            false,
		"::1":                false,
		"fd00::1":            false,
		"fe80::1":            false,
		"::ffff:127.0.0.1":   false,
		"64:ff9b::a00:1":     false,
		"2002:a00:1::1":      false,
		"224.0.0.1":          false,
		"255.255.255.255":    false,
		"198.18.0.1":         false,
		"::ffff:8.8.8.8":     true,
		"192.0.0.8":          false,
		"240.0.0.1":          false,
		"0.0.0.0":            false,
		"ff02::1":            false,
		"64:ff9b:1::a00:1":   false,
		"2001:4860::8888":    true,
		"100.127.255.255":    false,
		"100.128.0.1":        true,
		"172.32.0.1":         true,
		"169.255.0.1":        true,
		"11.0.0.1":           true,
		"::ffff:10.0.0.1":    false,
		"::ffff:169.254.1.1": false,
	}
	for ip, want := range tests {
		if got := isPublic(netip.MustParseAddr(ip)); got != want {
			t.Errorf("isPublic(%s) = %v, want %v", ip, got, want)
		}
	}
}

// A JWK server on loopback reached through a hostname passes the URL check
// (it is not a literal address) and must be refused before any connection is
// made: the dial resolves the name itself and refuses non-public addresses.
func TestValidateTenant_RefusesLoopbackBehindHostname(t *testing.T) {
	var conns atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	// Listen where "localhost" resolves first (often ::1), the address a
	// dial without the check would reach.
	ips, err := net.DefaultResolver.LookupNetIP(context.Background(), "ip", "localhost")
	if err != nil || len(ips) == 0 {
		t.Fatalf("resolve localhost: %v", err)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(ips[0].String(), "0"))
	if err != nil {
		t.Fatalf("listen on %s: %v", ips[0], err)
	}
	server.Listener = listener
	server.StartTLS()
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	jwkURL := "https://localhost:" + u.Port() + "/jwks.json"
	_, key := startJWKS(t)
	token := signES256(t, key, jwt.MapClaims{"sub": "user-123", "exp": time.Now().Add(time.Hour).Unix()})

	// keyfunc swallows a failed first fetch, so the refusal surfaces as a
	// rejected token; what matters is that no connection reached the address.
	if _, err := ValidateTenant(token, jwkURL); err == nil {
		t.Errorf("ValidateTenant(%s) accepted a token", jwkURL)
	}
	if conns.Load() != 0 {
		t.Errorf("Shield opened %d connections to a loopback address", conns.Load())
	}
}

func TestValidateTenant_AllowlistedInternalHost(t *testing.T) {
	server, key := startJWKS(t)
	u, _ := url.Parse(server.URL)
	t.Setenv(AllowedHostsEnv, u.Host)
	token := signES256(t, key, jwt.MapClaims{"sub": "user-123", "exp": time.Now().Add(time.Hour).Unix()})

	sub, err := ValidateTenant(token, server.URL+"/jwks.json")

	if err != nil || sub != "user-123" {
		t.Errorf("ValidateTenant on an allowlisted host = %q, %v; want user-123", sub, err)
	}
}

// An allowlisted host must not be usable to bounce the fetch to another
// internal address.
func TestValidateTenant_RefusesRedirectToInternalAddress(t *testing.T) {
	var internalHits atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { internalHits.Add(1) }))
	t.Cleanup(internal.Close)
	bouncer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL+"/jwks.json", http.StatusFound)
	}))
	t.Cleanup(bouncer.Close)
	u, _ := url.Parse(bouncer.URL)
	t.Setenv(AllowedHostsEnv, u.Host)

	_, _ = ValidateTenant("a.b.c", bouncer.URL+"/jwks.json")

	if internalHits.Load() != 0 {
		t.Errorf("the internal server received %d requests", internalHits.Load())
	}
}

func TestValidateTenant_CapsRedirects(t *testing.T) {
	var hits atomic.Int32
	loop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, fmt.Sprintf("/hop/%d", hits.Load()), http.StatusFound)
	}))
	t.Cleanup(loop.Close)
	u, _ := url.Parse(loop.URL)
	t.Setenv(AllowedHostsEnv, u.Host)

	_, _ = ValidateTenant("a.b.c", loop.URL+"/jwks.json")

	if hits.Load() > maxRedirects+1 {
		t.Errorf("followed %d requests, want at most %d", hits.Load(), maxRedirects+1)
	}
}

// Validation reuses one fetched key set per URL instead of fetching it on
// every request.
func TestValidate_FetchesKeySetOnce(t *testing.T) {
	var hits atomic.Int32
	server, key := startJWKS(t)
	counting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		server.Config.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(counting.Close)
	token := signES256(t, key, jwt.MapClaims{"sub": "user-123", "exp": time.Now().Add(time.Hour).Unix()})

	for i := 0; i < 5; i++ {
		if _, err := Validate(token, []string{counting.URL}); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	}

	if hits.Load() != 1 {
		t.Errorf("key set fetched %d times for 5 validations, want 1", hits.Load())
	}
}

// A slow JWK endpoint must only delay validations against that endpoint, not
// every other tenant's.
func TestValidate_SlowKeySetDoesNotBlockOthers(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(slow.Close)
	t.Cleanup(func() { close(release) })
	fast, key := startJWKS(t)
	token := signES256(t, key, jwt.MapClaims{"sub": "user-123", "exp": time.Now().Add(time.Hour).Unix()})

	go func() { _, _ = Validate("a.b.c", []string{slow.URL}) }()
	time.Sleep(100 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := Validate(token, []string{fast.URL})
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Validate against the fast key set: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("validation against a fast key set waited on a slow, unrelated one")
	}
}
