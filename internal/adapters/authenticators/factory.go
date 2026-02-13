package authenticators

import (
	projauth "go.openfort.xyz/shield/internal/adapters/authenticators/project_authenticator"
	usrauth "go.openfort.xyz/shield/internal/adapters/authenticators/user_authenticator"
	"go.openfort.xyz/shield/internal/core/domain/project"
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
	return projauth.NewProjectAuthenticator(f.projectRepo, apiKey, apiSecret)
}

func (f *authenticatorFactory) CreateUserAuthenticator(proj *project.Project, token string, identityFactory factories.Identity) factories.Authenticator {
	return usrauth.NewUserAuthenticator(f.userService, proj, token, identityFactory)
}
