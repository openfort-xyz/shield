package authenticationmgr

import (
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providersmgr"
	"strings"
)

type Manager struct {
	APISecretAuthenticator authentication.APISecretAuthenticator
	UserAuthenticator      authentication.UserAuthenticator
}

func NewManager(repo repositories.ProjectRepository, providerManager *providersmgr.Manager, userService services.UserService) *Manager {
	return &Manager{
		APISecretAuthenticator: newAPISecretAuthenticator(repo),
		UserAuthenticator:      newUserAuthenticator(repo, providerManager, userService),
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
