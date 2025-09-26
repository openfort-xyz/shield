package usercontactrepo

import (
	"context"
	"errors"
	"log/slog"

	"go.openfort.xyz/shield/internal/adapters/repositories/sql"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/usercontact"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
	"gorm.io/gorm"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.UserContactRepository = &repository{}

func New(db *sql.Client) repositories.UserContactRepository {
	return &repository{
		db:     db,
		logger: logger.New("user_contact_repository"),
		parser: newParser(),
	}
}

func (r *repository) Save(ctx context.Context, contactInfo *usercontact.UserContact) error {
	r.logger.InfoContext(ctx, "saving user contact information", slog.String("external user ID", contactInfo.ExternalUserID))

	dbContactInfo := r.parser.toDatabase(contactInfo)
	err := r.db.Create(dbContactInfo).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error saving user contact information", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*usercontact.UserContact, error) {
	r.logger.InfoContext(ctx, "selecting user contact information by user ID", slog.String("external user ID", userID))

	dbContact := &UserContact{}
	err := r.db.Where("external_user_id = ?", userID).First(dbContact).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrUserContactNotFound
		}
		r.logger.ErrorContext(ctx, "error selecting user contact information by user ID", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbContact), nil
}
