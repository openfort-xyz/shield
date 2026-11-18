package rest

import (
	"bytes"
	"encoding/json"
	"io"
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

func TestHandler_LimitsRequestBodySize(t *testing.T) {
	const limit = 256 << 10
	h := newTestHandler(t, &Config{RPS: 0})

	tests := []struct {
		name        string
		path        string
		size        int
		hideLength  bool
		wantStatus  int
		wantMessage string
	}{
		{"public route over the limit", "/register", limit + 1, false, http.StatusRequestEntityTooLarge, ""},
		{"public route over the limit, no Content-Length", "/register", limit + 1, true, http.StatusRequestEntityTooLarge, ""},
		{"authenticated route over the limit", "/shares/encryption/user/bulk", limit + 1, false, http.StatusRequestEntityTooLarge, ""},
		{"body at the limit reaches the handler", "/register", limit, false, http.StatusBadRequest, "failed to parse request body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader = bytes.NewReader(bytes.Repeat([]byte("a"), tt.size))
			if tt.hideLength {
				body = io.MultiReader(body)
			}
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			status, got := serve(h, req)
			if status != tt.wantStatus {
				t.Fatalf("got %d %v, want %d", status, got, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusRequestEntityTooLarge && (got["code"] != "BODY_TOO_LARGE" || got["message"] == "") {
				t.Fatalf("body = %v, want code BODY_TOO_LARGE and a message", got)
			}
			if tt.wantMessage != "" && got["message"] != tt.wantMessage {
				t.Fatalf("message = %v, want %q", got["message"], tt.wantMessage)
			}
		})
	}
}

func TestGetConfigFromEnv_RejectsNegativeRPS(t *testing.T) {
	t.Setenv("REQUESTS_PER_SECOND", "-1")
	if _, err := GetConfigFromEnv(); err == nil {
		t.Fatal("REQUESTS_PER_SECOND=-1 accepted, want a configuration error")
	}
}
