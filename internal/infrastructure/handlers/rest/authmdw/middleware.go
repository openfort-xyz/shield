package authmdw

import (
	"go.openfort.xyz/shield/internal/infrastructure/authentication"
	"go.openfort.xyz/shield/pkg/ofcontext"
	"net/http"
	"strings"
)

const TokenHeader = "Authorization"
const AuthProviderHeader = "X-Auth-Provider"
const APIKeyHeader = "X-API-Key"
const APISecretHeader = "X-API-Secret"

type Middleware struct {
	manager *authentication.Manager
}

func New(manager *authentication.Manager) *Middleware {
	return &Middleware{manager: manager}
}

func (m *Middleware) AuthenticateAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		projectID, err := m.manager.GetAPIKeyAuthenticator().Authenticate(r.Context(), apiKey)
		if err != nil {
			http.Error(w, "invalid api key", http.StatusUnauthorized)
			return
		}

		ctx := ofcontext.WithProjectID(r.Context(), projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) AuthenticateAPISecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		apiSecret := r.Header.Get(APISecretHeader)
		if apiSecret == "" {
			http.Error(w, "missing api secret", http.StatusUnauthorized)
			return
		}

		projectID, err := m.manager.GetAPISecretAuthenticator().Authenticate(r.Context(), apiKey, apiSecret)
		if err != nil {
			http.Error(w, "invalid api secret", http.StatusUnauthorized)
			return
		}

		ctx := ofcontext.WithProjectID(r.Context(), projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) AuthenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		token := r.Header.Get(TokenHeader)
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		splittedToken := strings.Split(token, " ")
		if len(splittedToken) != 2 {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		token = splittedToken[1]

		providerStr := r.Header.Get(AuthProviderHeader)
		if providerStr == "" {
			http.Error(w, "missing auth provider", http.StatusUnauthorized)
			return
		}

		provider, err := m.manager.GetAuthProvider(providerStr)
		if err != nil {
			http.Error(w, "invalid auth provider", http.StatusUnauthorized)
			return
		}

		userID, err := m.manager.GetUserAuthenticator().Authenticate(r.Context(), apiKey, token, provider)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := ofcontext.WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
