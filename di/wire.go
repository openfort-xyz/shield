//go:build wireinject
// +build wireinject

package di

import (
	"time"

	"github.com/google/wire"
	"github.com/openfort-xyz/shield/internal/adapters/authenticators"
	"github.com/openfort-xyz/shield/internal/adapters/authenticators/identity"
	ofidty "github.com/openfort-xyz/shield/internal/adapters/authenticators/identity/openfort_identity"
	"github.com/openfort-xyz/shield/internal/adapters/encryption"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/bunt"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/bunt/encryptionpartsrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/keychainrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/notificationsrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/projectrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/providerrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/sharerepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/usercontactrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/userrepo"
	"github.com/openfort-xyz/shield/internal/applications/healthzapp"
	"github.com/openfort-xyz/shield/internal/applications/notificationsapp"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shamirjob"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/internal/core/services/projectsvc"
	"github.com/openfort-xyz/shield/internal/core/services/providersvc"
	"github.com/openfort-xyz/shield/internal/core/services/sharesvc"
	"github.com/openfort-xyz/shield/internal/core/services/usersvc"
	"github.com/openfort-xyz/shield/pkg/otp"
)

func ProvideSQL() (c *sql.Client, err error) {
	wire.Build(
		sql.New,
		sql.GetConfigFromEnv,
	)

	return
}

func ProvideBuntDB() (c *bunt.Client, err error) {
	wire.Build(
		bunt.New,
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

func ProvideSQLKeychainRepository() (r repositories.KeychainRepository, err error) {
	wire.Build(
		keychainrepo.New,
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

func ProvideSQLNotificationsRepository() (r repositories.NotificationsRepository, err error) {
	wire.Build(
		notificationsrepo.New,
		ProvideSQL,
	)

	return
}

func ProvideSQLUserContactRepository() (r repositories.UserContactRepository, err error) {
	wire.Build(
		usercontactrepo.New,
		ProvideSQL,
	)

	return
}

func ProvideInMemoryEncryptionPartsRepository() (r repositories.EncryptionPartsRepository, err error) {
	wire.Build(
		encryptionpartsrepo.New,
		ProvideBuntDB,
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

func ProvideEncryptionFactory() (f factories.EncryptionFactory, err error) {
	wire.Build(
		encryption.NewEncryptionFactory,
		ProvideInMemoryEncryptionPartsRepository,
		ProvideSQLProjectRepository,
	)

	return
}

func ProvideShareService() (s services.ShareService, err error) {
	wire.Build(
		sharesvc.New,
		ProvideSQLShareRepository,
		ProvideSQLKeychainRepository,
		ProvideEncryptionFactory,
	)

	return
}

func ProvideShamirJob() (j *shamirjob.Job, err error) {
	wire.Build(
		shamirjob.New,
		ProvideSQLProjectRepository,
		ProvideSQLShareRepository,
	)

	return
}

func ProvideShareApplication() (a *shareapp.ShareApplication, err error) {
	wire.Build(
		shareapp.New,
		ProvideShareService,
		ProvideSQLShareRepository,
		ProvideSQLProjectRepository,
		ProvideSQLUserRepository,
		ProvideSQLKeychainRepository,
		ProvideEncryptionFactory,
		ProvideShamirJob,
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
		ProvideSQLNotificationsRepository,
		ProvideSQLUserContactRepository,
		ProvideEncryptionFactory,
		ProvideInMemoryEncryptionPartsRepository,
		ProvideOTPService,
		ProvideNotificationService,
		ProvideProjectRateLimiter,
	)

	return
}

func ProvideAuthenticationFactory() (f factories.AuthenticationFactory, err error) {
	wire.Build(
		authenticators.NewAuthenticatorFactory,
		ProvideUserService,
		ProvideSQLProjectRepository,
	)

	return
}

func ProvideIdentityFactory() (f factories.IdentityFactory, err error) {
	wire.Build(
		identity.NewIdentityFactory,
		ofidty.GetConfigFromEnv,
		ProvideSQLProviderRepository,
	)

	return
}

func ProvideHealthzApplication() (a *healthzapp.Application, err error) {
	wire.Build(
		ProvideSQL,
		healthzapp.New,
	)

	return
}

type clockImpl struct{}

func (c clockImpl) Now() time.Time {
	return time.Now()
}

// Two different function which return same struct under different interfaces
// because wire quires this. At compile time it expects to receive exact interface from exact package
func ProvideOTPClock() otp.Clock {
	return clockImpl{}
}
func ProvideProjectClock() projectapp.Clock {
	return clockImpl{}
}

func ProvideOnboardingTrackerConfig() otp.OnboardingTrackerConfig {
	return otp.OnboardingTrackerConfig{
		WindowMS:              otp.DefaultSecurityConfig.UserOnboardingWindowMS,
		OTPGenerationWindowMS: otp.DefaultSecurityConfig.OTPGenerationWindowMS,
		MaxAttempts:           otp.DefaultSecurityConfig.MaxUserOnboardAttempts,
	}
}

func ProvideOnboardingTracker() (t *otp.OnboardingTracker, err error) {
	wire.Build(
		otp.NewOnboardingTracker,
		ProvideOnboardingTrackerConfig,
		ProvideOTPClock,
	)

	return
}

func ProvideOTPService() (s *otp.InMemoryOTPService, err error) {
	wire.Build(
		otp.NewInMemoryOTPService,
		ProvideInMemoryEncryptionPartsRepository,
		ProvideOnboardingTracker,
		wire.Value(otp.DefaultSecurityConfig),
		ProvideOTPClock,
	)

	return
}

func ProvideProjectRateLimiter() (s *projectapp.RequestTracker, err error) {
	wire.Build(
		projectapp.NewRequestTracker,
		ProvideProjectClock,
	)

	return
}

func NewNotificationService() (services.NotificationsService, error) {
	app, err := notificationsapp.NewNotificationApp()
	if err != nil {
		return nil, err
	}
	return app, nil
}

func ProvideNotificationService() (c services.NotificationsService, err error) {
	wire.Build(
		NewNotificationService,
	)
	return
}

func ProvideRESTServer() (s *rest.Server, err error) {
	wire.Build(
		rest.New,
		rest.GetConfigFromEnv,
		ProvideShareApplication,
		ProvideProjectApplication,
		ProvideHealthzApplication,
		ProvideUserService,
		ProvideAuthenticationFactory,
		ProvideIdentityFactory,
		ProvideProjectService,
	)

	return
}
