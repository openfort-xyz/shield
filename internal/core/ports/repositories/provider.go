package repositories

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

var (
	ErrCustomProviderNotFound   = errors.New("custom authentication not found")
	ErrOpenfortProviderNotFound = errors.New("openfort not found")
	ErrSupabaseProviderNotFound = errors.New("supabase not found")
)

type ProviderRepository interface {
	List(ctx context.Context, projectID string) ([]*provider.Provider, error)

	CreateCustom(ctx context.Context, provider *provider.Custom) error
	GetCustom(ctx context.Context, projectID string) (*provider.Custom, error)

	CreateOpenfort(ctx context.Context, provider *provider.Openfort) error
	GetOpenfort(ctx context.Context, projectID string) (*provider.Openfort, error)

	CreateSupabase(ctx context.Context, provider *provider.Supabase) error
	GetSupabase(ctx context.Context, projectID string) (*provider.Supabase, error)
}
