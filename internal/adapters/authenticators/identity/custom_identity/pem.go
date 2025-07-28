package cstmidty

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

func getKeyFuncFromPEM(pem []byte, keyType provider.KeyType) (jwt.Keyfunc, error) {
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
		return nil, errors.ErrCertTypeNotSupported
	}
	if err != nil {
		return nil, err
	}

	return func(*jwt.Token) (interface{}, error) {
		return pubKey, nil
	}, nil
}

func CheckPEM(pem []byte, keyType provider.KeyType) error {
	// Any error in trying to parse the PEM will mean the PEM is invalid
	// PEMs can be invalid either because of the format or because they are not compatible with supported key types
	_, err := getKeyFuncFromPEM(pem, keyType)
	return err
}
