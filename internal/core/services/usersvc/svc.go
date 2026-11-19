package usersvc

import (
	"context"
	"log/slog"

	"github.com/openfort-xyz/shield/internal/core/domain/user"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/logger"
)

type service struct {
	repo   repositories.UserRepository
	logger *slog.Logger
}

var _ services.UserService = (*service)(nil)

func New(repo repositories.UserRepository) services.UserService {
	return &service{
		repo:   repo,
		logger: logger.New("user_service"),
	}
}

func (s *service) GetOrCreate(ctx context.Context, projectID, externalUserID, providerID string) (*user.User, error) {
	s.logger.InfoContext(ctx, "getting or creating user", slog.String("project_id", projectID), slog.String("external_user_id", externalUserID), slog.String("provider_id", providerID))

	usr, err := s.repo.GetOrCreate(ctx, projectID, externalUserID, providerID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get or create user", logger.Error(err))
		return nil, err
	}

	return usr, nil
}
