package jwk

import (
	"encoding/base64"
	"strings"

	keyfunc "github.com/MicahParks/keyfunc/v3"
	jwt "github.com/golang-jwt/jwt/v5"
)

func Validate(token string, jwkURLs []string) (string, error) {
	k, err := keyfunc.NewDefault(jwkURLs)
	if err != nil {
		return "", err
	}

	parsed, err := jwt.Parse(token, k.Keyfunc)
	if err != nil {
		return "", ErrInvalidToken
	}

	claims := parsed.Claims.(jwt.MapClaims)
	return claims["sub"].(string), nil
}

// IsJWT checks if the provided string is a valid JWT token format.
// Returns true if the string is a JWT, false if it's an arbitrary access token.
func IsJWT(token string) bool {
	// JWT tokens have exactly 3 parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	// Each part should be base64url encoded
	// Check if each part can be decoded (basic validation)
	for _, part := range parts {
		if part == "" {
			return false
		}

		// Try to decode each part as base64url
		// JWT uses base64url encoding (RawURLEncoding)
		_, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			// Try with padding in case it's standard base64
			_, err = base64.URLEncoding.DecodeString(part)
			if err != nil {
				return false
			}
		}
	}

	return true
}
