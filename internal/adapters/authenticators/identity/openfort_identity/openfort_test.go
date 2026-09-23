package ofidty

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/core/domain/provider"
)

const sessionCookie = "__Secure-openfort.session_token=abc.def; other=1"

func identify(t *testing.T, sessionCookie, token string, status int) (string, error, http.Header) {
	t.Helper()

	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		if r.URL.Path != "/iam/v2/auth/get-session" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"session":{"expiresAt":%q},"user":{"id":"usr_1"}}`, time.Now().Add(time.Hour).Format(time.RFC3339))
	}))
	t.Cleanup(srv.Close)

	identity := NewOpenfortIdentityFactory(
		&Config{OpenfortBaseURL: srv.URL},
		&provider.OpenfortConfig{PublishableKey: "pk_test_1"},
		nil, nil, sessionCookie,
	)
	userID, err := identity.Identify(context.Background(), token)
	return userID, err, got
}

func TestIdentifyForwardsSessionCookie(t *testing.T) {
	userID, err, got := identify(t, sessionCookie, "", http.StatusOK)
	if err != nil || userID != "usr_1" {
		t.Fatalf("got (%q, %v), want (usr_1, nil)", userID, err)
	}
	if got.Get("Cookie") != sessionCookie {
		t.Errorf("Cookie = %q, want the raw header forwarded", got.Get("Cookie"))
	}
	if got.Get("Authorization") != "" {
		t.Errorf("Authorization = %q, want none on a cookie request", got.Get("Authorization"))
	}
	if got.Get("x-project-key") != "pk_test_1" {
		t.Errorf("x-project-key = %q", got.Get("x-project-key"))
	}
}

func TestIdentifyBearerUnchanged(t *testing.T) {
	userID, err, got := identify(t, "", "opaque.token", http.StatusOK)
	if err != nil || userID != "usr_1" {
		t.Fatalf("got (%q, %v), want (usr_1, nil)", userID, err)
	}
	if got.Get("Authorization") != "Bearer opaque.token" || got.Get("Cookie") != "" {
		t.Errorf("Authorization = %q, Cookie = %q", got.Get("Authorization"), got.Get("Cookie"))
	}
}

func TestIdentifyRejectsInvalidSessionCookie(t *testing.T) {
	if _, err, _ := identify(t, sessionCookie, "", http.StatusUnauthorized); err == nil {
		t.Fatal("want an error when the API rejects the cookie")
	}
}
