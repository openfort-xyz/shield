package projectapp

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	pem "github.com/openfort-xyz/shield/internal/adapters/authenticators/identity/custom_identity"

	"github.com/google/uuid"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/notifications"
	"github.com/openfort-xyz/shield/internal/core/domain/usercontact"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"
	"github.com/openfort-xyz/shield/pkg/otp"
	"github.com/openfort-xyz/shield/pkg/random"
	"github.com/openfort-xyz/shield/pkg/validation"
	"github.com/tidwall/buntdb"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/contexter"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/jwk"
	"github.com/openfort-xyz/shield/pkg/logger"
)

type ProjectApplication struct {
	projectSvc          services.ProjectService
	projectRepo         repositories.ProjectRepository
	providerSvc         services.ProviderService
	providerRepo        repositories.ProviderRepository
	sharesRepo          repositories.ShareRepository
	notificationsRepo   repositories.NotificationsRepository
	userContactRepo     repositories.UserContactRepository
	logger              *slog.Logger
	encryptionFactory   factories.EncryptionFactory
	encryptionPartsRepo repositories.EncryptionPartsRepository
	otpService          *otp.InMemoryOTPService
	notificationService services.NotificationsService
	rateLimiter         *RequestTracker
}

const OTPEmailSubject = "Openfort OTP"

// 2 request per hour
const DefaultProjectSMSOTPRateLimit = 2

// 120 request per hour
const DefaultProjectEmailOTPRateLimit = 120

func New(
	projectSvc services.ProjectService,
	projectRepo repositories.ProjectRepository,
	providerSvc services.ProviderService,
	providerRepo repositories.ProviderRepository,
	sharesRepo repositories.ShareRepository,
	notificationsRepo repositories.NotificationsRepository,
	userContactRepo repositories.UserContactRepository,
	encryptionFactory factories.EncryptionFactory,
	encryptionPartsRepo repositories.EncryptionPartsRepository,
	otpService *otp.InMemoryOTPService,
	notificationService services.NotificationsService,
	rateLimiter *RequestTracker,
) *ProjectApplication {
	return &ProjectApplication{
		projectSvc:          projectSvc,
		projectRepo:         projectRepo,
		providerSvc:         providerSvc,
		providerRepo:        providerRepo,
		sharesRepo:          sharesRepo,
		notificationsRepo:   notificationsRepo,
		userContactRepo:     userContactRepo,
		logger:              logger.New("project_application"),
		encryptionFactory:   encryptionFactory,
		encryptionPartsRepo: encryptionPartsRepo,
		otpService:          otpService,
		notificationService: notificationService,
		rateLimiter:         rateLimiter,
	}
}

func (a *ProjectApplication) CreateProject(ctx context.Context, name string, enable2fa bool, opts ...ProjectOption) (*project.Project, error) {
	a.logger.InfoContext(ctx, "creating project")

	proj, err := a.projectSvc.Create(ctx, name, enable2fa)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create project", logger.Error(err))
		return nil, fromDomainError(err)
	}

	err = a.projectSvc.SaveProjectRateLimits(ctx, proj.ID, DefaultProjectSMSOTPRateLimit, DefaultProjectEmailOTPRateLimit)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to save project rate limits", logger.Error(err))
		return nil, err
	}

	var o projectOptions
	for _, opt := range opts {
		opt(&o)
	}

	if o.generateEncryptionKey {
		part, err := a.registerEncryptionKey(ctx, proj.ID, project.StillCurrent(proj.APIKey, proj.APISecret))
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to register encryption key", logger.Error(err))
			errD := a.projectRepo.Delete(ctx, proj.ID)
			if errD != nil {
				a.logger.Error("failed to delete project", logger.Error(errD))
				err = errors.Join(err, errD)
			}
			return nil, fromDomainError(err)
		}

		proj.EncryptionPart = part
	}

	return proj, nil
}

