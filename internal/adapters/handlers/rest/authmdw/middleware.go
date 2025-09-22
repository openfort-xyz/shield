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

func getTokenFromHeader(header string) (string, error) {
	if header == "" {
		return "", api.ErrMissingToken
	}

	splittedToken := strings.Split(header, " ")

	// TODO: Are we supporting other token types than Bearer? e.g. Basic, Digest, etc.
	if len(splittedToken) != 2 {
		return "", api.ErrInvalidToken
	}

	return splittedToken[1], nil
}

// This is a bit weird, but there's no standard name for the cookie field
// RFC 6265 specifies at most the name of the cookie header, but nothing for this particular field
func getTokenFromCookie(r *http.Request, cookieFieldName string) (string, error) {
	cookie, err := r.Cookie(cookieFieldName)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", api.ErrMissingToken
		}
		return "", api.ErrInvalidToken
	}

	token := cookie.Value
	if token == "" {
		return "", api.ErrMissingToken
	}

	return token, nil
}

func (m *Middleware) AuthenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			api.RespondWithError(w, api.ErrMissingAPIKey)
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

		// Determine the token source based on the identity type
		var token string
		if identity.GetCookieFieldName() == "" {
			// Also default path for Openfort identity (which does not use cookies)
			token, err = getTokenFromHeader(r.Header.Get(TokenHeader))
		} else {
			// Cookie vs header ARE mutually exclusive, otherwise it's not clear which one we should obey
			if r.Header.Get(TokenHeader) != "" {
				api.RespondWithError(w, api.ErrInvalidToken)
				return
			}
			token, err = getTokenFromCookie(r, identity.GetCookieFieldName())
			// We could potentially recover from the previous error and fall back to the header,
			// but again it would be a bit weird to have both a cookie and a header for the same identity type.
			// So we just return an error if the cookie is not present
		}

		if err != nil {
			api.RespondWithError(w, api.ErrInvalidToken)
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
		ctx = contexter.WithExternalUserID(ctx, authentication.ExternalUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
