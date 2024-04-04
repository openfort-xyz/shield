package services

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type ProviderService interface {
	Configure(ctx context.Context, prov *provider.Provider) error
}
