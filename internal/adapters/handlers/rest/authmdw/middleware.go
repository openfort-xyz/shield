package authmdw

import (
	"net/http"
	"strings"

	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/internal/core/ports/services"

	"go.openfort.xyz/shield/internal/adapters/handlers/rest/api"
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
const EncryptionSessionHeader = "X-Encryption-Session"               //nolint:gosec
const UserIDHeader = "X-User-ID"                                     //nolint:gosec
const AuthenticationTypeCustom = "custom"                            //nolint:gosec
const AuthenticationTypeOpenfort = "openfort"                        //nolint:gosec
const RequestIDHeader = "X-Request-ID"                               //nolint:gosec

type Middleware struct {
	authenticationFactory factories.AuthenticationFactory
	identityFactory       factories.IdentityFactory
	userService           services.UserService
}

func New(authenticationFactory factories.AuthenticationFactory, identityFactory factories.IdentityFactory, userService services.UserService) *Middleware {
	return &Middleware{
		authenticationFactory: authenticationFactory,
		identityFactory:       identityFactory,
		userService:           userService,
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

		authenticator := m.authenticationFactory.CreateProjectAuthenticator(apiKey, apiSecret)
		authentication, err := authenticator.Authenticate(r.Context())
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidAPICredentials)
			return
		}

		ctx := contexter.WithProjectID(r.Context(), authentication.ProjectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) PreRegisterUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			api.RespondWithError(w, api.ErrMissingAPIKey)
			return
		}

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

		var identity factories.Identity
		var err error
		switch providerStr {
		case AuthenticationTypeCustom:
			identity, err = m.identityFactory.CreateCustomIdentity(r.Context(), apiKey)
		case AuthenticationTypeOpenfort:
			identity, err = m.identityFactory.CreateOpenfortIdentity(r.Context(), apiKey, nil, nil)
		default:
			api.RespondWithError(w, api.ErrInvalidAuthProvider)
			return
		}
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidAuthProvider)
			return
		}

		usr, err := m.userService.GetOrCreate(r.Context(), contexter.GetProjectID(r.Context()), userID, identity.GetProviderID())
		if err != nil {
			api.RespondWithError(w, api.ErrPreRegisterUser)
			return
		}

		ctx := contexter.WithUserID(r.Context(), usr.ID)
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

		var identity factories.Identity
		var err error

		switch providerStr {
		case AuthenticationTypeCustom:
			identity, err = m.identityFactory.CreateCustomIdentity(r.Context(), apiKey)
		case AuthenticationTypeOpenfort:
			var openfortProvider *string
			if r.Header.Get(OpenfortProviderHeader) != "" {
				openfortProvider = new(string)
				*openfortProvider = r.Header.Get(OpenfortProviderHeader)
			}
			var openfortTokenType *string
			if r.Header.Get(OpenfortTokenTypeHeader) != "" {
				openfortTokenType = new(string)
				*openfortTokenType = r.Header.Get(OpenfortTokenTypeHeader)
			}
			identity, err = m.identityFactory.CreateOpenfortIdentity(r.Context(), apiKey, openfortProvider, openfortTokenType)
		default:
			api.RespondWithError(w, api.ErrInvalidAuthProvider)
			return
		}
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidAuthProvider)
			return
		}

		authenticator := m.authenticationFactory.CreateUserAuthenticator(apiKey, token, identity)
		authentication, err := authenticator.Authenticate(r.Context())
		if err != nil {
			api.RespondWithError(w, api.ErrInvalidToken)
			return
		}

		ctx := contexter.WithUserID(r.Context(), authentication.UserID)
		ctx = contexter.WithProjectID(ctx, authentication.ProjectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
