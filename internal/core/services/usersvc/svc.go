package usersvc

import (
	"context"
	"errors"
	"log/slog"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

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

	usr, err := s.getByExternal(ctx, externalUserID, providerID)
	if err != nil && !errors.Is(err, domainErrors.ErrExternalUserNotFound) {
		s.logger.ErrorContext(ctx, "failed to get user by external", logger.Error(err))
		return nil, err
	}

	if usr == nil {
		usr, err = s.create(ctx, projectID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create user", logger.Error(err))
			return nil, err
		}

		_, err = s.createExternal(ctx, usr, externalUserID, providerID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create external user", logger.Error(err))
			return nil, err
		}
	}

	return usr, nil
}

func (s *service) create(ctx context.Context, projectID string) (*user.User, error) {
	s.logger.InfoContext(ctx, "creating user", slog.String("project_id", projectID))
	usr := &user.User{
		ProjectID: projectID,
	}

	err := s.repo.Create(ctx, usr)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create user", logger.Error(err))
		return nil, err
	}

	return usr, nil
}

func (s *service) getByExternal(ctx context.Context, externalUserID, providerID string) (*user.User, error) {
	s.logger.InfoContext(ctx, "getting user by external user", slog.String("external_user_id", externalUserID), slog.String("provider_id", providerID))

	usr, err := s.repo.FindUserByExternalID(ctx, externalUserID, providerID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user by external ID", logger.Error(err))
		return nil, err
	}

	return usr, nil
}

func (s *service) createExternal(ctx context.Context, usr *user.User, externalUserID, providerID string) (*user.ExternalUser, error) {
	s.logger.InfoContext(ctx, "creating external user")

	extUsr := &user.ExternalUser{
		UserID:         usr.ID,
		ExternalUserID: externalUserID,
		ProviderID:     providerID,
	}

	err := s.repo.CreateExternal(ctx, extUsr)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create external user", logger.Error(err))
		return nil, err
	}

	return extUsr, nil
}
