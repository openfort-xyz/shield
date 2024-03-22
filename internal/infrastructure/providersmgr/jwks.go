package providersmgr

import (
	"context"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

func validateJWKs(ctx context.Context, token, jwkUrl string) (string, error) {
	k, err := keyfunc.NewDefault([]string{jwkUrl})
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
