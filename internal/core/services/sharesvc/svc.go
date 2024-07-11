package sharesvc

import (
	"context"
	"errors"
	"log/slog"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"

	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/logger"
)

type service struct {
	repo              repositories.ShareRepository
	logger            *slog.Logger
	encryptionFactory factories.EncryptionFactory
}

var _ services.ShareService = (*service)(nil)

func New(repo repositories.ShareRepository, encryptionFactory factories.EncryptionFactory) services.ShareService {
	return &service{
		repo:              repo,
		logger:            logger.New("share_service"),
		encryptionFactory: encryptionFactory,
	}
}

func (s *service) Create(ctx context.Context, shr *share.Share, opts ...services.ShareOption) error {
	s.logger.InfoContext(ctx, "creating share", slog.String("user_id", shr.UserID))

	shrRepo, err := s.repo.GetByUserID(ctx, shr.UserID)
	if err != nil && !errors.Is(err, domainErrors.ErrShareNotFound) {
		s.logger.ErrorContext(ctx, "failed to get share", logger.Error(err))
		return err
	}

	if shrRepo != nil {
		s.logger.ErrorContext(ctx, "share already exists", slog.String("user_id", shr.UserID))
		return domainErrors.ErrShareAlreadyExists
	}

	var o services.ShareOptions
	for _, opt := range opts {
		opt(&o)
	}

	if shr.RequiresEncryption() {
		if o.EncryptionKey == nil {
			return domainErrors.ErrEncryptionPartRequired
		}

		cypher := s.encryptionFactory.CreateEncryptionStrategy(*o.EncryptionKey)
		shr.Secret, err = cypher.Encrypt(shr.Secret)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to encrypt secret", logger.Error(err))
			return err
		}
	}

	err = s.repo.Create(ctx, shr)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create share", logger.Error(err))
		return err
	}

	return nil
}
