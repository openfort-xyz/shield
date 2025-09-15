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

func (r *repository) Get(ctx context.Context, key string) (string, error) {
	var data string
	err := r.db.View(func(tx *buntdb.Tx) error {
		var err error
		data, err = tx.Get(key)
		return err
	})
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return "", domainErrors.ErrDataInDBNotFound
		}
		r.logger.ErrorContext(ctx, "error getting value by key", logger.Error(err))
		return "", err
	}

	if data == "" {
		return "", domainErrors.ErrDataInDBNotFound
	}

	return data, nil
}

func (r *repository) Set(ctx context.Context, key, value string, options *buntdb.SetOptions) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, value, options)
		if err != nil {
			if errors.Is(err, buntdb.ErrIndexExists) {
				return domainErrors.ErrKeyInDBAlreadyExists
			}
			r.logger.ErrorContext(ctx, "error setting value by key into buntdb", logger.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) Update(ctx context.Context, key, value string, options *buntdb.SetOptions) error {
	return r.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, value, options)
		if err != nil && !errors.Is(err, buntdb.ErrIndexExists) {
			r.logger.ErrorContext(ctx, "error setting value by key into buntdb", logger.Error(err))
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