func (a *ProjectApplication) GetProject(ctx context.Context) (*project.Project, error) {
	a.logger.InfoContext(ctx, "getting project")
	projectID := contexter.GetProjectID(ctx)

	proj, err := a.projectRepo.Get(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get project", logger.Error(err))
		return nil, fromDomainError(err)
	}

	return proj, nil
}

// DeleteProject irreversibly hard-deletes the authenticated project and all
// its data: providers, users, external users, shares (recovery share halves),
// keychains, encryption parts, notifications, rate limits, passkey references
// and shamir migration records. After this call the project's API key and
// secret no longer authenticate.
func (a *ProjectApplication) DeleteProject(ctx context.Context) error {
	projectID := contexter.GetProjectID(ctx)
	a.logger.InfoContext(ctx, "deleting project and all its data", slog.String("project_id", projectID))

	err := a.projectRepo.HardDelete(ctx, projectID, stillCurrent(ctx))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to delete project", logger.Error(err))
		return fromDomainError(err)
	}

	a.logger.InfoContext(ctx, "project deleted", slog.String("project_id", projectID))
	return nil
}

// SetProjectDisabled disables or re-enables the project with apiKey.
func (a *ProjectApplication) SetProjectDisabled(ctx context.Context, apiKey string, disabled bool) error {
	a.logger.InfoContext(ctx, "setting project disabled", slog.Bool("disabled", disabled))

	if err := a.projectSvc.SetDisabled(ctx, apiKey, disabled); err != nil {
		a.logger.ErrorContext(ctx, "failed to set project disabled", logger.Error(err))
		return fromDomainError(err)
	}
	return nil
}

func (a *ProjectApplication) Enable2FA(ctx context.Context) error {
	a.logger.InfoContext(ctx, "enabling 2FA for project")
	projectID := contexter.GetProjectID(ctx)

	proj, err := a.projectRepo.Get(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get project", logger.Error(err))
		return fromDomainError(err)
	}

	// If 2FA is already enabled, return error
	if proj.Enable2FA {
		a.logger.InfoContext(ctx, "2FA already enabled for project", slog.String("project_id", projectID))
		return ErrProject2FAAlreadyEnabled
	}

	err = a.projectRepo.Update2FA(ctx, projectID, true, stillCurrent(ctx))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to update 2FA", logger.Error(err))
		return fromDomainError(err)
	}

	a.logger.InfoContext(ctx, "2FA enabled successfully", slog.String("project_id", projectID))
	return nil
}

func (a *ProjectApplication) AddProviders(ctx context.Context, opts ...ProviderOption) ([]*provider.Provider, error) {
	a.logger.InfoContext(ctx, "adding providers")
	projectID := contexter.GetProjectID(ctx)

	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var providers []*provider.Provider
	if cfg.openfortPublishableKey != nil {
		providers = append(providers, &provider.Provider{ProjectID: projectID, Type: provider.TypeOpenfort, Config: &provider.OpenfortConfig{PublishableKey: *cfg.openfortPublishableKey}})
	}

	if cfg.jwkURL != nil && cfg.pem != nil {
		return nil, ErrJWKPemConflict
	}

	if cfg.jwkURL != nil {
		if err := jwk.CheckTenantURL(*cfg.jwkURL); err != nil {
			a.logger.WarnContext(ctx, "rejected custom JWK URL", logger.Error(err))
			return nil, ErrJWKURLNotAllowed
		}
		providers = append(providers, &provider.Provider{ProjectID: projectID, Type: provider.TypeCustom, Config: &provider.CustomConfig{JWK: *cfg.jwkURL, CookieFieldName: cfg.cookieFieldName}})
	}

	if cfg.pem != nil {
		if cfg.keyType == provider.KeyTypeUnknown {
			return nil, ErrKeyTypeNotSpecified
		}
		err := pem.CheckPEM([]byte(*cfg.pem), cfg.keyType)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to validate PEM", logger.Error(err))
			return nil, ErrInvalidPemCertificate
		}
		providers = append(providers, &provider.Provider{ProjectID: projectID, Type: provider.TypeCustom, Config: &provider.CustomConfig{PEM: *cfg.pem, KeyType: cfg.keyType, CookieFieldName: cfg.cookieFieldName}})
	}

	if len(providers) == 0 {
		return nil, ErrNoProviderSpecified
	}

	if err := a.providerSvc.Configure(ctx, projectID, providers, stillCurrent(ctx)); err != nil {
		a.logger.ErrorContext(ctx, "failed to create providers", logger.Error(err))
		return nil, fromDomainError(err)
	}

	return providers, nil
}

