package deadlinemdw

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	metrics "github.com/openfort-xyz/metrics"
)

const (
	writeTimeout = 200 * time.Millisecond
	handlerDelay = 3 * writeTimeout
)

// newServer mirrors the production chain: the deadline middleware runs before
// metrics.HTTPMiddleware, whose ResponseWriter wrapper has no Unwrap.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	slow := func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(handlerDelay)
		w.WriteHeader(http.StatusNoContent)
	}

	r := mux.NewRouter()
	r.Use(ExtendWriteDeadline(map[string]time.Duration{"slow-extended": 10 * handlerDelay}))
	r.Use(metrics.HTTPMiddleware)
	r.HandleFunc("/extended", slow).Name("slow-extended")
	r.HandleFunc("/default", slow).Name("slow-default")

	srv := httptest.NewUnstartedServer(r)
	srv.Config.WriteTimeout = writeTimeout
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func TestExtendWriteDeadline_NamedRouteOutlivesWriteTimeout(t *testing.T) {
	srv := newServer(t)

	resp, err := srv.Client().Get(srv.URL + "/extended")
	if err != nil {
		t.Fatalf("expected a response past WriteTimeout, got error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestExtendWriteDeadline_OtherRoutesKeepWriteTimeout(t *testing.T) {
	srv := newServer(t)

	resp, err := srv.Client().Get(srv.URL + "/default")
	if err == nil {
		resp.Body.Close()
		t.Fatalf("expected the connection to be dropped after WriteTimeout, got status %d", resp.StatusCode)
	}
}
