package authenticationmgr

import (
	"context"
	"strings"

	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providersmgr"
)

type Manager struct {
	APISecretAuthenticator authentication.APISecretAuthenticator
	UserAuthenticator      authentication.UserAuthenticator
	repo                   repositories.ProjectRepository
	mapOrigins             map[string][]string
}

func NewManager(repo repositories.ProjectRepository, providerManager *providersmgr.Manager, userService services.UserService) *Manager {
	return &Manager{
		repo:                   repo,
		APISecretAuthenticator: newAPISecretAuthenticator(repo),
		UserAuthenticator:      newUserAuthenticator(repo, providerManager, userService),
		mapOrigins:             make(map[string][]string),
	}
}

func (m *Manager) GetAPISecretAuthenticator() authentication.APISecretAuthenticator {
	return m.APISecretAuthenticator
}

func (m *Manager) GetUserAuthenticator() authentication.UserAuthenticator {
	return m.UserAuthenticator
}

func (m *Manager) GetAuthProvider(providerStr string) (provider.Type, error) {
	switch strings.ToLower(providerStr) {
	case "openfort":
		return provider.TypeOpenfort, nil
	case "custom":
		return provider.TypeCustom, nil
	default:
		return provider.TypeUnknown, domain.ErrUnknownProviderType
	}
}

func (m *Manager) IsAllowedOrigin(ctx context.Context, apiKey string, origin string) (bool, error) {
	if cachedOrigins, cached := m.mapOrigins[apiKey]; cached {
		for _, o := range cachedOrigins {
			if o == origin {
				return true, nil
			}
		}
	}

	dbOrigins, err := m.repo.GetAllowedOriginsByAPIKey(ctx, apiKey)
	if err != nil {

		return false, err
	}
	m.mapOrigins[apiKey] = dbOrigins

	for _, o := range dbOrigins {
		if o == origin {
			return true, nil
		}
	}

	return false, nil
}
