package di

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/openfort-xyz/shield/pkg/logger/logtest"
)

// Neither the old nor the new API secret may reach the logs on any path of a
// secret rotation, including rejected and failed ones, at debug level.
func TestAPISecretRotation_SecretsNeverLogged(t *testing.T) {
	rec := logtest.Start(slog.LevelDebug)
	defer rec.Stop()

	c, key, _ := newProject(t)
	oldSecret := c.secret
	const commit = "/project/rotate-api-secret/commit"

	lost := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil).str("api_secret")
	newSecret := c.do(http.MethodPost, "/project/rotate-api-secret/prepare", key, nil).str("api_secret")
	c.withSecret(lost).do(http.MethodPost, commit, key, nil)
	c.do(http.MethodPost, commit, key, nil)
	c.withSecret(newSecret).do(http.MethodGet, "/project", key, nil)
	expect(t, "commit", c.withSecret(newSecret).do(http.MethodPost, commit, key, nil), http.StatusOK, "")
	c.do(http.MethodGet, "/project", key, nil)
	c.withSecret(newSecret).do(http.MethodPost, commit, key, nil)
	c.withSecret(newSecret).do(http.MethodDelete, "/project", key, nil)

	records := rec.Records()
	if len(records) == 0 {
		t.Fatal("no log records captured")
	}
	out, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	for name, secret := range map[string]string{"old": oldSecret, "replaced pending": lost, "new": newSecret} {
		if secret == "" || strings.Contains(string(out), secret) {
			t.Errorf("%s secret %q found in logs (or empty)", name, secret)
		}
	}
}
