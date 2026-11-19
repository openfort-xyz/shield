package services

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
)

type ProviderService interface {
	// Configure stores the providers and their configs, all or none, once
	// check passes under the project row lock.
	Configure(ctx context.Context, projectID string, provs []*provider.Provider, check project.CheckAPISecretFunc) error
}
