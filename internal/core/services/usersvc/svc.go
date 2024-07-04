package usersvc

import (
	"context"
	"errors"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain/user"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/logger"
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

		_, err = s.createExternal(ctx, projectID, usr.ID, externalUserID, providerID)
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

	extUsrs, err := s.repo.FindExternalBy(ctx, s.repo.WithExternalUserID(externalUserID), s.repo.WithProviderID(providerID))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get external user", logger.Error(err))
		return nil, err
	}

	if len(extUsrs) == 0 {
		s.logger.ErrorContext(ctx, "external user not found", slog.String("external_user_id", externalUserID), slog.String("provider_id", providerID))
		return nil, domainErrors.ErrExternalUserNotFound
	}

	extUsr := extUsrs[0]
	usr, err := s.repo.Get(ctx, extUsr.UserID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user", logger.Error(err))
		return nil, err
	}

	return usr, nil
}

func (s *service) createExternal(ctx context.Context, projectID, userID, externalUserID, providerID string) (*user.ExternalUser, error) {
	s.logger.InfoContext(ctx, "creating external user", slog.String("project_id", projectID))

	usr, err := s.repo.Get(ctx, userID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user", logger.Error(err))
		return nil, err
	}

	if usr == nil {
		s.logger.ErrorContext(ctx, "user not found", slog.String("user_id", userID))
		return nil, domainErrors.ErrUserNotFound
	}

	if usr.ProjectID != projectID {
		s.logger.ErrorContext(ctx, "user does not belong to project", slog.String("project_id", projectID), slog.String("user_id", userID))
		return nil, domainErrors.ErrUserNotFound
	}

	extUsrs, err := s.repo.FindExternalBy(ctx, s.repo.WithUserID(userID), s.repo.WithProviderID(providerID))
	if err != nil && !errors.Is(err, domainErrors.ErrExternalUserNotFound) {
		s.logger.ErrorContext(ctx, "failed to get external user", logger.Error(err))
		return nil, err
	}

	if len(extUsrs) != 0 {
		s.logger.ErrorContext(ctx, "external user already exists for this user and provider", slog.String("user_id", userID), slog.String("provider_type", providerID))
		return nil, domainErrors.ErrExternalUserAlreadyExists
	}

	extUsr := &user.ExternalUser{
		UserID:         userID,
		ExternalUserID: externalUserID,
		ProviderID:     providerID,
	}

	err = s.repo.CreateExternal(ctx, extUsr)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create external user", logger.Error(err))
		return nil, err
	}

	return extUsr, nil
}
