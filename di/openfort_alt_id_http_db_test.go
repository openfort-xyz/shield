package di

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

// fakeOpenfortAPI answers Shield's third-party token check the way the Openfort
// API does: token "<name>" belongs to player pla_<name> and user usr_<name>.
func fakeOpenfortAPI(t *testing.T, suffix string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/iam/v1/oauth/third_party" {
			http.NotFound(w, r)
			return
		}
		var body struct{ Token string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id": "pla_" + body.Token + suffix, "userId": "usr_" + body.Token + suffix,
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OPENFORT_BASE_URL", srv.URL)
	// Shield refuses OTP requests without a notification service; the
	// skip-verification requests these tests make never send anything.
	t.Setenv("RESEND_API_KEY", "re_unused")
	t.Setenv("RESEND_FROM_EMAIL", "otp@example.com")
	t.Setenv("RESEND_EMAIL_SENDER_NAME", "Shield")
	t.Setenv("SMS_API_KEY", "unused")
}

func TestEncryptionSession_OpenfortSecondUserID_HTTP(t *testing.T) {
	suffix := "-" + uuid.NewString()
	fakeOpenfortAPI(t, suffix)
	c, apiKey, part := newProject(t)
	expect(t, "add openfort provider", c.do(http.MethodPost, "/project/providers", apiKey,
		map[string]any{"providers": map[string]any{"openfort": map[string]any{"publishable_key": "pk_test_" + uuid.NewString()}}}),
		http.StatusOK, "")
	asPlayer := func(method, path, token string, body any, headers ...string) reply {
		return c.do(method, path, "", body, append([]string{
			"X-API-Key", apiKey, "X-Auth-Provider", "openfort", "Authorization", "Bearer " + token,
			"X-Openfort-Provider", "firebase", "X-Openfort-Token-Type", "idToken",
		}, headers...)...)
	}

	expect(t, "register alice's share", asPlayer(http.MethodPost, "/shares", "alice",
		map[string]any{"secret": "alice-secret", "entropy": "project", "encryption_part": part}), http.StatusCreated, "")
	var linked []string
	err := dbtest.RequireDB(t).Raw(`SELECT eu.external_user_id FROM shld_external_users eu
		JOIN shld_users u ON u.id = eu.user_id JOIN shld_projects p ON p.id = u.project_id
		WHERE p.api_key = ?`, apiKey).Scan(&linked).Error
	if err != nil {
		t.Fatalf("read external users: %v", err)
	}
	if len(linked) != 1 || linked[0] != "pla_alice"+suffix {
		t.Fatalf("alice's share is linked to %v, want only her player ID pla_alice%s", linked, suffix)
	}
	expect(t, "register bob's share", asPlayer(http.MethodPost, "/shares", "bob",
		map[string]any{"secret": "bob-secret", "entropy": "project", "encryption_part": part}), http.StatusCreated, "")
	expect(t, "enable 2FA", c.do(http.MethodPost, "/project/enable-2fa", apiKey, nil), http.StatusOK, "")
	alicesUserID := "usr_alice" + suffix
	expect(t, "unverified OTP for alice's user ID", c.do(http.MethodPost, "/project/otp", apiKey,
		map[string]any{"user_id": alicesUserID, "dangerously_skip_verification": true}), http.StatusOK, "")
	minted := c.do(http.MethodPost, "/project/encryption-session", apiKey,
		map[string]any{"encryption_part": part, "user_id": alicesUserID})
	expect(t, "session for alice's user ID", minted, http.StatusOK, "")
	session := minted.str("session_id")
	if session == "" {
		t.Fatalf("no session_id in %v", minted.Body)
	}

	expect(t, "bob presenting alice's session",
		asPlayer(http.MethodGet, "/shares", "bob", nil, "X-Encryption-Session", session),
		http.StatusBadRequest, "EC_INVALID")
	expect(t, "alice presenting the session minted for her user ID",
		asPlayer(http.MethodGet, "/shares", "alice", nil, "X-Encryption-Session", session),
		http.StatusPreconditionRequired, "OTP_MISSING")
}
