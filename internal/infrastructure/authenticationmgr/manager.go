package authenticationmgr

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/logger"

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
	providerManager        *providersmgr.Manager
	userService            services.UserService
	mapOrigins             map[string][]string
	logger                 *slog.Logger
}

func NewManager(repo repositories.ProjectRepository, providerManager *providersmgr.Manager, userService services.UserService) *Manager {
	return &Manager{
		repo:                   repo,
		APISecretAuthenticator: newAPISecretAuthenticator(repo),
		providerManager:        providerManager,
		UserAuthenticator:      newUserAuthenticator(repo, providerManager, userService),
		userService:            userService,
		mapOrigins:             make(map[string][]string),
		logger:                 logger.New("authentication_manager"),
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

func (m *Manager) PreRegisterUser(ctx context.Context, userID string, providerType provider.Type) (string, error) {
	projID := contexter.GetProjectID(ctx)
	prov, err := m.providerManager.GetProvider(ctx, projID, providerType)
	if err != nil {
		m.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return "", err
	}

	usr, err := m.userService.GetByExternal(ctx, userID, prov.GetProviderID())
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) && !errors.Is(err, domain.ErrExternalUserNotFound) {
			m.logger.ErrorContext(ctx, "failed to get user by external", logger.Error(err))
			return "", err
		}
		usr, err = m.userService.Create(ctx, projID)
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to create user", logger.Error(err))
			return "", err
		}

		_, err = m.userService.CreateExternal(ctx, projID, usr.ID, userID, prov.GetProviderID())
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to create external user", logger.Error(err))
			return "", err
		}
	}

	return usr.ID, nil
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
