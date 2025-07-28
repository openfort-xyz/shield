package projectapp

import (
	"context"
	"errors"
	"log/slog"

	pem "go.openfort.xyz/shield/internal/adapters/authenticators/identity/custom_identity"

	"github.com/google/uuid"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/pkg/random"

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
	logger              *slog.Logger
	encryptionFactory   factories.EncryptionFactory
	encryptionPartsRepo repositories.EncryptionPartsRepository
}

func New(projectSvc services.ProjectService, projectRepo repositories.ProjectRepository, providerSvc services.ProviderService, providerRepo repositories.ProviderRepository, sharesRepo repositories.ShareRepository, encryptionFactory factories.EncryptionFactory, encryptionPartsRepo repositories.EncryptionPartsRepository) *ProjectApplication {
	return &ProjectApplication{
		projectSvc:          projectSvc,
		projectRepo:         projectRepo,
		providerSvc:         providerSvc,
		providerRepo:        providerRepo,
		sharesRepo:          sharesRepo,
		logger:              logger.New("project_application"),
		encryptionFactory:   encryptionFactory,
		encryptionPartsRepo: encryptionPartsRepo,
	}
}

func (a *ProjectApplication) CreateProject(ctx context.Context, name string, opts ...ProjectOption) (*project.Project, error) {
	a.logger.InfoContext(ctx, "creating project")

	proj, err := a.projectSvc.Create(ctx, name)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create project", logger.Error(err))
		return nil, fromDomainError(err)
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
		providers = append(providers, &provider.Provider{ProjectID: projectID, Type: provider.TypeCustom, Config: &provider.CustomConfig{JWK: *cfg.jwkURL}})
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
		providers = append(providers, &provider.Provider{ProjectID: projectID, Type: provider.TypeCustom, Config: &provider.CustomConfig{PEM: *cfg.pem, KeyType: cfg.keyType}})
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

	if cfg.jwkURL != nil {
		if prov.Type != provider.TypeCustom {
			return ErrProviderMismatch
		}

		err = a.providerRepo.UpdateCustom(ctx, &provider.CustomConfig{ProviderID: providerID, JWK: *cfg.jwkURL})
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

		err = a.providerRepo.UpdateCustom(ctx, &provider.CustomConfig{ProviderID: providerID, PEM: *cfg.pem, KeyType: cfg.keyType})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to update custom provider", logger.Error(err))
			return fromDomainError(err)
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

	builder, err := a.encryptionFactory.CreateEncryptionKeyBuilder(factories.Plain, isMigrated)
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

func (a *ProjectApplication) RegisterEncryptionSession(ctx context.Context, encryptionPart string) (string, error) {
	a.logger.InfoContext(ctx, "registering encryption session")

	sessionID := uuid.NewString()
	err := a.encryptionPartsRepo.Set(ctx, sessionID, encryptionPart)
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
