package services

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/provider"
)

type ProviderService interface {
	Configure(ctx context.Context, prov *provider.Provider) error
}
