package sharesvc

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/oflog"
)

type service struct {
	repo   repositories.ShareRepository
	logger *slog.Logger
}

var _ services.ShareService = (*service)(nil)

func New(repo repositories.ShareRepository) services.ShareService {
	return &service{
		repo:   repo,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("share_service"),
	}
}

func (s *service) Create(ctx context.Context, shr *share.Share) error {
	s.logger.InfoContext(ctx, "creating share", slog.String("user_id", shr.UserID))

	shrRepo, err := s.repo.GetByUserID(ctx, shr.UserID)
	if err != nil && !errors.Is(err, domain.ErrShareNotFound) {
		s.logger.ErrorContext(ctx, "failed to get share", slog.String("error", err.Error()))
		return err
	}

	if shrRepo != nil {
		s.logger.ErrorContext(ctx, "share already exists", slog.String("user_id", shr.UserID))
		return domain.ErrShareAlreadyExists
	}

	err = s.repo.Create(ctx, shr)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create share", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (s *service) GetByUserID(ctx context.Context, userID string) (*share.Share, error) {
	s.logger.InfoContext(ctx, "getting share by user", slog.String("user_id", userID))
	shr, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get share", slog.String("error", err.Error()))
		return nil, err
	}

	return shr, nil
}
