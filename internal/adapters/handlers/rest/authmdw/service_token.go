package authmdw

import (
	"crypto/subtle"
	"net/http"

	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/api"
)

const ServiceTokenHeader = "X-Openfort-Service-Token" //nolint:gosec

// minServiceTokenLength guards against a short or placeholder token being
// configured; with one, every service-token request is refused.
const minServiceTokenLength = 32

// AuthenticateServiceToken admits requests whose X-Openfort-Service-Token
// equals token. It refuses every request when token is unset or shorter
// than 32 characters.
func AuthenticateServiceToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			presented := r.Header.Get(ServiceTokenHeader)
			if len(token) < minServiceTokenLength || subtle.ConstantTimeCompare([]byte(presented), []byte(token)) != 1 {
				api.RespondWithError(w, api.ErrInvalidServiceToken)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