// verifyAndSaveUserContacts checks the contact against the one bound to the
// user, binding it when there is none. The bool reports whether this call
// bound it.
func (a *ProjectApplication) verifyAndSaveUserContacts(ctx context.Context, userID string, email *string, phone *string) (bool, error) {
	if email != nil {
		emailHash := sha512.Sum512([]byte(*email))
		emailHashStr := fmt.Sprintf("%x", emailHash)

		userContactInfo, err := a.userContactRepo.GetByUserID(ctx, userID)
		switch {
		case err != nil && !errors.Is(err, domainErrors.ErrUserContactNotFound):
			return false, err
		case errors.Is(err, domainErrors.ErrUserContactNotFound):
			if err := a.userContactRepo.Save(ctx, &usercontact.UserContact{ExternalUserID: userID, Email: emailHashStr}); err != nil {
				return false, err
			}
			return true, nil
		case emailHashStr != userContactInfo.Email:
			return false, ErrUserContactInformationMismatch
		}
	} else if phone != nil {
		phoneHash := sha512.Sum512([]byte(*phone))
		phoneHashStr := fmt.Sprintf("%x", phoneHash)

		userContactInfo, err := a.userContactRepo.GetByUserID(ctx, userID)
		switch {
		case err != nil && !errors.Is(err, domainErrors.ErrUserContactNotFound):
			return false, err
		case errors.Is(err, domainErrors.ErrUserContactNotFound):
			if err := a.userContactRepo.Save(ctx, &usercontact.UserContact{ExternalUserID: userID, Phone: phoneHashStr}); err != nil {
				return false, err
			}
			return true, nil
		case phoneHashStr != userContactInfo.Phone:
			return false, ErrUserContactInformationMismatch
		}
	}

	return false, nil
}

func (a *ProjectApplication) GenerateOTP(ctx context.Context, userID string, skipVerification bool, email *string, phone *string) error {
	if reflect.ValueOf(a.notificationService).IsNil() {
		return ErrMissingNotificationService
	}

	projectID := contexter.GetProjectID(ctx)

	project, err := a.projectRepo.GetWithRateLimit(ctx, projectID)
	if err != nil {
		return fromDomainError(err)
	}

	if !project.Enable2FA {
		return ErrProjectDoesntHave2FA
	}

	// usually this flag will be used at sign up phase,
	// if someone tries to use it during sign in verifications in other endpoints will fail.
	// we do not rate limit requests where we skip verification
	// because in this case we don't send OTP anyway,
	// and after such requests users can only create new accounts but not recover existing ones
	if skipVerification {
		_, err := a.otpService.GenerateOTP(ctx, projectID, userID, true)
		return fromDomainError(err)
	}

	var allow bool
	switch {
	case email != nil:
		if !validation.IsValidEmail(*email) {
			return ErrEmailIsInvalid
		}
		allow = a.rateLimiter.TrackRequest(projectID, project.EmailRateLimit)
	case phone != nil:
		if !validation.IsValidPhoneNumber(*phone) {
			return ErrPhoneNumberIsInvalid
		}
		allow = a.rateLimiter.TrackRequest(projectID, project.SMSRateLimit)
	default:
		return ErrNoUserContactInformationProvided
	}

	if !allow {
		return ErrOTPRateLimitExceeded
	}

	// Before an OTP exists, so a contact other than the bound one never
	// receives or replaces one.
	contactCreated, err := a.verifyAndSaveUserContacts(ctx, userID, email, phone)
	if err != nil {
		return err
	}

	notif, err := a.sendOTP(ctx, projectID, userID, email, phone)
	if err != nil {
		// The contact got no OTP, so it has not earned the binding, and
		// keeping it would lock the user out of correcting a mistyped one.
		if contactCreated {
			if delErr := a.userContactRepo.DeleteByUserID(ctx, userID); delErr != nil {
				err = errors.Join(err, delErr)
			}
		}
		return err
	}

	return a.notificationsRepo.Save(ctx, notif)
}

