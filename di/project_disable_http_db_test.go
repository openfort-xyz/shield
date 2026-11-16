package di

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

const (
	testServiceToken = "0123456789abcdef0123456789abcdef"
	disabledMessage  = "This project has been disabled. If you believe this is a mistake, contact founders@openfort.io."
)

func (c httpClient) setDisabled(action, apiKey string) reply {
	c.t.Helper()
	return c.do(http.MethodPost, "/internal/projects/"+action, "", map[string]any{"api_key": apiKey},
		"X-Openfort-Service-Token", testServiceToken)
}

func expectDisabled(t *testing.T, step string, got reply) {
	t.Helper()
	expect(t, step, got, http.StatusForbidden, "PJ_DISABLED")
	if got.str("message") != disabledMessage {
		t.Fatalf("%s: message = %q, want %q", step, got.str("message"), disabledMessage)
	}
}

func disabledAt(t *testing.T, apiKey string) *time.Time {
	t.Helper()
	var at sql.NullTime
	if err := dbtest.RequireDB(t).Raw(`SELECT disabled_at FROM shld_projects WHERE api_key = ?`, apiKey).Row().Scan(&at); err != nil {
		t.Fatalf("read disabled_at: %v", err)
	}
	if !at.Valid {
		return nil
	}
	return &at.Time
}

// userRoute calls a user-JWT route. With no provider registered it fails
// after the project lookup, so anything but 403 means the project got past
// the disabled check.
func (c httpClient) userRoute(apiKey string) reply {
	c.t.Helper()
	return c.do(http.MethodGet, "/shares", "", nil, "X-API-Key", apiKey, "X-Auth-Provider", "custom")
}

func TestProjectDisable_HTTP(t *testing.T) {
	t.Setenv("SHIELD_SERVICE_TOKEN", testServiceToken)
	c, key, _ := newProject(t)

	if primed := c.userRoute(key); primed.Status == http.StatusForbidden {
		t.Fatalf("user route refused before disable: %v", primed.Body)
	}

	expect(t, "disable", c.setDisabled("disable", key), http.StatusNoContent, "")
	expectDisabled(t, "key+secret route", c.do(http.MethodGet, "/project", key, nil))
	expectDisabled(t, "share encryption route", c.do(http.MethodPost, "/shares/encryption/reference/bulk", key, map[string]any{}))
	expectDisabled(t, "user route (cached project evicted)", c.userRoute(key))
	expectDisabled(t, "api key rotation commit", c.do(http.MethodPost, "/project/rotate-api-key/commit", key, nil))

	first := disabledAt(t, key)
	if first == nil {
		t.Fatal("disabled_at not set")
	}
	expect(t, "disable again", c.setDisabled("disable", key), http.StatusNoContent, "")
	if again := disabledAt(t, key); again == nil || !again.Equal(*first) {
		t.Fatalf("disabled_at after second disable = %v, want first value %v", again, first)
	}

	expect(t, "enable", c.setDisabled("enable", key), http.StatusNoContent, "")
	if at := disabledAt(t, key); at != nil {
		t.Fatalf("disabled_at after enable = %v, want NULL", at)
	}
	expect(t, "key+secret route after enable", c.do(http.MethodGet, "/project", key, nil), http.StatusOK, "")
	if got := c.userRoute(key); got.Status == http.StatusForbidden {
		t.Fatalf("user route after enable: %v", got.Body)
	}
	expect(t, "enable again", c.setDisabled("enable", key), http.StatusNoContent, "")
}

func TestProjectDisable_HTTP_PendingRotationSurvives(t *testing.T) {
	t.Setenv("SHIELD_SERVICE_TOKEN", testServiceToken)
	c, key, _ := newProject(t)

	pending := c.do(http.MethodPost, "/project/rotate-api-key/prepare", key, nil).str("api_key")
	expect(t, "disable", c.setDisabled("disable", key), http.StatusNoContent, "")
	expectDisabled(t, "commit with the pending key", c.do(http.MethodPost, "/project/rotate-api-key/commit", pending, nil))

	expect(t, "enable", c.setDisabled("enable", key), http.StatusNoContent, "")
	expect(t, "commit after enable", c.do(http.MethodPost, "/project/rotate-api-key/commit", pending, nil), http.StatusOK, "")
	expect(t, "pending key is current", c.do(http.MethodGet, "/project", pending, nil), http.StatusOK, "")
}

func TestProjectDisable_HTTP_DeleteStillWorks(t *testing.T) {
	t.Setenv("SHIELD_SERVICE_TOKEN", testServiceToken)
	c, key, _ := newProject(t)

	expect(t, "disable", c.setDisabled("disable", key), http.StatusNoContent, "")
	expect(t, "delete disabled project", c.do(http.MethodDelete, "/project", key, nil), http.StatusNoContent, "")
	expect(t, "project is gone", c.setDisabled("enable", key), http.StatusNotFound, "PJ_NOT_FOUND")
}

func TestProjectDisable_HTTP_Errors(t *testing.T) {
	t.Setenv("SHIELD_SERVICE_TOKEN", testServiceToken)
	c, key, _ := newProject(t)

	expect(t, "unknown api key", c.setDisabled("disable", uuid.NewString()), http.StatusNotFound, "PJ_NOT_FOUND")
	expect(t, "missing api key", c.do(http.MethodPost, "/internal/projects/disable", "", map[string]any{},
		"X-Openfort-Service-Token", testServiceToken), http.StatusBadRequest, "")
	expect(t, "wrong token", c.do(http.MethodPost, "/internal/projects/disable", "", map[string]any{"api_key": key},
		"X-Openfort-Service-Token", strings.Repeat("x", 32)), http.StatusUnauthorized, "A_INVALID")
	expect(t, "no token", c.do(http.MethodPost, "/internal/projects/disable", "", map[string]any{"api_key": key}),
		http.StatusUnauthorized, "A_INVALID")
	expect(t, "project secret is not a service token", c.do(http.MethodPost, "/internal/projects/disable", key, map[string]any{"api_key": key}),
		http.StatusUnauthorized, "A_INVALID")
	expect(t, "project untouched", c.do(http.MethodGet, "/project", key, nil), http.StatusOK, "")
}
