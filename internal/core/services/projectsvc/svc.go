package projectsvc

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/oflog"
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
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("project_service"),
		cost:   bcrypt.DefaultCost,
	}
}

func (s *service) Create(ctx context.Context, name string) (*project.Project, error) {
	s.logger.InfoContext(ctx, "creating project", slog.String("name", name))
	apiSecret := uuid.NewString()
	encryptedSecret, err := bcrypt.GenerateFromPassword([]byte(apiSecret), s.cost)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt secret", slog.String("error", err.Error()))
		return nil, err
	}

	proj := &project.Project{
		Name:      name,
		APIKey:    uuid.NewString(),
		APISecret: string(encryptedSecret),
	}

	err = s.repo.Create(ctx, proj)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create project", slog.String("error", err.Error()))
		return nil, err
	}

	proj.APISecret = apiSecret
	return proj, nil
}

func (s *service) Get(ctx context.Context, projectID string) (*project.Project, error) {
	s.logger.InfoContext(ctx, "getting project", slog.String("project_id", projectID))
	proj, err := s.repo.Get(ctx, projectID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get project", slog.String("error", err.Error()))
		return nil, err
	}

	return proj, nil
}

func (s *service) GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	s.logger.InfoContext(ctx, "getting project by API key", slog.String("api_key", apiKey))
	proj, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get project by API key", slog.String("error", err.Error()))
		return nil, err
	}

	return proj, nil
}
