package cstmidty

import (
	"context"
	"log/slog"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"
	"github.com/openfort-xyz/shield/pkg/jwk"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/openfort-xyz/shield/internal/core/domain/provider"
	"github.com/openfort-xyz/shield/pkg/logger"
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
		externalUserID, err = jwk.Validate(token, []string{c.config.JWK})
	default:
		return "", domainErrors.ErrProviderMisconfigured
	}
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to validate jwt", logger.Error(err))
		return "", err
	}

	return externalUserID, nil
}

func (c *CustomIdentityFactory) GetCookieFieldName() string {
	if c.config.CookieFieldName == nil {
		return ""
	}
	return *c.config.CookieFieldName
}

func (c *CustomIdentityFactory) validatePEM(token string) (string, error) {
	keyFunc, validMethods, err := getKeyFuncFromPEM([]byte(c.config.PEM), c.config.KeyType)

	if err != nil {
		c.logger.ErrorContext(context.Background(), "failed to parse PEM file", logger.Error(err))
		return "", err
	}

	parsed, err := jwt.Parse(token, keyFunc, jwt.WithValidMethods(validMethods))
	if err != nil {
		return "", err
	}
	claims := parsed.Claims.(jwt.MapClaims)
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", domainErrors.ErrInvalidToken
	}
	return sub, nil
}
