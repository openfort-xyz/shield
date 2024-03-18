package userrepo

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/user"
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

var _ repositories.UserRepository = (*repository)(nil)

func New(db *sql.Client) repositories.UserRepository {
	return &repository{
		db:     db,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("user_repository"),
		parser: newParser(),
	}
}

func (r *repository) Create(ctx context.Context, usr *user.User) error {
	r.logger.InfoContext(ctx, "creating user")

	dbUsr := r.parser.toDatabase(usr)
	err := r.db.Create(dbUsr).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating user", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (r *repository) Get(ctx context.Context, userID string) (*user.User, error) {
	r.logger.InfoContext(ctx, "getting user", slog.String("user_id", userID))

	var dbUsr *User
	err := r.db.Where("id = ?", userID).First(dbUsr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrUserNotFound
		}
		r.logger.ErrorContext(ctx, "error getting user", slog.String("error", err.Error()))
		return nil, err
	}

	return r.parser.toDomain(dbUsr), nil
}

func (r *repository) CreateExternal(ctx context.Context, extUsr *user.ExternalUser) error {
	r.logger.InfoContext(ctx, "creating external user")

	dbExtUsr := r.parser.toDatabaseExternalUser(extUsr)
	err := r.db.Create(dbExtUsr).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error creating external user", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (r *repository) FindExternalBy(ctx context.Context, opts ...repositories.Option) ([]*user.ExternalUser, error) {
	r.logger.InfoContext(ctx, "finding external user")

	options := &options{
		query: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(options)
	}

	var dbExtUsrs []*ExternalUser
	err := r.db.Where(options.query).Find(&dbExtUsrs).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "error finding external user", slog.String("error", err.Error()))
		return nil, err
	}

	extUsrs := make([]*user.ExternalUser, len(dbExtUsrs))
	for i, dbExtUsr := range dbExtUsrs {
		extUsrs[i] = r.parser.toDomainExternalUser(dbExtUsr)
	}

	return extUsrs, nil
}
