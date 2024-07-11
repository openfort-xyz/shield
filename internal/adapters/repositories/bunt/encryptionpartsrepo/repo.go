package encryptionpartsrepo

import (
	"context"
	"errors"
	"log/slog"

	"github.com/tidwall/buntdb"
	"go.openfort.xyz/shield/internal/adapters/repositories/bunt"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/pkg/logger"
)

type repository struct {
	db     *bunt.Client
	logger *slog.Logger
}

var _ repositories.EncryptionPartsRepository = &repository{}

func New(db *bunt.Client) repositories.EncryptionPartsRepository {
	return &repository{
		db:     db,
		logger: logger.New("encryption_parts_repository"),
	}
}

func (r *repository) Get(ctx context.Context, sessionID string) (string, error) {
	var part string
	err := r.db.View(func(tx *buntdb.Tx) error {
		var err error
		part, err = tx.Get(sessionID)
		return err
	})
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return "", domainErrors.ErrEncryptionPartNotFound
		}
		r.logger.ErrorContext(ctx, "error getting encryption part", logger.Error(err))
		return "", err
	}

	if part == "" {
		return "", domainErrors.ErrEncryptionPartNotFound
	}

	return part, nil
}

func (r *repository) Set(ctx context.Context, sessionID, part string) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(sessionID, part, nil)
		if err != nil {
			if errors.Is(err, buntdb.ErrIndexExists) {
				return domainErrors.ErrEncryptionPartAlreadyExists
			}
			r.logger.ErrorContext(ctx, "error setting encryption part", logger.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) Delete(ctx context.Context, sessionID string) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(sessionID)
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				return domainErrors.ErrEncryptionPartNotFound
			}
			r.logger.ErrorContext(ctx, "error deleting encryption part", logger.Error(err))
		}
		return err
	})
}
