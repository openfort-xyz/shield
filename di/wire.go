//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
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

func ProvideProviderManager() (pm *providersmgr.Manager, err error) {
	wire.Build(
		providersmgr.NewManager,
		providersmgr.GetConfigFromEnv,
		ProvideSQLProviderRepository,
	)

	return
}

func ProvideShareApplication() (a *shareapp.ShareApplication, err error) {
	wire.Build(
		shareapp.New,
		ProvideShareService,
		ProvideSQLShareRepository,
		ProvideSQLProjectRepository,
	)

	return
}

func ProvideProjectApplication() (a *projectapp.ProjectApplication, err error) {
	wire.Build(
		projectapp.New,
		ProvideProjectService,
		ProvideSQLProjectRepository,
		ProvideProviderService,
		ProvideSQLProviderRepository,
		ProvideSQLShareRepository,
	)

	return
}

func ProvideAuthenticationManager() (am *authenticationmgr.Manager, err error) {
	wire.Build(
		authenticationmgr.NewManager,
		ProvideSQLProjectRepository,
		ProvideProviderManager,
		ProvideUserService,
	)

	return
}

func ProvideRESTServer() (s *rest.Server, err error) {
	wire.Build(
		rest.New,
		rest.GetConfigFromEnv,
		ProvideShareApplication,
		ProvideProjectApplication,
		ProvideAuthenticationManager,
	)

	return
}
