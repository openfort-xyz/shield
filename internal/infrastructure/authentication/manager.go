package authentication

import (
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/authentication"
	"strings"
)

type Manager struct {
	APIKeyAuthenticator    authentication.APIKeyAuthenticator
	APISecretAuthenticator authentication.APISecretAuthenticator
	UserAuthenticator      authentication.UserAuthenticator
}

func NewManager(apiKeyAuthenticator authentication.APIKeyAuthenticator, apiSecretAuthenticator authentication.APISecretAuthenticator, userAuthenticator authentication.UserAuthenticator) *Manager {
	return &Manager{
		APIKeyAuthenticator:    apiKeyAuthenticator,
		APISecretAuthenticator: apiSecretAuthenticator,
		UserAuthenticator:      userAuthenticator,
	}
}

func (m *Manager) GetAPIKeyAuthenticator() authentication.APIKeyAuthenticator {
	return m.APIKeyAuthenticator
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
	case "supabase":
		return provider.TypeSupabase, nil
	case "custom":
		return provider.TypeCustom, nil
	default:
		return provider.TypeUnknown, domain.ErrUnknownProviderType
	}
}
