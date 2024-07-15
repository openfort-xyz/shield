// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package di

import (
	"go.openfort.xyz/shield/internal/adapters/authenticators"
	"go.openfort.xyz/shield/internal/adapters/authenticators/identity"
	"go.openfort.xyz/shield/internal/adapters/authenticators/identity/openfort_identity"
	"go.openfort.xyz/shield/internal/adapters/encryption"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest"
	"go.openfort.xyz/shield/internal/adapters/repositories/bunt"
	"go.openfort.xyz/shield/internal/adapters/repositories/bunt/encryptionpartsrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/sql"
	"go.openfort.xyz/shield/internal/adapters/repositories/sql/projectrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/sql/providerrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/sql/sharerepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/sql/userrepo"
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/applications/shamirjob"
	"go.openfort.xyz/shield/internal/applications/shareapp"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/core/services/projectsvc"
	"go.openfort.xyz/shield/internal/core/services/providersvc"
	"go.openfort.xyz/shield/internal/core/services/sharesvc"
	"go.openfort.xyz/shield/internal/core/services/usersvc"
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

func ProvideBuntDB() (*bunt.Client, error) {
	client, err := bunt.New()
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

func ProvideInMemoryEncryptionPartsRepository() (repositories.EncryptionPartsRepository, error) {
	client, err := ProvideBuntDB()
	if err != nil {
		return nil, err
	}
	encryptionPartsRepository := encryptionpartsrepo.New(client)
	return encryptionPartsRepository, nil
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

func ProvideEncryptionFactory() (factories.EncryptionFactory, error) {
	encryptionPartsRepository, err := ProvideInMemoryEncryptionPartsRepository()
	if err != nil {
		return nil, err
	}
	projectRepository, err := ProvideSQLProjectRepository()
	if err != nil {
		return nil, err
	}
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepository, projectRepository)
	return encryptionFactory, nil
}

func ProvideShareService() (services.ShareService, error) {
	shareRepository, err := ProvideSQLShareRepository()
	if err != nil {
		return nil, err
	}
	encryptionFactory, err := ProvideEncryptionFactory()
	if err != nil {
		return nil, err
	}
	shareService := sharesvc.New(shareRepository, encryptionFactory)
	return shareService, nil
}

func ProvideShamirJob() (*shamirjob.Job, error) {
	projectRepository, err := ProvideSQLProjectRepository()
	if err != nil {
		return nil, err
	}
	shareRepository, err := ProvideSQLShareRepository()
	if err != nil {
		return nil, err
	}
	job := shamirjob.New(projectRepository, shareRepository)
	return job, nil
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
	encryptionFactory, err := ProvideEncryptionFactory()
	if err != nil {
		return nil, err
	}
	job, err := ProvideShamirJob()
	if err != nil {
		return nil, err
	}
	shareApplication := shareapp.New(shareService, shareRepository, projectRepository, encryptionFactory, job)
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
	encryptionFactory, err := ProvideEncryptionFactory()
	if err != nil {
		return nil, err
	}
	encryptionPartsRepository, err := ProvideInMemoryEncryptionPartsRepository()
	if err != nil {
		return nil, err
	}
	projectApplication := projectapp.New(projectService, projectRepository, providerService, providerRepository, shareRepository, encryptionFactory, encryptionPartsRepository)
	return projectApplication, nil
}

func ProvideAuthenticationFactory() (factories.AuthenticationFactory, error) {
	projectRepository, err := ProvideSQLProjectRepository()
	if err != nil {
		return nil, err
	}
	userService, err := ProvideUserService()
	if err != nil {
		return nil, err
	}
	authenticationFactory := authenticators.NewAuthenticatorFactory(projectRepository, userService)
	return authenticationFactory, nil
}

func ProvideIdentityFactory() (factories.IdentityFactory, error) {
	config, err := ofidty.GetConfigFromEnv()
	if err != nil {
		return nil, err
	}
	providerRepository, err := ProvideSQLProviderRepository()
	if err != nil {
		return nil, err
	}
	identityFactory := identity.NewIdentityFactory(config, providerRepository)
	return identityFactory, nil
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
	authenticationFactory, err := ProvideAuthenticationFactory()
	if err != nil {
		return nil, err
	}
	identityFactory, err := ProvideIdentityFactory()
	if err != nil {
		return nil, err
	}
	userService, err := ProvideUserService()
	if err != nil {
		return nil, err
	}
	server := rest.New(config, projectApplication, shareApplication, authenticationFactory, identityFactory, userService)
	return server, nil
}