// sendOTP issues an OTP and delivers it to the contact. An OTP whose delivery
// failed is discarded, so a failed request leaves none pending.
func (a *ProjectApplication) sendOTP(ctx context.Context, projectID, userID string, email, phone *string) (*notifications.Notification, error) {
	otpCode, err := a.otpService.GenerateOTP(ctx, projectID, userID, false)
	if err != nil {
		return nil, fromDomainError(err)
	}

	notif := &notifications.Notification{ProjectID: projectID, ExternalUserID: userID}
	switch {
	case email != nil:
		notif.NotifType = notifications.EmailNotificationType
		notif.Price, err = a.notificationService.SendEmail(ctx, *email, OTPEmailSubject, otpCode, userID)
	case phone != nil:
		notif.NotifType = notifications.SMSNotificationType
		notif.Price, err = a.notificationService.SendSMS(ctx, *phone, otpCode)
	default:
		err = ErrOTPUserInfoMissing
	}
	if err != nil {
		if delErr := a.otpService.DeleteOTP(ctx, projectID, userID); delErr != nil {
			err = errors.Join(err, delErr)
		}
		return nil, err
	}

	return notif, nil
}

func (a *ProjectApplication) GetProviders(ctx context.Context) ([]*provider.Provider, error) {
	a.logger.InfoContext(ctx, "listing providers")
	projectID := contexter.GetProjectID(ctx)

	providers, err := a.providerRepo.List(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list providers", logger.Error(err))
		return nil, fromDomainError(err)
	}

	return providers, nil
}

func (a *ProjectApplication) GetProviderDetail(ctx context.Context, providerID string) (*provider.Provider, error) {
	a.logger.InfoContext(ctx, "getting provider detail")
	projectID := contexter.GetProjectID(ctx)

	prov, err := a.providerRepo.Get(ctx, providerID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return nil, fromDomainError(err)
	}

	if prov.ProjectID != projectID {
		a.logger.ErrorContext(ctx, "unauthorized access, trying to access provider from different project", slog.String("project_id", projectID), slog.String("provider_project_id", prov.ProjectID))
		return nil, ErrProviderNotFound
	}

	return prov, nil
}

