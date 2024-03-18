package projectrepo

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
}

var _ repositories.ProjectRepository = &repository{}

func New(db *sql.Client) repositories.ProjectRepository {
	return &repository{
		db:     db,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("project_repository"),
	}
}

func (r *repository) Create(ctx context.Context, project *project.Project) error {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Get(ctx context.Context, projectID string) (*project.Project, error) {
	//TODO implement me
	panic("implement me")
}
