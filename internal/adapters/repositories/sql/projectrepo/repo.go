package projectrepo

import (
	"context"
	"errors"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"

	"log/slog"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/adapters/repositories/sql"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
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

func (r *repository) SaveProjectRateLimits(ctx context.Context, rateLimits *project.RateLimit) error {
	r.logger.InfoContext(ctx, "saving rate limits for project", slog.String("project", rateLimits.ProjectID))

	dbRateLimits := r.parser.toDatabaseRateLimits(rateLimits)
	err := r.db.Create(dbRateLimits).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error saving project rate limits", logger.Error(err))
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
			return nil, domainErrors.ErrProjectNotFound
		}
		r.logger.ErrorContext(ctx, "error getting project", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbProj), nil
}

func (r *repository) UpdateAPISecret(ctx context.Context, projectID, encryptedSecret string) error {
	r.logger.InfoContext(ctx, "updating API secret", slog.String("project_id", projectID))

	err := r.db.Model(&Project{}).Where("id = ?", projectID).Update("api_secret", encryptedSecret).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "error updating API secret", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update2FA(ctx context.Context, projectID string, enable2FA bool) error {
	r.logger.InfoContext(ctx, "updating 2FA", slog.String("project_id", projectID), slog.Bool("enable_2fa", enable2FA))

	err := r.db.Model(&Project{}).Where("id = ?", projectID).Update("enable_2fa", enable2FA).Error

	if err != nil {
		r.logger.ErrorContext(ctx, "error updating 2FA", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetWithRateLimit(ctx context.Context, projectID string) (*project.WithRateLimit, error) {
	r.logger.InfoContext(ctx, "getting project with rate limit")

	dbProj := &ProjectWithRateLimit{}
	err := r.db.Table("shld_projects").
		Select("shld_projects.*, shld_rate_limit.email_requests_per_hour, shld_rate_limit.sms_requests_per_hour").
		Joins("LEFT JOIN shld_rate_limit ON shld_projects.id = shld_rate_limit.project_id").
		Where("shld_projects.id = ?", projectID).
		First(dbProj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrProjectNotFound
		}
		r.logger.ErrorContext(ctx, "error getting project", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomainWithRateLimit(dbProj), nil
}

func (r *repository) GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	r.logger.InfoContext(ctx, "getting project by API key")

	dbProj := &Project{}
	err := r.db.Where("api_key = ?", apiKey).First(dbProj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrProjectNotFound
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
	err := r.db.Model(&EncryptionPart{}).Where("project_id = ?", projectID).First(encryptionPart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domainErrors.ErrEncryptionPartNotFound
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

func (r *repository) CreateMigration(ctx context.Context, projectID string, success bool) error {
	r.logger.InfoContext(ctx, "creating migration", slog.String("project_id", projectID), slog.Bool("success", success))

	migration := &Migration{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Success:   success,
	}

	err := r.db.Create(migration).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating migration", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) HasSuccessfulMigration(ctx context.Context, projectID string) (bool, error) {
	r.logger.InfoContext(ctx, "checking for successful migration", slog.String("project_id", projectID))

	var count int64
	err := r.db.Model(&Migration{}).Where("project_id = ? AND success = ?", projectID, true).Count(&count).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error checking for successful migration", logger.Error(err))
		return false, err
	}

	return count > 0, nil
}
