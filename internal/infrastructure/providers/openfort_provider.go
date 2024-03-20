package providers

import (
	"context"
	"fmt"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type openfortProvider struct {
	publishableKey string
	baseURL        string
	providerID     string
	logger         *slog.Logger
}

var _ providers.IdentityProvider = (*openfortProvider)(nil)

func newOpenfortProvider(config openfortConfig, providerConfig *provider.OpenfortConfig) providers.IdentityProvider {
	return &openfortProvider{
		publishableKey: providerConfig.PublishableKey,
		providerID:     providerConfig.ProviderID,
		baseURL:        config.OpenfortBaseURL,
		logger:         slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("openfort_provider"),
	}
}

func (o *openfortProvider) GetProviderID() string {
	return o.providerID
}

func (o *openfortProvider) Identify(ctx context.Context, token string) (string, error) {
	o.logger.InfoContext(ctx, "identifying user")

	externalUserID, err := validateJWKs(ctx, token, fmt.Sprintf("%s/iam/v1/%s/jwks.json", o.baseURL, o.publishableKey))
	if err != nil {
		o.logger.ErrorContext(ctx, "failed to validate jwks", slog.String("error", err.Error()))
		return "", err
	}

	return externalUserID, nil
}
