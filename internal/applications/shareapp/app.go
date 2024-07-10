package shareapp

import (
	"context"
	"fmt"
	"go.openfort.xyz/shield/internal/core/ports/factories"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/logger"
)

type ShareApplication struct {
	shareSvc          services.ShareService
	shareRepo         repositories.ShareRepository
	projectRepo       repositories.ProjectRepository
	logger            *slog.Logger
	encryptionFactory factories.EncryptionFactory
}

func New(shareSvc services.ShareService, shareRepo repositories.ShareRepository, projectRepo repositories.ProjectRepository, encryptionFactory factories.EncryptionFactory) *ShareApplication {
	return &ShareApplication{
		shareSvc:          shareSvc,
		shareRepo:         shareRepo,
		projectRepo:       projectRepo,
		logger:            logger.New("share_application"),
		encryptionFactory: encryptionFactory,
	}
}

func (a *ShareApplication) RegisterShare(ctx context.Context, shr *share.Share, opts ...Option) error {
	a.logger.InfoContext(ctx, "registering share")
	usrID := contexter.GetUserID(ctx)
	projID := contexter.GetProjectID(ctx)
	shr.UserID = usrID

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

	err := a.shareSvc.Create(ctx, shr, shrOpts...)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create share", logger.Error(err))
		return fromDomainError(err)
	}

	return nil
}

func (a *ShareApplication) UpdateShare(ctx context.Context, shr *share.Share, opts ...Option) (*share.Share, error) {
	a.logger.InfoContext(ctx, "updating share")
	usrID := contexter.GetUserID(ctx)
	projID := contexter.GetProjectID(ctx)

	dbShare, err := a.shareRepo.GetByUserID(ctx, usrID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
		return nil, fromDomainError(err)
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

func (a *ShareApplication) GetShare(ctx context.Context, opts ...Option) (*share.Share, error) {
	a.logger.InfoContext(ctx, "getting share")
	usrID := contexter.GetUserID(ctx)
	projID := contexter.GetProjectID(ctx)

	shr, err := a.shareRepo.GetByUserID(ctx, usrID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
		return nil, fromDomainError(err)
	}

	fmt.Println(shr)

	var opt options
	for _, o := range opts {
		o(&opt)
	}

	if shr.RequiresEncryption() {
		encryptionKey, err := a.reconstructEncryptionKey(ctx, projID, opt)
		if err != nil {
			return nil, err
		}
		fmt.Println(encryptionKey)

		cypher := a.encryptionFactory.CreateEncryptionStrategy(encryptionKey)
		shr.Secret, err = cypher.Decrypt(shr.Secret)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to decrypt secret", logger.Error(err))
			return nil, ErrInternal
		}
	}

	return shr, nil
}

func (a *ShareApplication) DeleteShare(ctx context.Context) error {
	a.logger.InfoContext(ctx, "deleting share")
	usrID := contexter.GetUserID(ctx)

	shr, err := a.shareRepo.GetByUserID(ctx, usrID)
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

	builder, err := a.encryptionFactory.CreateEncryptionKeyBuilder(builderType)
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

	return encryptionKey, nil
}
