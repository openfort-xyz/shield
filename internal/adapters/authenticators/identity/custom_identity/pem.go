package cstmidty

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
)

// validMethodsForKeyType returns the JWT signing methods allowed for a given key type.
func validMethodsForKeyType(keyType provider.KeyType) []string {
	switch keyType {
	case provider.KeyTypeRSA:
		return []string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512"}
	case provider.KeyTypeECDSA:
		return []string{"ES256", "ES384", "ES512"}
	case provider.KeyTypeEd25519:
		return []string{"EdDSA"}
	default:
		return nil
	}
}

func getKeyFuncFromPEM(pem []byte, keyType provider.KeyType) (jwt.Keyfunc, []string, error) {
	var pubKey interface{}
	var err error
	// PEM parsing happens outside the keyfunc so malformed PEMs will return an error
	// (otherwise we won't realize we created/udated a provider with an invalid PEM until someone tries to use it)
	switch keyType {
	case provider.KeyTypeRSA:
		pubKey, err = jwt.ParseRSAPublicKeyFromPEM(pem)
	case provider.KeyTypeECDSA:
		pubKey, err = jwt.ParseECPublicKeyFromPEM(pem)
	case provider.KeyTypeEd25519:
		pubKey, err = jwt.ParseEdPublicKeyFromPEM(pem)
	default:
		return nil, nil, errors.ErrCertTypeNotSupported
	}
	if err != nil {
		return nil, nil, err
	}

	allowed := validMethodsForKeyType(keyType)

	keyfunc := func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || !isAllowedMethod(token.Method.Alg(), allowed) {
			return nil, errors.ErrInvalidToken
		}
		return pubKey, nil
	}

	return keyfunc, allowed, nil
}

func isAllowedMethod(alg string, allowed []string) bool {
	for _, a := range allowed {
		if a == alg {
			return true
		}
	}
	return false
}

func CheckPEM(pem []byte, keyType provider.KeyType) error {
	// Any error in trying to parse the PEM will mean the PEM is invalid
	// PEMs can be invalid either because of the format or because they are not compatible with supported key types
	_, _, err := getKeyFuncFromPEM(pem, keyType)
	return err
}
