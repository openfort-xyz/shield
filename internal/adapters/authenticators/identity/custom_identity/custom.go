package cstmidty

import (
	"context"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/pkg/jwk"

	"github.com/golang-jwt/jwt/v5"

	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/pkg/logger"
)

type CustomIdentityFactory struct {
	config *provider.CustomConfig
	logger *slog.Logger
}

var _ factories.Identity = (*CustomIdentityFactory)(nil)

func NewCustomIdentityFactory(providerConfig *provider.CustomConfig) factories.Identity {
	return &CustomIdentityFactory{
		config: providerConfig,
		logger: logger.New("custom_provider"),
	}
}

func (c *CustomIdentityFactory) GetProviderID() string {
	return c.config.ProviderID
}

func (c *CustomIdentityFactory) Identify(ctx context.Context, token string) (string, error) {
	c.logger.InfoContext(ctx, "identifying user")

	var externalUserID string
	var err error
	switch {
	case c.config.PEM != "" && c.config.KeyType != provider.KeyTypeUnknown:
		externalUserID, err = c.validatePEM(token)
	case c.config.JWK != "":
		externalUserID, err = jwk.Validate(token, c.config.JWK)
	default:
		return "", errors.ErrProviderMisconfigured
	}
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to validate jwt", logger.Error(err))
		return "", err
	}

	return externalUserID, nil
}

func (c *CustomIdentityFactory) validatePEM(token string) (string, error) {
	var keyFunc jwt.Keyfunc
	switch c.config.KeyType {
	case provider.KeyTypeRSA:
		keyFunc = func(*jwt.Token) (interface{}, error) {
			return jwt.ParseRSAPublicKeyFromPEM([]byte(c.config.PEM))
		}
	case provider.KeyTypeECDSA:
		keyFunc = func(*jwt.Token) (interface{}, error) {
			return jwt.ParseECPublicKeyFromPEM([]byte(c.config.PEM))
		}
	case provider.KeyTypeEd25519:
		keyFunc = func(*jwt.Token) (interface{}, error) {
			return jwt.ParseEdPublicKeyFromPEM([]byte(c.config.PEM))
		}
	default:
		return "", errors.ErrCertTypeNotSupported
	}

	parsed, err := jwt.Parse(token, keyFunc)
	if err != nil {
		return "", err
	}

	claims := parsed.Claims.(jwt.MapClaims)
	return claims["sub"].(string), nil
}
