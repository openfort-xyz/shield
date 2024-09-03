package jwk

import (
	"fmt"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

func Validate(token, jwkURL string) (string, error) {
	k, err := keyfunc.NewDefault([]string{jwkURL})
	if err != nil {
		fmt.Println("Error creating Keyfunc:", err)
		return "", err
	}

	parsed, err := jwt.Parse(token, k.Keyfunc)
	if err != nil {
		fmt.Println("Error parsing token:", err)
		return "", ErrInvalidToken
	}

	claims := parsed.Claims.(jwt.MapClaims)
	return claims["sub"].(string), nil
}
