package projectapp

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	pem "go.openfort.xyz/shield/internal/adapters/authenticators/identity/custom_identity"
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/notifications"
	"go.openfort.xyz/shield/internal/core/domain/usercontact"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/pkg/otp"
	"go.openfort.xyz/shield/pkg/random"
	"go.openfort.xyz/shield/pkg/validation"

	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/logger"
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

const OTP_EMAIL_SUBJECT = "Openfort OTP"

// 2 request per hour
const DEFAULT_PROJECT_SMS_OTP_RATE_LIMIT = 2

// 120 request per hour
const DEFAULT_PROJECT_EMAIL_OTP_RATE_LIMIT = 120

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

	err = a.projectSvc.SaveProjectRateLimits(ctx, proj.ID, DEFAULT_PROJECT_SMS_OTP_RATE_LIMIT, DEFAULT_PROJECT_EMAIL_OTP_RATE_LIMIT)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to save project rate limits", logger.Error(err))
		return nil, err
	}

	var o projectOptions
	for _, opt := range opts {
		opt(&o)
	}

	if o.generateEncryptionKey {
		part, err := a.registerEncryptionKey(ctx, proj.ID)
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

func (a *ProjectApplication) ResetAPISecret(ctx context.Context) (string, error) {
	a.logger.InfoContext(ctx, "resetting API secret")
	projectID := contexter.GetProjectID(ctx)
	newAPISecretBytes := make([]byte, 32)
	_, err := rand.Read(newAPISecretBytes)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to generate new API secret", logger.Error(err))
		return "", fromDomainError(err)
	}
	encryptedSecret, err := bcrypt.GenerateFromPassword(newAPISecretBytes, bcrypt.DefaultCost)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to encrypt new API secret", logger.Error(err))
		return "", fromDomainError(err)
	}
	err = a.projectRepo.UpdateAPISecret(ctx, projectID, string(encryptedSecret))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to update API secret", logger.Error(err))
		return "", fromDomainError(err)
	}
	return hex.EncodeToString(newAPISecretBytes), nil
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

	err = a.projectRepo.Update2FA(ctx, projectID, true)
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
		prov, err := a.providerRepo.GetByProjectAndType(ctx, projectID, provider.TypeOpenfort)
		if err != nil && !errors.Is(err, domainErrors.ErrProviderNotFound) {
			a.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
			return nil, fromDomainError(err)
		}
		if err == nil && prov != nil {
			return nil, ErrProviderAlreadyExists
		}
		providers = append(providers, &provider.Provider{ProjectID: projectID, Type: provider.TypeOpenfort, Config: &provider.OpenfortConfig{PublishableKey: *cfg.openfortPublishableKey}})
	}

	if cfg.jwkURL != nil && cfg.pem != nil {
		return nil, ErrJWKPemConflict
	}

	if cfg.jwkURL != nil {
		prov, err := a.providerRepo.GetByProjectAndType(ctx, projectID, provider.TypeCustom)
		if err != nil && !errors.Is(err, domainErrors.ErrProviderNotFound) {
			a.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
			return nil, fromDomainError(err)
		}
		if err == nil && prov != nil {
			return nil, ErrProviderAlreadyExists
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
		prov, err := a.providerRepo.GetByProjectAndType(ctx, projectID, provider.TypeCustom)
		if err != nil && !errors.Is(err, domainErrors.ErrProviderNotFound) {
			a.logger.ErrorContext(ctx, "failed to get provider", logger.Error(err))
			return nil, fromDomainError(err)
		}
		if err == nil && prov != nil {
			return nil, ErrProviderAlreadyExists
		}
		providers = append(providers, &provider.Provider{ProjectID: projectID, Type: provider.TypeCustom, Config: &provider.CustomConfig{PEM: *cfg.pem, KeyType: cfg.keyType, CookieFieldName: cfg.cookieFieldName}})
	}

	if len(providers) == 0 {
		return nil, ErrNoProviderSpecified
	}

	for _, prov := range providers {
		err := a.providerSvc.Configure(ctx, prov)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to create provider", logger.Error(err))
			return nil, fromDomainError(err)
		}
	}

	return providers, nil
}

