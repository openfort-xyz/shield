//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/applications/userapp"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/core/services/projectsvc"
	"go.openfort.xyz/shield/internal/core/services/providersvc"
	"go.openfort.xyz/shield/internal/core/services/sharesvc"
	"go.openfort.xyz/shield/internal/core/services/usersvc"
	"go.openfort.xyz/shield/internal/infrastructure/providers"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/projectrepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/providerrepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/sharerepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql/userrepo"
)

func ProvideSQL() (c *sql.Client, err error) {
	wire.Build(
		sql.New,
		sql.GetConfigFromEnv,
	)

	return
}

func ProvideSQLUserRepository() (r repositories.UserRepository, err error) {
	wire.Build(
		userrepo.New,
		ProvideSQL,
	)

	return
}

func ProvideSQLProjectRepository() (r repositories.ProjectRepository, err error) {
	wire.Build(
		projectrepo.New,
		ProvideSQL,
	)

	return
}

func ProvideSQLProviderRepository() (r repositories.ProviderRepository, err error) {
	wire.Build(
		providerrepo.New,
		ProvideSQL,
	)

	return
}

func ProvideSQLShareRepository() (r repositories.ShareRepository, err error) {
	wire.Build(
		sharerepo.New,
		ProvideSQL,
	)

	return
}

func ProvideProjectService() (s services.ProjectService, err error) {
	wire.Build(
		projectsvc.New,
		ProvideSQLProjectRepository,
	)

	return
}

func ProvideProviderService() (s services.ProviderService, err error) {
	wire.Build(
		providersvc.New,
		ProvideSQLProviderRepository,
	)

	return
}

func ProvideUserService() (s services.UserService, err error) {
	wire.Build(
		usersvc.New,
		ProvideSQLUserRepository,
	)

	return
}

func ProvideShareService() (s services.ShareService, err error) {
	wire.Build(
		sharesvc.New,
		ProvideSQLShareRepository,
	)

	return
}

func ProvideProviderManager() (pm *providers.Manager, err error) {
	wire.Build(
		providers.NewManager,
		providers.GetConfigFromEnv,
		ProvideSQLProviderRepository,
	)

	return
}

func ProvideUserApplication() (a *userapp.UserApplication, err error) {
	wire.Build(
		userapp.New,
		ProvideUserService,
		ProvideShareService,
		ProvideProjectService,
		ProvideProviderService,
		ProvideProviderManager,
	)

	return
}

func ProvideProjectApplication() (a *projectapp.ProjectApplication, err error) {
	wire.Build(
		projectapp.New,
		ProvideProjectService,
		ProvideProviderService,
	)

	return
}
