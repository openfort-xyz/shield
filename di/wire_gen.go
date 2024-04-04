// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package di

import (
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/applications/shareapp"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/core/services/projectsvc"
	"go.openfort.xyz/shield/internal/core/services/providersvc"
	"go.openfort.xyz/shield/internal/core/services/sharesvc"
	"go.openfort.xyz/shield/internal/core/services/usersvc"
	"go.openfort.xyz/shield/internal/infrastructure/authenticationmgr"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest"
	"go.openfort.xyz/shield/internal/infrastructure/providersmgr"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/projectrepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/providerrepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/sharerepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/userrepo"
)

// Injectors from wire.go:

func ProvideSQL() (*sql.Client, error) {
	config, err := sql.GetConfigFromEnv()
	if err != nil {
		return nil, err
	}
	client, err := sql.New(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func ProvideSQLUserRepository() (repositories.UserRepository, error) {
	client, err := ProvideSQL()
	if err != nil {
		return nil, err
	}
	userRepository := userrepo.New(client)
	return userRepository, nil
}

func ProvideSQLProjectRepository() (repositories.ProjectRepository, error) {
	client, err := ProvideSQL()
	if err != nil {
		return nil, err
	}
	projectRepository := projectrepo.New(client)
	return projectRepository, nil
}

func ProvideSQLProviderRepository() (repositories.ProviderRepository, error) {
	client, err := ProvideSQL()
	if err != nil {
		return nil, err
	}
	providerRepository := providerrepo.New(client)
	return providerRepository, nil
}

func ProvideSQLShareRepository() (repositories.ShareRepository, error) {
	client, err := ProvideSQL()
	if err != nil {
		return nil, err
	}
	shareRepository := sharerepo.New(client)
	return shareRepository, nil
}

func ProvideProjectService() (services.ProjectService, error) {
	projectRepository, err := ProvideSQLProjectRepository()
	if err != nil {
		return nil, err
	}
	projectService := projectsvc.New(projectRepository)
	return projectService, nil
}

func ProvideProviderService() (services.ProviderService, error) {
	providerRepository, err := ProvideSQLProviderRepository()
	if err != nil {
		return nil, err
	}
	providerService := providersvc.New(providerRepository)
	return providerService, nil
}

func ProvideUserService() (services.UserService, error) {
	userRepository, err := ProvideSQLUserRepository()
	if err != nil {
		return nil, err
	}
	userService := usersvc.New(userRepository)
	return userService, nil
}

func ProvideShareService() (services.ShareService, error) {
	shareRepository, err := ProvideSQLShareRepository()
	if err != nil {
		return nil, err
	}
	shareService := sharesvc.New(shareRepository)
	return shareService, nil
}

func ProvideProviderManager() (*providersmgr.Manager, error) {
	config, err := providersmgr.GetConfigFromEnv()
	if err != nil {
		return nil, err
	}
	providerRepository, err := ProvideSQLProviderRepository()
	if err != nil {
		return nil, err
	}
	manager := providersmgr.NewManager(config, providerRepository)
	return manager, nil
}

func ProvideShareApplication() (*shareapp.ShareApplication, error) {
	shareService, err := ProvideShareService()
	if err != nil {
		return nil, err
	}
	shareRepository, err := ProvideSQLShareRepository()
	if err != nil {
		return nil, err
	}
	projectRepository, err := ProvideSQLProjectRepository()
	if err != nil {
		return nil, err
	}
	shareApplication := shareapp.New(shareService, shareRepository, projectRepository)
	return shareApplication, nil
}

func ProvideProjectApplication() (*projectapp.ProjectApplication, error) {
	projectService, err := ProvideProjectService()
	if err != nil {
		return nil, err
	}
	projectRepository, err := ProvideSQLProjectRepository()
	if err != nil {
		return nil, err
	}
	providerService, err := ProvideProviderService()
	if err != nil {
		return nil, err
	}
	providerRepository, err := ProvideSQLProviderRepository()
	if err != nil {
		return nil, err
	}
	shareRepository, err := ProvideSQLShareRepository()
	if err != nil {
		return nil, err
	}
	projectApplication := projectapp.New(projectService, projectRepository, providerService, providerRepository, shareRepository)
	return projectApplication, nil
}

func ProvideAuthenticationManager() (*authenticationmgr.Manager, error) {
	projectRepository, err := ProvideSQLProjectRepository()
	if err != nil {
		return nil, err
	}
	manager, err := ProvideProviderManager()
	if err != nil {
		return nil, err
	}
	userService, err := ProvideUserService()
	if err != nil {
		return nil, err
	}
	authenticationmgrManager := authenticationmgr.NewManager(projectRepository, manager, userService)
	return authenticationmgrManager, nil
}

func ProvideRESTServer() (*rest.Server, error) {
	config, err := rest.GetConfigFromEnv()
	if err != nil {
		return nil, err
	}
	projectApplication, err := ProvideProjectApplication()
	if err != nil {
		return nil, err
	}
	shareApplication, err := ProvideShareApplication()
	if err != nil {
		return nil, err
	}
	manager, err := ProvideAuthenticationManager()
	if err != nil {
		return nil, err
	}
	server := rest.New(config, projectApplication, shareApplication, manager)
	return server, nil
}
