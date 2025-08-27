package shareapp

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/keychain"

	"go.openfort.xyz/shield/internal/applications/shamirjob"

	"go.openfort.xyz/shield/internal/core/ports/factories"

	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/logger"
)

type ShareApplication struct {
	shareSvc           services.ShareService
	shareRepo          repositories.ShareRepository
	keychainRepository repositories.KeychainRepository
	projectRepo        repositories.ProjectRepository
	logger             *slog.Logger
	encryptionFactory  factories.EncryptionFactory
	shamirJob          *shamirjob.Job
}

func New(shareSvc services.ShareService, shareRepo repositories.ShareRepository, projectRepo repositories.ProjectRepository, keychainRepository repositories.KeychainRepository, encryptionFactory factories.EncryptionFactory, shamirJob *shamirjob.Job) *ShareApplication {
	return &ShareApplication{
		shareSvc:           shareSvc,
		shareRepo:          shareRepo,
		keychainRepository: keychainRepository,
		projectRepo:        projectRepo,
		logger:             logger.New("share_application"),
		encryptionFactory:  encryptionFactory,
		shamirJob:          shamirJob,
	}
}

func (a *ShareApplication) RegisterShare(ctx context.Context, shr *share.Share, opts ...Option) error {
	a.logger.InfoContext(ctx, "registering share")
	usrID := contexter.GetUserID(ctx)
	projID := contexter.GetProjectID(ctx)
	shr.UserID = usrID

	_, err := a.migrateToKeychainIfRequired(ctx, usrID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to migrate keychain shares", logger.Error(err))
		return fromDomainError(err)
	}

	var opt options
	for _, o := range opts {
		o(&opt)
	}

	var shrOpts []services.ShareOption
	if shr.RequiresEncryption() {
		encryptionKey, err := a.reconstructEncryptionKey(ctx, projID, opt)
		if err != nil {
			return err
		}

		shrOpts = append(shrOpts, services.WithEncryptionKey(encryptionKey))
	}

	err = a.shareSvc.Create(ctx, shr, shrOpts...)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create share", logger.Error(err))
		return fromDomainError(err)
	}

	return nil
}

func (a *ShareApplication) UpdateShare(ctx context.Context, shr *share.Share, reference string, opts ...Option) (*share.Share, error) {
	a.logger.InfoContext(ctx, "updating share")
	usrID := contexter.GetUserID(ctx)
	projID := contexter.GetProjectID(ctx)

	dbShare, err := a.shareSvc.Find(ctx, usrID, nil, &reference)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
		return nil, fromDomainError(err)
	}

	if shr.Entropy != 0 {
		dbShare.Entropy = shr.Entropy
	}

	if (shr.Entropy == share.EntropyNone || shr.Entropy == share.EntropyProject) && shr.EncryptionParameters != nil {
		shr.EncryptionParameters = nil
		dbShare.EncryptionParameters = nil
	}

	if shr.EncryptionParameters != nil {
		dbShare.EncryptionParameters = shr.EncryptionParameters
	}

	if shr.Secret != "" {
		dbShare.Secret = shr.Secret
	}

	var opt options
	for _, o := range opts {
		o(&opt)
	}

	if dbShare.RequiresEncryption() {
		encryptionKey, err := a.reconstructEncryptionKey(ctx, projID, opt)
		if err != nil {
			return nil, err
		}

		cypher := a.encryptionFactory.CreateEncryptionStrategy(encryptionKey)
		dbShare.Secret, err = cypher.Encrypt(dbShare.Secret)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to encrypt secret", logger.Error(err))
			return nil, ErrInternal
		}
	}

	err = a.shareRepo.Update(ctx, dbShare)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create share", logger.Error(err))
		return nil, fromDomainError(err)
	}

	return shr, nil
}

func (a *ShareApplication) GetShareEncryption(ctx context.Context) (share.Entropy, *share.EncryptionParameters, error) {
	a.logger.InfoContext(ctx, "getting share encryption & encryption parameters")
	usrID := contexter.GetUserID(ctx)

	shr, err := a.shareSvc.Find(ctx, usrID, nil, nil)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
		return 0, nil, fromDomainError(err)
	}

	return shr.Entropy, shr.EncryptionParameters, nil
}

func (a *ShareApplication) GetSharesEncryptionForReferences(ctx context.Context, references []string) map[string]share.Entropy {
	// This layer doesn't know anything about having to default to anything: it won't have any entry that
	// a) doesn't exist
	// b) exists but didn't belong to the same project as the requester user
	// So it's the transport layer's responsibility to implement this behavior
	return map[string]share.Entropy{"hello": share.EntropyNone}
}

func (a *ShareApplication) migrateToKeychainIfRequired(ctx context.Context, usrID string) (string, error) {
	userKeychain, err := a.keychainRepository.GetByUserID(ctx, usrID)
	if err != nil && !errors.Is(err, domainErrors.ErrKeychainNotFound) {
		a.logger.ErrorContext(ctx, "failed to get keychain by user ID", logger.Error(err))
		return "", err
	}

	if userKeychain == nil {
		userKeychain = &keychain.Keychain{
			ID:     uuid.NewString(),
			UserID: usrID,
		}

		err = a.keychainRepository.Create(ctx, userKeychain)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to create keychain", logger.Error(err))
			return "", err
		}
	}

	usrShr, err := a.shareRepo.GetByUserID(ctx, usrID)
	if err != nil && !errors.Is(err, domainErrors.ErrShareNotFound) {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
		return "", err
	}

	if usrShr != nil {
		usrShr.KeychainID = &userKeychain.ID
		ref := share.DefaultReference
		usrShr.Reference = &ref

		err = a.shareRepo.Update(ctx, usrShr)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to update share", logger.Error(err))
			return "", err
		}
	}

	return userKeychain.ID, nil
}

