package encryptionpartsrepo

import (
	"context"
	"errors"
	"github.com/tidwall/buntdb"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/bunt"
	"go.openfort.xyz/shield/pkg/logger"
	"log/slog"
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

func (r *repository) Get(ctx context.Context, sessionId string) (string, error) {
	var part string
	err := r.db.View(func(tx *buntdb.Tx) error {
		var err error
		part, err = tx.Get(sessionId)
		return err
	})
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return "", domain.ErrEncryptionPartNotFound
		}
		r.logger.ErrorContext(ctx, "error getting encryption part", logger.Error(err))
		return "", err
	}

	if part == "" {
		return "", domain.ErrEncryptionPartNotFound
	}

	return part, nil
}

func (r *repository) Set(ctx context.Context, sessionId, part string) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(sessionId, part, nil)
		if errors.Is(err, buntdb.ErrIndexExists) {
			return domain.ErrEncryptionPartAlreadyExists
		}
		r.logger.ErrorContext(ctx, "error setting encryption part", logger.Error(err))
		return err
	})
}

func (r *repository) Delete(ctx context.Context, sessionId string) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(sessionId)
		if errors.Is(err, buntdb.ErrNotFound) {
			return domain.ErrEncryptionPartNotFound
		}
		r.logger.ErrorContext(ctx, "error deleting encryption part", logger.Error(err))
		return err
	})
}
