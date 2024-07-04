package repositories

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type ProviderRepository interface {
	Create(ctx context.Context, prov *provider.Provider) error
	Get(ctx context.Context, id string) (*provider.Provider, error)
	GetByProjectAndType(ctx context.Context, projectID string, providerType provider.Type) (*provider.Provider, error)
	GetByAPIKeyAndType(ctx context.Context, apiKey string, providerType provider.Type) (*provider.Provider, error)
	List(ctx context.Context, projectID string) ([]*provider.Provider, error)
	Delete(ctx context.Context, providerID string) error

	CreateCustom(ctx context.Context, provider *provider.CustomConfig) error
	GetCustom(ctx context.Context, providerID string) (*provider.CustomConfig, error)
	UpdateCustom(ctx context.Context, provider *provider.CustomConfig) error

	CreateOpenfort(ctx context.Context, provider *provider.OpenfortConfig) error
	GetOpenfort(ctx context.Context, providerID string) (*provider.OpenfortConfig, error)
	UpdateOpenfort(ctx context.Context, provider *provider.OpenfortConfig) error
}
