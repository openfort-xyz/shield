package providersmgr

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/pkg/logger"
)

type custom struct {
	config *provider.CustomConfig
	logger *slog.Logger
}

var _ providers.IdentityProvider = (*custom)(nil)

func newCustomProvider(providerConfig *provider.CustomConfig) providers.IdentityProvider {
	return &custom{
		config: providerConfig,
		logger: logger.New("custom_provider"),
	}
}

func (c *custom) GetProviderID() string {
	return c.config.ProviderID
}

func (c *custom) Identify(ctx context.Context, token string, _ ...providers.CustomOption) (string, error) {
	c.logger.InfoContext(ctx, "identifying user")

	var externalUserID string
	var err error
	switch {
	case c.config.PEM != "" && c.config.KeyType != provider.KeyTypeUnknown:
		externalUserID, err = c.validatePEM(token)
	case c.config.JWK != "":
		externalUserID, err = validateJWKs(token, c.config.JWK)
	default:
		return "", ErrProviderMisconfigured
	}
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to validate jwt", logger.Error(err))
		return "", err
	}

	return externalUserID, nil
}

func (c *custom) validatePEM(token string) (string, error) {
	var keyFunc jwt.Keyfunc
	switch c.config.KeyType {
	case provider.KeyTypeRSA:
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			return jwt.ParseRSAPublicKeyFromPEM([]byte(c.config.PEM))
		}
	case provider.KeyTypeECDSA:
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			return jwt.ParseECPublicKeyFromPEM([]byte(c.config.PEM))
		}
	case provider.KeyTypeEd25519:
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			return jwt.ParseEdPublicKeyFromPEM([]byte(c.config.PEM))
		}
	default:
		return "", ErrCertTypeNotSupported
	}

	parsed, err := jwt.Parse(token, keyFunc)
	if err != nil {
		return "", err
	}

	claims := parsed.Claims.(jwt.MapClaims)
	return claims["sub"].(string), nil
}
