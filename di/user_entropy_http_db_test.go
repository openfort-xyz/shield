package di

import (
	"net/http"
	"testing"
)

func TestUserEntropyChecksDigestSpecificParametersOnWrite(t *testing.T) {
	p := newUserProject(t)
	for _, tc := range []struct {
		name       string
		digest     string
		iterations int
		length     int
		status     int
	}{
		{"argon2", "argon2", 12, 32, http.StatusCreated},
		{"sha256 legacy", "sha256", 1000, 256, http.StatusCreated},
		{"sha512 legacy", "sha512", 2000, 512, http.StatusCreated},
		{"negative iterations", "argon2", -1, 32, http.StatusBadRequest},
		{"negative length", "argon2", 12, -32, http.StatusBadRequest},
		{"unsupported digest", "unsupported", 12, 32, http.StatusBadRequest},
		{"SHA digest with Argon2 units", "sha256", 12, 32, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{
				"secret": "opaque-ciphertext", "entropy": "user", "reference": tc.name,
				"salt": "fixture-salt", "digest": tc.digest, "iterations": tc.iterations, "length": tc.length,
			}
			got := p.asUser(t, http.MethodPost, "/shares", "user-entropy", payload)
			expect(t, "register recovery share", got, tc.status, "")

			if tc.status == http.StatusBadRequest {
				payload["reference"] = "argon2"
				expect(t, "reject invalid change to the existing share",
					p.asUser(t, http.MethodPut, "/shares", "user-entropy", payload), http.StatusBadRequest, "")
			}
		})
	}
	got := p.asUser(t, http.MethodGet, "/shares/argon2", "user-entropy", nil)
	expect(t, "read original recovery share", got, http.StatusOK, "")
	if got.str("digest") != "argon2" || got.Body["iterations"] != float64(12) {
		t.Fatalf("unexpected saved parameters: digest %v iterations %v", got.Body["digest"], got.Body["iterations"])
	}
}