func (a *ProjectApplication) verifyAndSaveUserContacts(ctx context.Context, userId string, email *string, phone *string) error {
	if email != nil {
		emailHash := sha512.Sum512([]byte(*email))
		emailHashStr := fmt.Sprintf("%x", emailHash)

		userContactInfo, err := a.userContactRepo.GetByUserID(ctx, userId)
		if err != nil && err != domainErrors.ErrUserContactNotFound {
			return err
		} else if err == domainErrors.ErrUserContactNotFound {
			err = a.userContactRepo.Save(ctx, &usercontact.UserContact{ExternalUserID: userId, Email: emailHashStr})
			if err != nil {
				return err
			}
		} else {
			if userContactInfo.Email != "" && emailHashStr != userContactInfo.Email {
				return ErrUserContactInformationMismatch
			}
		}
	} else if phone != nil {
		phoneHash := sha512.Sum512([]byte(*phone))
		phoneHashStr := fmt.Sprintf("%x", phoneHash)

		userContactInfo, err := a.userContactRepo.GetByUserID(ctx, userId)
		if err != nil && err != domainErrors.ErrUserContactNotFound {
			return err
		} else if err == domainErrors.ErrUserContactNotFound {
			err = a.userContactRepo.Save(ctx, &usercontact.UserContact{ExternalUserID: userId, Phone: phoneHashStr})
			if err != nil {
				return err
			}
		} else {
			if userContactInfo.Phone != "" && phoneHashStr != userContactInfo.Phone {
				return ErrUserContactInformationMismatch
			}
		}
	}

	return nil
}

func (a *ProjectApplication) GenerateOTP(ctx context.Context, userId string, skipVerification bool, email *string, phone *string) error {
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

	// we do not rate limit requests where we skip verification
	// because in this case we don't send OTP anyway,
	// and after such requests users can only create new accounts but not recover existing ones
	if !skipVerification {
		allow := false

		if email != nil {
			allow = a.rateLimiter.TrackRequest(projectID, project.EmailRateLimit)
		} else if phone != nil {
			allow = a.rateLimiter.TrackRequest(projectID, project.SMSRateLimit)
		} else {
			return ErrNoUserContactInformationProvided
		}

		if !allow {
			return ErrOTPRateLimitExceeded
		}
	}

	err = a.verifyAndSaveUserContacts(ctx, userId, email, phone)
	if err != nil {
		return err
	}

	otpCode, err := a.otpService.GenerateOTP(ctx, userId, skipVerification)
	if err != nil {
		return fromDomainError(err)
	}

	// usually this flag will be used at sign up phase,
	// if someone tries to use it during sign in verifications in other endpoints will fail
	if skipVerification {
		return nil
	}

	if email != nil {
		if !validation.IsValidEmail(*email) {
			return ErrEmailIsInvalid
		}

		price, err := a.notificationService.SendEmail(ctx, *email, OTP_EMAIL_SUBJECT, otpCode, userId)
		if err != nil {
			return err
		}

		err = a.notificationsRepo.Save(ctx, &notifications.Notification{ProjectID: projectID, ExternalUserID: userId, NotifType: notifications.EmailNotificationType, Price: price})
		if err != nil {
			return err
		}

		return nil
	} else if phone != nil {
		if !validation.IsValidPhoneNumber(*phone) {
			return ErrPhoneNumberIsInvalid
		}

		price, err := a.notificationService.SendSMS(ctx, *phone, otpCode)
		if err != nil {
			return err
		}

		err = a.notificationsRepo.Save(ctx, &notifications.Notification{ProjectID: projectID, ExternalUserID: userId, NotifType: notifications.SMSNotificationType, Price: price})
		if err != nil {
			return err
		}

		return nil
	} else {
		return ErrOTPUserInfoMissing
	}
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

	if cfg.jwkURL != nil {
		if prov.Type != provider.TypeCustom {
			return ErrProviderMismatch
		}

		err = a.providerRepo.UpdateCustom(ctx, &provider.CustomConfig{ProviderID: providerID, JWK: *cfg.jwkURL, CookieFieldName: cfg.cookieFieldName})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to update custom provider", logger.Error(err))
			return fromDomainError(err)
		}
	}

	if cfg.openfortPublishableKey != nil {
		if prov.Type != provider.TypeOpenfort {
			return ErrProviderMismatch
		}

		err = a.providerRepo.UpdateOpenfort(ctx, &provider.OpenfortConfig{ProviderID: providerID, PublishableKey: *cfg.openfortPublishableKey})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to update openfort provider", logger.Error(err))
			return fromDomainError(err)
		}
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

		err = a.providerRepo.UpdateCustom(ctx, &provider.CustomConfig{ProviderID: providerID, PEM: *cfg.pem, KeyType: cfg.keyType, CookieFieldName: cfg.cookieFieldName})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to update custom provider", logger.Error(err))
			return fromDomainError(err)
		}
	}

	if cfg.cookieFieldName != nil {
		if prov.Type != provider.TypeCustom {
			return ErrProviderMismatch
		}
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

	err = a.providerRepo.Delete(ctx, providerID)
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

	builder, err := a.encryptionFactory.CreateEncryptionKeyBuilder(factories.Plain, isMigrated, false)
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

	var encryptedShares []*share.Share
	for _, shr := range shares {
		if shr.EncryptionParameters != nil || shr.Entropy != share.EntropyNone {
			continue
		}

		cypher := a.encryptionFactory.CreateEncryptionStrategy(encryptionKey)
		shr.Secret, err = cypher.Encrypt(shr.Secret)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to encrypt share", logger.Error(err))
			return fromDomainError(err)
		}

		shr.Entropy = share.EntropyProject

		encryptedShares = append(encryptedShares, shr)
	}

	for _, encryptedShare := range encryptedShares {
		err = a.sharesRepo.UpdateProjectEncryption(ctx, encryptedShare.ID, encryptedShare.Secret)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to update share", logger.Error(err))
			return fromDomainError(err)
		}
	}

	return nil
}

