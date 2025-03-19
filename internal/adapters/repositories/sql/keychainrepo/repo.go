package keychainrepo

import (
	"context"
	"errors"
	"log/slog"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/keychain"
	"gorm.io/gorm"

	"go.openfort.xyz/shield/internal/adapters/repositories/sql"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
)

type repository struct {
	db     *sql.Client
	logger *slog.Logger
	parser *parser
}

var _ repositories.KeychainRepository = (*repository)(nil)

func New(db *sql.Client) repositories.KeychainRepository {
	return &repository{
		db:     db,
		logger: logger.New("keychain_repository"),
		parser: newParser(),
	}
}

func (r *repository) Create(ctx context.Context, keychain *keychain.Keychain) error {
	r.logger.InfoContext(ctx, "creating keychain")

	dbKeychain := r.parser.toDatabase(keychain)
	err := r.db.Create(dbKeychain).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating keychain", logger.Error(err))
		return err
	}

	return nil
}

func (r *repository) Get(ctx context.Context, keychainID string) (*keychain.Keychain, error) {
	r.logger.InfoContext(ctx, "getting keychain", slog.String("id", keychainID))

	dbKeychain := &Keychain{}
	err := r.db.Where("id = ?", keychainID).First(dbKeychain).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrKeychainNotFound
		}
		r.logger.ErrorContext(ctx, "error getting keychain", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbKeychain), nil
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*keychain.Keychain, error) {
	r.logger.InfoContext(ctx, "getting keychain", slog.String("user_id", userID))

	dbKeychain := &Keychain{}
	err := r.db.Where("user_id = ?", userID).First(dbKeychain).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrKeychainNotFound
		}
		r.logger.ErrorContext(ctx, "error getting keychain", logger.Error(err))
		return nil, err
	}

	return r.parser.toDomain(dbKeychain), nil
}

func (r *repository) Delete(ctx context.Context, keychainID string) error {
	r.logger.InfoContext(ctx, "deleting keychain", slog.String("id", keychainID))

	err := r.db.Delete(&Keychain{}, keychainID).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error deleting keychain", logger.Error(err))
		return err
	}

	return nil
}
