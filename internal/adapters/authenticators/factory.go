package authenticators

import (
	projauth "github.com/openfort-xyz/shield/internal/adapters/authenticators/project_authenticator"
	usrauth "github.com/openfort-xyz/shield/internal/adapters/authenticators/user_authenticator"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
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
