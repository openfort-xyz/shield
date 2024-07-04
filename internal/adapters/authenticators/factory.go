package authenticators

import (
	"go.openfort.xyz/shield/internal/adapters/authenticators/project_authenticator"
	"go.openfort.xyz/shield/internal/adapters/authenticators/user_authenticator"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
)

type authenticatorFactory struct {
	projectRepo repositories.ProjectRepository
	userService services.UserService
}

func NewAuthenticatorFactory(projectRepo repositories.ProjectRepository, userService services.UserService) factories.AuthenticationFactory {
	return &authenticatorFactory{
		projectRepo: projectRepo,
		userService: userService,
	}
}

func (f *authenticatorFactory) CreateProjectAuthenticator(apiKey, apiSecret string) factories.Authenticator {
	return project_authenticator.NewProjectAuthenticator(f.projectRepo, apiKey, apiSecret)
}

func (f *authenticatorFactory) CreateUserAuthenticator(apiKey, token string, identityFactory factories.Identity) factories.Authenticator {
	return user_authenticator.NewUserAuthenticator(f.projectRepo, f.userService, apiKey, token, identityFactory)
}

// func (m *Manager) PreRegisterUser(ctx context.Context, userID string, providerType provider.Type) (string, error) {
//	projID := contexter.GetProjectID(ctx)
//	prov, err := m.providerManager.GetProvider(ctx, projID, providerType)
//	if err != nil {
//		m.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
//		return "", err
//	}
//
//	usr, err := m.userService.GetByExternal(ctx, userID, prov.GetProviderID())
//	if err != nil {
//		if !errors.Is(err, domainErrors.ErrUserNotFound) && !errors.Is(err, domainErrors.ErrExternalUserNotFound) {
//			m.logger.ErrorContext(ctx, "failed to get user by external", logger.Error(err))
//			return "", err
//		}
//		usr, err = m.userService.Create(ctx, projID)
//		if err != nil {
//			m.logger.ErrorContext(ctx, "failed to create user", logger.Error(err))
//			return "", err
//		}
//
//		_, err = m.userService.CreateExternal(ctx, projID, usr.ID, userID, prov.GetProviderID())
//		if err != nil {
//			m.logger.ErrorContext(ctx, "failed to create external user", logger.Error(err))
//			return "", err
//		}
//	}
//
//	return usr.ID, nil
//}
