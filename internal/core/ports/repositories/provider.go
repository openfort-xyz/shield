package repositories

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
)

type ProviderRepository interface {
	// Create, Delete and the Update methods write only after check passes
	// under a lock on the project row, in the same transaction; they return
	// its error, or ErrProjectNotFound or ErrProjectDisabled. Create stores
	// the providers together with their configs, all or none, and returns
	// ErrProviderAlreadyExists when the project has one of their types.
	Create(ctx context.Context, projectID string, provs []*provider.Provider, check project.CheckAPISecretFunc) error
	Get(ctx context.Context, id string) (*provider.Provider, error)
	GetByProjectAndType(ctx context.Context, projectID string, providerType provider.Type) (*provider.Provider, error)
	List(ctx context.Context, projectID string) ([]*provider.Provider, error)
	Delete(ctx context.Context, projectID, providerID string, check project.CheckAPISecretFunc) error

	GetCustom(ctx context.Context, providerID string) (*provider.CustomConfig, error)
	UpdateCustom(ctx context.Context, projectID string, provider *provider.CustomConfig, check project.CheckAPISecretFunc) error

	GetOpenfort(ctx context.Context, providerID string) (*provider.OpenfortConfig, error)
	UpdateOpenfort(ctx context.Context, projectID string, provider *provider.OpenfortConfig, check project.CheckAPISecretFunc) error
}
