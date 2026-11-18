package authmdw

import (
	"context"
	"net/http"
	"testing"

	"github.com/openfort-xyz/shield/internal/core/domain/user"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/contexter"
)

type customIdentity struct{ factories.Identity }

func (customIdentity) GetProviderID() string { return "provider" }

type identities struct{ factories.IdentityFactory }

func (identities) CreateCustomIdentity(context.Context, string) (factories.Identity, error) {
	return customIdentity{}, nil
}

type users struct{ services.UserService }

func (users) GetOrCreate(_ context.Context, projectID, _, _ string) (*user.User, error) {
	return &user.User{ID: "internal-user", ProjectID: projectID}, nil
}

// Encryption sessions are bound to the external user ID, which X-User-ID
// carries, so handlers behind PreRegisterUser must see it.
func TestPreRegisterUser_ExposesExternalUserID(t *testing.T) {
	var internalID, externalID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internalID = contexter.GetUserID(r.Context())
		externalID = contexter.GetExternalUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	mdw := New(nil, identities{}, users{}, nil)

	got := serve(mdw.PreRegisterUser(next), map[string]string{UserIDHeader: "ext-user", AuthProviderHeader: AuthenticationTypeCustom})
	if got != http.StatusOK {
		t.Fatalf("status = %d, want 200", got)
	}
	if internalID != "internal-user" || externalID != "ext-user" {
		t.Errorf("context user IDs = (%q, %q), want (%q, %q)", internalID, externalID, "internal-user", "ext-user")
	}
}
