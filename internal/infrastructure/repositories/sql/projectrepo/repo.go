package projectrepo

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
	"go.openfort.xyz/shield/pkg/oflog"
	"gorm.io/gorm"
	"log/slog"
	"os"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.ProjectRepository = &repository{}

func New(db *sql.Client) repositories.ProjectRepository {
	return &repository{
		db:     db,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("project_repository"),
		parser: newParser(),
	}
}

func (r *repository) Create(ctx context.Context, proj *project.Project) error {
	r.logger.InfoContext(ctx, "creating project", slog.String("name", proj.Name))
	if proj.ID == "" {
		proj.ID = uuid.NewString()
	}

	dbProj := r.parser.toDatabase(proj)
	err := r.db.Create(dbProj).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating project", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (r *repository) Get(ctx context.Context, projectID string) (*project.Project, error) {
	r.logger.InfoContext(ctx, "getting project")

	dbProj := &Project{}
	err := r.db.Where("id = ?", projectID).First(dbProj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProjectNotFound
		}
		r.logger.ErrorContext(ctx, "error getting project", slog.String("error", err.Error()))
		return nil, err
	}

	return r.parser.toDomain(dbProj), nil
}

func (r *repository) GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	r.logger.InfoContext(ctx, "getting project by API key")

	dbProj := &Project{}
	err := r.db.Where("api_key = ?", apiKey).First(dbProj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProjectNotFound
		}
		r.logger.ErrorContext(ctx, "error getting project", slog.String("error", err.Error()))
		return nil, err
	}

	return r.parser.toDomain(dbProj), nil
}