func (a *ProjectApplication) UpdateProvider(ctx context.Context, providerID string, opts ...ProviderOption) error {
	a.logger.InfoContext(ctx, "updating provider")
	projectID := contexter.GetProjectID(ctx)

	prov, err := a.providerRepo.Get(ctx, providerID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return fromDomainError(err)
	}

	if prov.ProjectID != projectID {
		a.logger.ErrorContext(ctx, "unauthorized access, trying to update provider from different project", slog.String("project_id", projectID), slog.String("provider_project_id", prov.ProjectID))
		return ErrProviderNotFound
	}

	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.cookieFieldName != nil && prov.Type != provider.TypeCustom {
		a.logger.ErrorContext(ctx, "cookie field name can only be set for custom providers")
		return ErrProviderMismatch
	}

	// Validated in full before one write, so a refused request changes
	// nothing. A PEM replaces a JWK sent alongside it.
	var custom *provider.CustomConfig
	if cfg.jwkURL != nil {
		if prov.Type != provider.TypeCustom {
			return ErrProviderMismatch
		}
		if err := jwk.CheckTenantURL(*cfg.jwkURL); err != nil {
			a.logger.WarnContext(ctx, "rejected custom JWK URL", logger.Error(err))
			return ErrJWKURLNotAllowed
		}
		custom = &provider.CustomConfig{ProviderID: providerID, JWK: *cfg.jwkURL, CookieFieldName: cfg.cookieFieldName}
	}

	if cfg.openfortPublishableKey != nil && prov.Type != provider.TypeOpenfort {
		return ErrProviderMismatch
	}

	if cfg.pem != nil {
		if prov.Type != provider.TypeCustom {
			return ErrProviderMismatch
		}

		if prov.Config.(*provider.CustomConfig).KeyType == provider.KeyTypeUnknown && cfg.keyType == provider.KeyTypeUnknown {
			return ErrKeyTypeNotSpecified
		}

		err := pem.CheckPEM([]byte(*cfg.pem), cfg.keyType)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to validate PEM", logger.Error(err))
			return ErrInvalidPemCertificate
		}
		custom = &provider.CustomConfig{ProviderID: providerID, PEM: *cfg.pem, KeyType: cfg.keyType, CookieFieldName: cfg.cookieFieldName}
	}

	if cfg.cookieFieldName != nil && custom == nil {
		custom = &provider.CustomConfig{ProviderID: providerID, CookieFieldName: cfg.cookieFieldName}
	}

	switch {
	case custom != nil:
		err = a.providerRepo.UpdateCustom(ctx, projectID, custom, stillCurrent(ctx))
	case cfg.openfortPublishableKey != nil:
		err = a.providerRepo.UpdateOpenfort(ctx, projectID, &provider.OpenfortConfig{ProviderID: providerID, PublishableKey: *cfg.openfortPublishableKey}, stillCurrent(ctx))
	}
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to update provider", logger.Error(err))
		return fromDomainError(err)
	}
	return nil
}

func (a *ProjectApplication) RemoveProvider(ctx context.Context, providerID string) error {
	a.logger.InfoContext(ctx, "removing provider")
	projectID := contexter.GetProjectID(ctx)

	prov, err := a.providerRepo.Get(ctx, providerID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
		return fromDomainError(err)
	}

	if prov.ProjectID != projectID {
		a.logger.ErrorContext(ctx, "unauthorized access, trying to remove provider from different project", slog.String("project_id", projectID), slog.String("provider_project_id", prov.ProjectID))
		return ErrProviderNotFound
	}

	err = a.providerRepo.Delete(ctx, projectID, providerID, stillCurrent(ctx))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to remove provider", logger.Error(err))
		return fromDomainError(err)
	}

	return nil
}

func (a *ProjectApplication) EncryptProjectShares(ctx context.Context, externalPart string) error {
	a.logger.InfoContext(ctx, "encrypting project shares")
	projectID := contexter.GetProjectID(ctx)

	isMigrated, err := a.projectRepo.HasSuccessfulMigration(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to check migration", logger.Error(err))
		return ErrInternal
	}

	builder, err := a.encryptionFactory.CreateEncryptionKeyBuilder(factories.Plain, isMigrated, factories.OTPOptional)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create encryption key builder", logger.Error(err))
		return ErrInternal
	}

	err = builder.SetDatabasePart(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get encryption part", logger.Error(err))
		return fromDomainError(err)
	}

	err = builder.SetProjectPart(ctx, externalPart)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get encryption part", logger.Error(err))
		return fromDomainError(err)
	}

	encryptionKey, err := builder.Build(ctx)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to reconstruct encryption key", logger.Error(err))
		return ErrInvalidEncryptionPart
	}

	shares, err := a.sharesRepo.ListProjectIDAndEntropy(ctx, projectID, share.EntropyNone)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list shares", logger.Error(err))
		return fromDomainError(err)
	}

	guard := &share.EncryptionGuard{
		ProjectID:  projectID,
		StoredPart: builder.GetDatabasePart(ctx),
		Migrated:   isMigrated,
		Key:        encryptionKey,
	}
	cypher := a.encryptionFactory.CreateEncryptionStrategy(encryptionKey)
	for _, shr := range shares {
		if shr.EncryptionParameters != nil || shr.Entropy != share.EntropyNone {
			continue
		}

		encrypted, err := cypher.Encrypt(shr.Secret)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to encrypt share", logger.Error(err))
			return fromDomainError(err)
		}

		err = a.sharesRepo.UpdateProjectEncryption(ctx, shr.ID, shr.Secret, encrypted, guard)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to update share", logger.Error(err))
			return fromDomainError(err)
		}
	}

	return nil
}

