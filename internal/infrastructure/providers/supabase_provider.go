package providers

import (
	"context"
	"github.com/supabase-community/gotrue-go"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/providers"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type supabaseProvider struct {
	client     gotrue.Client
	providerID string
	logger     *slog.Logger
}

var _ providers.IdentityProvider = (*supabaseProvider)(nil)

func newSupabaseProvider(config supabaseConfig, providerConfig *provider.Supabase) providers.IdentityProvider {
	client := gotrue.New(providerConfig.SupabaseProjectReference, config.SupabaseAPIKey)
	if config.SupabaseBaseURL != "" {
		client = client.WithCustomGoTrueURL(config.SupabaseBaseURL)
	}
	return &supabaseProvider{
		client:     client,
		providerID: providerConfig.ProviderID,
		logger:     slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("supabase_provider"),
	}
}

func (s *supabaseProvider) GetProviderID() string {
	return s.providerID
}

func (s *supabaseProvider) Identify(ctx context.Context, token string) (string, error) {
	s.logger.InfoContext(ctx, "identifying user")

	authedClient := s.client.WithToken(token)
	externalUser, err := authedClient.GetUser()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user", slog.String("error", err.Error()))
		return "", err
	}

	return externalUser.ID.String(), nil
}
