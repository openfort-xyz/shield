package repositories

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

var (
	ErrProviderNotFound = errors.New("custom authentication not found")
)

type ProviderRepository interface {
	Create(ctx context.Context, prov *provider.Provider) error
	GetByProjectAndType(ctx context.Context, projectID string, providerType provider.Type) (*provider.Provider, error)
	List(ctx context.Context, projectID string) ([]*provider.Provider, error)
	Delete(ctx context.Context, providerID string) error

	CreateCustom(ctx context.Context, provider *provider.Custom) error
	GetCustom(ctx context.Context, providerID string) (*provider.Custom, error)

	CreateOpenfort(ctx context.Context, provider *provider.Openfort) error
	GetOpenfort(ctx context.Context, providerID string) (*provider.Openfort, error)

	CreateSupabase(ctx context.Context, provider *provider.Supabase) error
	GetSupabase(ctx context.Context, providerID string) (*provider.Supabase, error)
}
