package authmdw

import (
	"net/http"
	"strings"

	authenticate "go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/infrastructure/authenticationmgr"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"
	"go.openfort.xyz/shield/pkg/contexter"
)

const TokenHeader = "Authorization"                                  //nolint:gosec
const AuthProviderHeader = "X-Auth-Provider"                         //nolint:gosec
const APIKeyHeader = "X-API-Key"                                     //nolint:gosec
const APISecretHeader = "X-API-Secret"                               //nolint:gosec
const OpenfortProviderHeader = "X-Openfort-Provider"                 //nolint:gosec
const OpenfortTokenTypeHeader = "X-Openfort-Token-Type"              //nolint:gosec
const AccessControlAllowOriginHeader = "Access-Control-Allow-Origin" //nolint:gosec
const EncryptionPartHeader = "X-Encryption-Part"                     //nolint:gosec
const UserIDHeader = "X-User-ID"                                     //nolint:gosec

type Middleware struct {
	manager *authenticationmgr.Manager
}

func New(manager *authenticationmgr.Manager) *Middleware {
	return &Middleware{
		manager: manager,
	}
}

func (m *Middleware) AuthenticateAPISecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			api.RespondWithError(w, api.ErrMissingAPIKey)
			return
		}

		apiSecret := r.Header.Get(APISecretHeader)
		if apiSecret == "" {
			api.RespondWithError(w, api.ErrMissingAPISecret)
			return
		}

		projectID, err := m.manager.GetAPISecretAuthenticator().Authenticate(r.Context(), apiKey, apiSecret)
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidAPISecret)
			return
		}

		ctx := contexter.WithProjectID(r.Context(), projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) PreRegisterUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(UserIDHeader)
		if userID == "" {
			api.RespondWithError(w, api.ErrMissingUserID)
			return
		}

		providerStr := r.Header.Get(AuthProviderHeader)
		if providerStr == "" {
			api.RespondWithError(w, api.ErrMissingAuthProvider)
			return
		}

		provider, err := m.manager.GetAuthProvider(providerStr)
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidAuthProvider)
			return
		}

		usr, err := m.manager.PreRegisterUser(r.Context(), userID, provider)
		if err != nil {
			api.RespondWithError(w, api.ErrPreRegisterUser)
			return
		}

		ctx := contexter.WithUserID(r.Context(), usr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) AuthenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			api.RespondWithError(w, api.ErrMissingAPIKey)
			return
		}

		token := r.Header.Get(TokenHeader)
		if token == "" {
			api.RespondWithError(w, api.ErrMissingToken)
			return
		}

		splittedToken := strings.Split(token, " ")
		if len(splittedToken) != 2 {
			api.RespondWithError(w, api.ErrInvalidToken)
			return
		}

		token = splittedToken[1]

		providerStr := r.Header.Get(AuthProviderHeader)
		if providerStr == "" {
			api.RespondWithError(w, api.ErrMissingAuthProvider)
			return
		}

		openfortProvider := r.Header.Get(OpenfortProviderHeader)
		openfortTokenType := r.Header.Get(OpenfortTokenTypeHeader)

		var customOptions []authenticate.CustomOption
		if openfortProvider != "" && openfortTokenType != "" {
			customOptions = append(customOptions, authenticate.WithOpenfortProvider(openfortProvider))
			customOptions = append(customOptions, authenticate.WithOpenfortTokenType(openfortTokenType))
		}

		provider, err := m.manager.GetAuthProvider(providerStr)
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidAuthProvider)
			return
		}

		auth, err := m.manager.GetUserAuthenticator().Authenticate(r.Context(), apiKey, token, provider, customOptions...)
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidToken)
			return
		}

		ctx := contexter.WithUserID(r.Context(), auth.UserID)
		ctx = contexter.WithProjectID(ctx, auth.ProjectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
