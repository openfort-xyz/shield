package shareapp

import (
	"context"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/cypher"
	"go.openfort.xyz/shield/pkg/logger"
)

type ShareApplication struct {
	shareSvc    services.ShareService
	shareRepo   repositories.ShareRepository
	projectRepo repositories.ProjectRepository
	logger      *slog.Logger
}

func New(shareSvc services.ShareService, shareRepo repositories.ShareRepository, projectRepo repositories.ProjectRepository) *ShareApplication {
	return &ShareApplication{
		shareSvc:    shareSvc,
		shareRepo:   shareRepo,
		projectRepo: projectRepo,
		logger:      logger.New("share_application"),
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
		if opt.encryptionPart == nil {
			return ErrEncryptionPartRequired
		}

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

func (a *ShareApplication) GetShare(ctx context.Context, opts ...Option) (*share.Share, error) {
	a.logger.InfoContext(ctx, "getting share")
	usrID := contexter.GetUserID(ctx)
	projID := contexter.GetProjectID(ctx)

	shr, err := a.shareRepo.GetByUserID(ctx, usrID)
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

		shr.Secret, err = cypher.Decrypt(shr.Secret, encryptionKey)
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to decrypt secret", logger.Error(err))
			return nil, ErrInternal
		}
	}

	return shr, nil
}

func (a *ShareApplication) reconstructEncryptionKey(ctx context.Context, projID string, opt options) (string, error) {
	if opt.encryptionPart == nil || *opt.encryptionPart == "" {
		return "", ErrEncryptionPartRequired
	}

	storedPart, err := a.projectRepo.GetEncryptionPart(ctx, projID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get encryption part", logger.Error(err))
		return "", fromDomainError(err)
	}

	encryptionKey, err := cypher.ReconstructEncryptionKey(storedPart, *opt.encryptionPart)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to reconstruct encryption key", logger.Error(err))
		return "", ErrInvalidEncryptionPart
	}
	return encryptionKey, nil
}
