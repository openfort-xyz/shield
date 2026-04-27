package sql

import (
	"path/filepath"

	"github.com/pressly/goose"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Client struct {
	*gorm.DB
}

func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, ErrMissingConfig
	}

	dsn, err := cfg.postgresDSN()
	if err != nil {
		return nil, err
	}

	dialector := postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Client{db}, nil
}

func (c *Client) Migrate() error {
	migrationDir, err := filepath.Abs(migrationDirectory)
	if err != nil {
		return err
	}

	if err := goose.SetDialect(c.Name()); err != nil {
		return err
	}

	db, err := c.DB.DB()
	if err != nil {
		return err
	}

	return goose.Run("up", db, migrationDir)
}

func (c *Client) Rollback() error {
	migrationDir, err := filepath.Abs(migrationDirectory)
	if err != nil {
		return err
	}

	if err := goose.SetDialect(c.Name()); err != nil {
		return err
	}

	db, err := c.DB.DB()
	if err != nil {
		return err
	}

	return goose.Run("down", db, migrationDir)
}

func CreateMigration(name string) error {
	migrationDir, err := filepath.Abs(migrationDirectory)
	if err != nil {
		return err
	}

	return goose.Run("create", nil, migrationDir, name, "sql")
}

func (c *Client) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
