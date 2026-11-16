package authmdw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
)

func serve(h http.Handler, headers map[string]string) int {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestAuthenticateServiceToken(t *testing.T) {
	token := strings.Repeat("t", 32)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	tests := []struct {
		name       string
		configured string
		presented  string
		want       int
	}{
		{"matching token", token, token, http.StatusNoContent},
		{"missing header", token, "", http.StatusUnauthorized},
		{"wrong token", token, strings.Repeat("u", 32), http.StatusUnauthorized},
		{"token prefix", token, token[:31], http.StatusUnauthorized},
		{"unset token refuses even an empty header", "", "", http.StatusUnauthorized},
		{"short configured token refuses its own value", token[:31], token[:31], http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serve(AuthenticateServiceToken(tt.configured)(ok), map[string]string{ServiceTokenHeader: tt.presented})
			if got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

type projectByKey struct {
	services.ProjectService
	proj *project.Project
}

func (p projectByKey) GetByAPIKey(context.Context, string) (*project.Project, error) {
	return p.proj, nil
}

func TestAuthenticateUser_DisabledProject(t *testing.T) {
	now := time.Now()
	reached := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mdw := New(nil, nil, nil, projectByKey{proj: &project.Project{ID: "p", DisabledAt: &now}})

	got := serve(mdw.AuthenticateUser(reached), map[string]string{APIKeyHeader: "key", AuthProviderHeader: AuthenticationTypeCustom})
	if got != http.StatusForbidden {
		t.Errorf("status = %d, want 403", got)
	}
}
