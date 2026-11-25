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
		{"WebCrypto SHA-256 bits", "SHA-256", 1000, 256, http.StatusCreated},
		{"WebCrypto SHA-512 bits", "SHA-512", 1000, 512, http.StatusCreated},
		{"SHA-256 bytes", "SHA-256", 1000, 32, http.StatusCreated},
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

func TestUserEntropyLegacyMetadataSurvivesUpdateAndMigration(t *testing.T) {
	for _, digest := range []string{"SHA-256", "SHA-512"} {
		t.Run(digest, func(t *testing.T) {
			p := newUserProject(t)
			payload := map[string]any{
				"secret": "opaque-ciphertext", "entropy": "user", "reference": "original",
				"salt": "fixture-salt", "digest": digest, "iterations": 1000, "length": 256,
			}
			expect(t, "create legacy share", p.asUser(t, http.MethodPost, "/shares", "owner", payload), http.StatusCreated, "")
			expect(t, "update legacy share", p.asUser(t, http.MethodPut, "/shares", "owner", payload), http.StatusOK, "")
			exported := p.do(http.MethodGet, "/shares/migration/export/original", p.apiKey, nil)
			expect(t, "export legacy share", exported, http.StatusOK, "")
			destination := p.do(http.MethodPost, "/user", p.apiKey,
				map[string]any{"external_user_id": "import-owner", "provider_id": p.providerID})
			expect(t, "create destination user", destination, http.StatusCreated, "")
			exported.Body["user_id"] = destination.str("user_id")
			exported.Body["reference"] = "imported"
			expect(t, "import legacy share", p.do(http.MethodPost, "/shares/migration/import", p.apiKey, exported.Body), http.StatusCreated, "")
			restored := p.do(http.MethodGet, "/shares/migration/export/imported", p.apiKey, nil)
			expect(t, "read imported share", restored, http.StatusOK, "")
			for _, key := range []string{"secret", "salt", "digest", "iterations", "length"} {
				if restored.Body[key] != exported.Body[key] {
					t.Fatalf("migration changed %s: %v != %v", key, restored.Body[key], exported.Body[key])
				}
			}
		})
	}
}
