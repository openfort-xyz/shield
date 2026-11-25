package di

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestShareRegistration_RejectsDuplicateReferences_HTTP(t *testing.T) {
	p := newUserProject(t)
	for _, reference := range []string{"", "signer-fixture"} {
		user := "owner-" + uuid.NewString()
		original := map[string]any{"secret": "original", "entropy": "project", "encryption_part": p.part}
		ref := reference
		if ref == "" {
			ref = "default"
		} else {
			original["reference"] = ref
		}
		preregister := func(payload map[string]any) reply {
			return p.do(http.MethodPost, "/admin/preregister", p.apiKey, payload, "X-User-ID", user, "X-Auth-Provider", "custom")
		}
		expect(t, "initial preregistration", preregister(original), http.StatusCreated, "")
		expect(t, "identical preregistration retry", preregister(original), http.StatusConflict, "SH_EXISTS")
		changed := map[string]any{"secret": "other", "entropy": "project", "encryption_part": p.part, "reference": ref}
		expect(t, "registration of the same reference", p.asUser(t, http.MethodPost, "/shares", user, changed), http.StatusConflict, "SH_EXISTS")
		expect(t, "another user may register the reference",
			p.asUser(t, http.MethodPost, "/shares", "other-"+uuid.NewString(), changed), http.StatusCreated, "")
		got := p.asUser(t, http.MethodGet, "/shares/"+ref, user, nil, "X-Encryption-Part", p.part)
		expect(t, "read original share", got, http.StatusOK, "")
		if got.str("secret") != "original" {
			t.Fatal("duplicate registration changed the original share")
		}
	}
}