func (a *ShareApplication) GetKeychainShares(ctx context.Context, reference *string, opts ...Option) ([]*share.Share, error) {
	a.logger.InfoContext(ctx, "getting keychain shares")
	usrID := contexter.GetUserID(ctx)

	var opt options
	for _, o := range opts {
		o(&opt)
	}

	keychainID, err := a.migrateToKeychainIfRequired(ctx, usrID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to migrate keychain shares", logger.Error(err))
		return nil, fromDomainError(err)
	}

	if reference != nil {
		shr, err := a.shareRepo.GetByReference(ctx, *reference, keychainID)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to get share by reference", logger.Error(err))
			return nil, fromDomainError(err)
		}

		if shr.UserID != usrID {
			return nil, ErrShareNotFound
		}

		if shr.RequiresEncryption() {
			encryptionKey, err := a.reconstructEncryptionKey(ctx, contexter.GetProjectID(ctx), opt)
			if err != nil {
				return nil, err
			}

			cypher := a.encryptionFactory.CreateEncryptionStrategy(encryptionKey)
			shr.Secret, err = cypher.Decrypt(shr.Secret)
			if err != nil {
				a.logger.ErrorContext(ctx, "failed to decrypt secret", logger.Error(err))
				return nil, ErrInternal
			}
		}

		return []*share.Share{shr}, nil
	}

	shrs, err := a.shareRepo.ListByKeychainID(ctx, keychainID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list shares by keychain ID", logger.Error(err))
		return nil, fromDomainError(err)
	}

	if len(shrs) == 0 {
		return nil, nil
	}

	if shrs[0].RequiresEncryption() {
		encryptionKey, err := a.reconstructEncryptionKey(ctx, contexter.GetProjectID(ctx), opt)
		if err != nil {
			return nil, err
		}

		for _, shr := range shrs {
			cypher := a.encryptionFactory.CreateEncryptionStrategy(encryptionKey)
			shr.Secret, err = cypher.Decrypt(shr.Secret)
			if err != nil {
				a.logger.ErrorContext(ctx, "failed to decrypt secret", logger.Error(err))
				return nil, ErrInternal
			}
		}
	}

	return shrs, nil
}

func (a *ShareApplication) GetShare(ctx context.Context, opts ...Option) (*share.Share, error) {
	a.logger.InfoContext(ctx, "getting share")
	usrID := contexter.GetUserID(ctx)
	projID := contexter.GetProjectID(ctx)

	shr, err := a.shareSvc.Find(ctx, usrID, nil, nil)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
		return nil, fromDomainError(err)
	}

	var opt options
	for _, o := range opts {
		o(&opt)
	}

	if shr.RequiresEncryption() {
		encryptionKey, err := a.reconstructEncryptionKey(ctx, projID, opt)
		if err != nil {
			return nil, err
		}

		cypher := a.encryptionFactory.CreateEncryptionStrategy(encryptionKey)
		shr.Secret, err = cypher.Decrypt(shr.Secret)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to decrypt secret", logger.Error(err))
			return nil, ErrInternal
		}
	}

	return shr, nil
}

func (a *ShareApplication) DeleteShare(ctx context.Context, reference *string) error {
	a.logger.InfoContext(ctx, "deleting share")
	usrID := contexter.GetUserID(ctx)

	shr, err := a.shareSvc.Find(ctx, usrID, nil, reference)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
		return fromDomainError(err)
	}

	err = a.shareRepo.Delete(ctx, shr.ID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to delete share", logger.Error(err))
		return fromDomainError(err)
	}

	return nil
}

func (a *ShareApplication) reconstructEncryptionKey(ctx context.Context, projID string, opt options) (string, error) {
	var builderType factories.EncryptionKeyBuilderType
	var identifier string
	switch {
	case opt.encryptionPart != nil && *opt.encryptionPart != "":
		builderType = factories.Plain
		identifier = *opt.encryptionPart
	case opt.encryptionSession != nil && *opt.encryptionSession != "":
		builderType = factories.Session
		identifier = *opt.encryptionSession
	default:
		return "", ErrEncryptionPartRequired
	}

	isMigrated, err := a.projectRepo.HasSuccessfulMigration(ctx, projID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to check migration", logger.Error(err))
		return "", ErrInternal
	}

	builder, err := a.encryptionFactory.CreateEncryptionKeyBuilder(builderType, isMigrated)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create encryption key builder", logger.Error(err))
		return "", ErrInternal
	}

	err = builder.SetDatabasePart(ctx, projID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get database encryption part", logger.Error(err))
		return "", fromDomainError(err)
	}

	err = builder.SetProjectPart(ctx, identifier)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get project encryption part", logger.Error(err))
		return "", fromDomainError(err)
	}

	encryptionKey, err := builder.Build(ctx)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to reconstruct encryption key", logger.Error(err))
		return "", ErrInvalidEncryptionPart
	}

	if !isMigrated {
		ctxWithoutCancel := context.WithoutCancel(ctx)
		go func() {
			err = a.shamirJob.Execute(ctxWithoutCancel, projID, builder.GetDatabasePart(ctxWithoutCancel), builder.GetProjectPart(ctxWithoutCancel), encryptionKey)
			if err != nil {
				a.logger.ErrorContext(ctx, "failed to execute shamir job", logger.Error(err))
			}
		}()
	}

	return encryptionKey, nil
}

func (a *ShareApplication) GetShareStorageMethods(ctx context.Context) ([]*share.StorageMethod, error) {
	a.logger.InfoContext(ctx, "getting share storage methods")

	storageMethods, err := a.shareRepo.GetShareStorageMethods(ctx)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share storage methods", logger.Error(err))
		return nil, fromDomainError(err)
	}

	return storageMethods, nil
}
