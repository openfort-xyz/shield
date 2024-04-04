package sharesvc

import (
	"context"
	"errors"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/cypher"
	"go.openfort.xyz/shield/pkg/logger"
)

type service struct {
	repo   repositories.ShareRepository
	logger *slog.Logger
}

var _ services.ShareService = (*service)(nil)

func New(repo repositories.ShareRepository) services.ShareService {
	return &service{
		repo:   repo,
		logger: logger.New("share_service"),
	}
}

func (s *service) Create(ctx context.Context, shr *share.Share, opts ...services.ShareOption) error {
	s.logger.InfoContext(ctx, "creating share", slog.String("user_id", shr.UserID))

	shrRepo, err := s.repo.GetByUserID(ctx, shr.UserID)
	if err != nil && !errors.Is(err, domain.ErrShareNotFound) {
		s.logger.ErrorContext(ctx, "failed to get share", logger.Error(err))
		return err
	}

	if shrRepo != nil {
		s.logger.ErrorContext(ctx, "share already exists", slog.String("user_id", shr.UserID))
		return domain.ErrShareAlreadyExists
	}

	var o services.ShareOptions
	for _, opt := range opts {
		opt(&o)
	}

	if shr.RequiresEncryption() {
		if o.EncryptionKey == nil {
			return domain.ErrEncryptionPartRequired
		}

		shr.Secret, err = cypher.Encrypt(shr.Secret, *o.EncryptionKey)
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
