package projectrepo

import (
	"context"
	"errors"

	"log/slog"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
	"go.openfort.xyz/shield/pkg/logger"
	"gorm.io/gorm"
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
		logger: logger.New("project_repository"),
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
		r.logger.ErrorContext(ctx, "error creating project", logger.Error(err))
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
		r.logger.ErrorContext(ctx, "error getting project", logger.Error(err))
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
		r.logger.ErrorContext(ctx, "error getting project", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbProj), nil
}

func (r *repository) Delete(ctx context.Context, projectID string) error {
	r.logger.InfoContext(ctx, "deleting project")

	err := r.db.Delete(&Project{}, "id = ?", projectID).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error deleting project", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetEncryptionPart(ctx context.Context, projectID string) (string, error) {
	r.logger.InfoContext(ctx, "getting encryption part")

	encryptionPart := &EncryptionPart{}
	err := r.db.Where("project_id = ?", projectID).First(encryptionPart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domain.ErrEncryptionPartNotFound
		}
		r.logger.ErrorContext(ctx, "error getting encryption part", logger.Error(err))
		return "", err
	}

	return encryptionPart.Part, nil
}

func (r *repository) SetEncryptionPart(ctx context.Context, projectID, part string) error {
	r.logger.InfoContext(ctx, "setting encryption part")

	encryptionPart := &EncryptionPart{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Part:      part,
	}

	err := r.db.Create(encryptionPart).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error setting encryption part", logger.Error(err))
		return err
	}

	return nil
}
