package providersmgr

import (
	"context"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/pkg/logger"
)

type custom struct {
	jwkURL     string
	providerID string
	logger     *slog.Logger
}

var _ providers.IdentityProvider = (*custom)(nil)

func newCustomProvider(providerConfig *provider.CustomConfig) providers.IdentityProvider {
	return &custom{
		jwkURL:     providerConfig.JWK,
		providerID: providerConfig.ProviderID,
		logger:     logger.New("custom_provider"),
	}
}

func (c *custom) GetProviderID() string {
	return c.providerID
}

func (c *custom) Identify(ctx context.Context, token string, _ ...providers.CustomOption) (string, error) {
	c.logger.InfoContext(ctx, "identifying user")

	externalUserID, err := validateJWKs(token, c.jwkURL)
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to validate jwks", logger.Error(err))
		return "", err
	}

	return externalUserID, nil
}
