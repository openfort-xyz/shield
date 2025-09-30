package projectsvc

import (
	"context"
	"errors"
	"log/slog"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type service struct {
	repo   repositories.ProjectRepository
	logger *slog.Logger
	cost   int
}

var _ services.ProjectService = (*service)(nil)

func New(repo repositories.ProjectRepository) services.ProjectService {
	return &service{
		repo:   repo,
		logger: logger.New("project_service"),
		cost:   bcrypt.DefaultCost,
	}
}

func (s *service) Create(ctx context.Context, name string, enable2fa bool) (*project.Project, error) {
	s.logger.InfoContext(ctx, "creating project", slog.String("name", name))
	apiSecret := uuid.NewString()
	encryptedSecret, err := bcrypt.GenerateFromPassword([]byte(apiSecret), s.cost)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt secret", logger.Error(err))
		return nil, err
	}

	proj := &project.Project{
		Name:      name,
		APIKey:    uuid.NewString(),
		APISecret: string(encryptedSecret),
		Enable2FA: enable2fa,
	}

	err = s.repo.Create(ctx, proj)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create project", logger.Error(err))
		return nil, err
	}

	proj.APISecret = apiSecret
	return proj, nil
}

func (s *service) SaveProjectRateLimits(ctx context.Context, projectID string, smsRateLimit int64, emailRateLimit int64) error {
	s.logger.InfoContext(ctx, "save project rate limits", slog.String("project", projectID))

	projectRateLimit := &project.RateLimit{
		ProjectID:            projectID,
		SMSRequestsPerHour:   smsRateLimit,
		EmailRequestsPerHour: emailRateLimit,
	}

	err := s.repo.SaveProjectRateLimits(ctx, projectRateLimit)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to save project rate limits", logger.Error(err))
	}

	return err
}

func (s *service) SetEncryptionPart(ctx context.Context, projectID, part string) error {
	s.logger.InfoContext(ctx, "setting encryption part", slog.String("project_id", projectID))
	ep, err := s.repo.GetEncryptionPart(ctx, projectID)
	if err != nil && !errors.Is(err, domainErrors.ErrEncryptionPartNotFound) {
		s.logger.ErrorContext(ctx, "failed to get encryption part", logger.Error(err))
		return err
	}

	if ep != "" {
		s.logger.Warn("encryption part already exists", slog.String("project_id", projectID))
		return domainErrors.ErrEncryptionPartAlreadyExists
	}

	err = s.repo.SetEncryptionPart(ctx, projectID, part)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to set encryption part", logger.Error(err))
		return err
	}

	return nil
}