func (a *ProjectApplication) RegisterEncryptionSession(ctx context.Context, encryptionPart string, userID string, otpCode *string) (string, error) {
	a.logger.InfoContext(ctx, "registering encryption session")
	projectID := contexter.GetProjectID(ctx)

	proj, err := a.projectRepo.Get(ctx, projectID)
	if err != nil {
		return "", err
	}

	otpVerified, signupOnly := false, false

	if proj.Enable2FA {
		otpRequest, err := a.otpService.VerifyOTP(ctx, projectID, userID, otpCode)
		if err != nil {
			if errors.Is(err, domainErrors.ErrDataInDBNotFound) {
				return "", ErrOTPRequired
			}
			return "", fromDomainError(err)
		}

		signupOnly = otpRequest.SkipVerification
		otpVerified = !signupOnly
	}

	sessionID := uuid.NewString()

	encPartData := share.EncryptionPart{
		EncPart:     encryptionPart,
		ProjectID:   projectID,
		UserID:      userID,
		OTPVerified: otpVerified,
		SignupOnly:  signupOnly,
	}
	encPartDataBytes, err := json.Marshal(encPartData)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to serialize encryption part with signer ID", logger.Error(err))
		return "", fromDomainError(err)
	}

	options := buntdb.SetOptions{
		Expires: true,
		TTL:     time.Duration(5*60*1000) * time.Millisecond, // 5 minutes TTL
	}
	err = a.encryptionPartsRepo.Set(ctx, share.EncryptionSessionKey(sessionID), string(encPartDataBytes), &options)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to set encryption part", logger.Error(err))
		return "", fromDomainError(err)
	}

	return sessionID, nil
}

func (a *ProjectApplication) RegisterEncryptionKey(ctx context.Context) (string, error) {
	a.logger.InfoContext(ctx, "registering encryption key")
	projectID := contexter.GetProjectID(ctx)

	externalPart, err := a.registerEncryptionKey(ctx, projectID, stillCurrent(ctx))
	if err != nil {
		if !errors.Is(err, domainErrors.ErrEncryptionPartAlreadyExists) {
			a.logger.ErrorContext(ctx, "failed to register encryption key", logger.Error(err))
		}
		return "", fromDomainError(err)
	}

	return externalPart, nil
}

// registerEncryptionKey creates the project's key once check passes under the
// project row lock. Registration at project creation checks the credentials
// it just issued; a request checks the ones it authenticated with.
func (a *ProjectApplication) registerEncryptionKey(ctx context.Context, projectID string, check project.CheckAPISecretFunc) (externalPart string, err error) {
	key, err := random.GenerateRandomString(32)
	if err != nil {
		a.logger.Error("failed to generate random key", logger.Error(err))
		return "", ErrInternal
	}

	reconstructionStrategy := a.encryptionFactory.CreateReconstructionStrategy(true)
	storedPart, projectPart, err := reconstructionStrategy.Split(key)
	if err != nil {
		a.logger.Error("failed to split encryption key", logger.Error(err))
		return "", ErrInternal
	}

	computed, err := cypher.KeyCheck(key)
	if err != nil {
		a.logger.Error("failed to compute key check", logger.Error(err))
		return "", ErrInternal
	}
	keyCheck := project.KeyCheck(computed)

	err = a.projectSvc.SetEncryptionKey(ctx, projectID, &project.EncryptionKey{Part: storedPart, KeyCheck: &keyCheck}, check)
	if err != nil {
		return "", err
	}

	return string(projectPart), nil
}
