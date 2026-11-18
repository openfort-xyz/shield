package rest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/iotest"
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

	// At the default 100 requests per second, 300 requests take at least
	// 2.8 s even with the limiter's slack of 10.
	const burst = 300
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

const bodyLimit = 256 << 10

func TestHandler_LimitsRequestBodySize(t *testing.T) {
	const limit = bodyLimit
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
		{"authenticated route over the limit, no credentials", "/shares/encryption/user/bulk", limit + 1, false, http.StatusUnauthorized, "Missing API key"},
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

// A caller that declares a body and never sends it must still get its
// authentication failure, not wait out the read timeout.
func TestHandler_AuthenticatesBeforeReadingBody(t *testing.T) {
	srv := httptest.NewServer(newTestHandler(t, &Config{RPS: 0}))
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(conn, "POST /shares/encryption/user/bulk HTTP/1.1\r\nHost: shield\r\nContent-Length: %d\r\n\r\n", bodyLimit+1); err != nil {
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
}

func TestHandler_LimitsChunkedBody(t *testing.T) {
	h := newTestHandler(t, &Config{RPS: 0})
	var transferEncoding []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		transferEncoding = r.TransferEncoding
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()

	body := io.MultiReader(bytes.NewReader(bytes.Repeat([]byte("a"), bodyLimit+1)))
	res, err := http.Post(srv.URL+"/register", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	got := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&got)
	if res.StatusCode != http.StatusRequestEntityTooLarge || got["code"] != "BODY_TOO_LARGE" {
		t.Fatalf("got %d %v, want 413 BODY_TOO_LARGE", res.StatusCode, got)
	}
	if len(transferEncoding) != 1 || transferEncoding[0] != "chunked" {
		t.Fatalf("request Transfer-Encoding = %v, want chunked", transferEncoding)
	}
}

func TestHandler_BodyReadErrorIsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/register", iotest.ErrReader(errors.New("connection reset")))
	status, body := serve(newTestHandler(t, &Config{RPS: 0}), req)
	if status != http.StatusBadRequest || body["message"] != "failed to read request body" {
		t.Fatalf("got %d %v, want 400 failed to read request body", status, body)
	}
}

func TestGetConfigFromEnv_RejectsNegativeRPS(t *testing.T) {
	t.Setenv("REQUESTS_PER_SECOND", "-1")
	if _, err := GetConfigFromEnv(); err == nil {
		t.Fatal("REQUESTS_PER_SECOND=-1 accepted, want a configuration error")
	}
}
