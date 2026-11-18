package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/authenticators"
)

// newTestHandler builds the real router with no repositories or applications
// behind it. Requests that an auth middleware rejects never reach them.
func newTestHandler(t *testing.T, cfg *Config) http.Handler {
	t.Helper()
	return New(cfg, nil, nil, authenticators.NewAuthenticatorFactory(nil, nil), nil, nil, nil, nil).Handler()
}

func serve(h http.Handler, req *http.Request) (int, map[string]any) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestHandler_ZeroRPSDisablesRateLimit(t *testing.T) {
	t.Setenv("REQUESTS_PER_SECOND", "0")
	cfg, err := GetConfigFromEnv()
	if err != nil {
		t.Fatalf("REQUESTS_PER_SECOND=0 refused: %v", err)
	}
	h := newTestHandler(t, cfg)

	const burst = 50
	start := time.Now()
	for i := 0; i < burst; i++ {
		req := httptest.NewRequest(http.MethodGet, "/project", nil)
		req.Header.Set("X-API-Key", "key")
		if status, body := serve(h, req); status != http.StatusUnauthorized || body["message"] != "Missing API secret" {
			t.Fatalf("request %d: got %d %v, want 401 Missing API secret", i, status, body)
		}
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("%d requests took %s with REQUESTS_PER_SECOND=0, want no rate limiting", burst, elapsed)
	}
}

func TestGetConfigFromEnv_RejectsNegativeRPS(t *testing.T) {
	t.Setenv("REQUESTS_PER_SECOND", "-1")
	if _, err := GetConfigFromEnv(); err == nil {
		t.Fatal("REQUESTS_PER_SECOND=-1 accepted, want a configuration error")
	}
}
