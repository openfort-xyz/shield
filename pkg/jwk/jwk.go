package jwk

import (
	keyfunc "github.com/MicahParks/keyfunc/v3"
	jwt "github.com/golang-jwt/jwt/v5"
)

func Validate(token, jwkURL string) (string, error) {
	k, err := keyfunc.NewDefault([]string{jwkURL})
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
