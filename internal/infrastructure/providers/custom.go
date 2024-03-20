package providers

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type custom struct {
	jwkUrl     string
	providerID string
	logger     *slog.Logger
}

var _ providers.IdentityProvider = (*custom)(nil)

func newCustomProvider(providerConfig *provider.CustomConfig) providers.IdentityProvider {
	return &custom{
		jwkUrl:     providerConfig.JWK,
		providerID: providerConfig.ProviderID,
		logger:     slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("custom_provider"),
	}
}

func (c *custom) GetProviderID() string {
	return c.providerID
}

func (c *custom) Identify(ctx context.Context, token string) (string, error) {
	c.logger.InfoContext(ctx, "identifying user")

	externalUserID, err := validateJWKs(ctx, token, c.jwkUrl)
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to validate jwks", slog.String("error", err.Error()))
		return "", err
	}

	return externalUserID, nil
}