func (a *ProjectApplication) RegisterEncryptionSession(ctx context.Context, encryptionPart string, userId string, otpCode *string) (string, error) {
	a.logger.InfoContext(ctx, "registering encryption session")
	projectID := contexter.GetProjectID(ctx)

	proj, err := a.projectRepo.Get(ctx, projectID)
	if err != nil {
		return "", err
	}

	otpVerified := false

	if proj.Enable2FA {
		// in case OTP was generated with `SkipVerification` flag we might send there empty string
		code := ""
		if otpCode != nil {
			code = *otpCode
		}

		otpRequest, err := a.otpService.VerifyOTP(ctx, userId, code)
		if err != nil {
			if err == domainErrors.ErrDataInDBNotFound {
				return "", ErrOTPRecordNotFound
			}
			return "", err
		}

		if !otpRequest.SkipVerification {
			otpVerified = true
		}
	}

	sessionID := uuid.NewString()

	encPartData := share.EncryptionPart{
		EncPart:     encryptionPart,
		UserID:      userId,
		OTPVerified: otpVerified,
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
	err = a.encryptionPartsRepo.Set(ctx, sessionID, string(encPartDataBytes), &options)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to set encryption part", logger.Error(err))
		return "", fromDomainError(err)
	}

	return sessionID, nil
}

func (a *ProjectApplication) RegisterEncryptionKey(ctx context.Context) (string, error) {
	a.logger.InfoContext(ctx, "registering encryption key")
	projectID := contexter.GetProjectID(ctx)

	ep, err := a.projectRepo.GetEncryptionPart(ctx, projectID)
	if err != nil && !errors.Is(err, domainErrors.ErrEncryptionPartNotFound) {
		a.logger.ErrorContext(ctx, "failed to get encryption part", logger.Error(err))
		return "", fromDomainError(err)
	}

	if ep != "" {
		a.logger.Warn("encryption part already exists", slog.String("project_id", projectID))
		return "", ErrEncryptionPartAlreadyExists
	}

	externalPart, err := a.registerEncryptionKey(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to register encryption key", logger.Error(err))
		return "", fromDomainError(err)
	}

	return externalPart, nil
}

func (a *ProjectApplication) registerEncryptionKey(ctx context.Context, projectID string) (externalPart string, err error) {
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

	err = a.projectSvc.SetEncryptionPart(ctx, projectID, storedPart)
	if err != nil {
		return "", err
	}

	err = a.projectRepo.CreateMigration(ctx, projectID, true)
	if err != nil {
		a.logger.Error("failed to create migration", logger.Error(err))
		return "", ErrInternal
	}

	return projectPart, nil
}
