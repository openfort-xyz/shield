package sql

import (
	"database/sql"
	"path/filepath"

	"github.com/pressly/goose"
	"gorm.io/driver/mysql"
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

	if cfg.Driver == "" {
		return nil, ErrMissingDriver
	}

	var dialector gorm.Dialector
	var err error
	switch cfg.Driver {
	case DriverMySQL:
		dialector, err = newMySQL(cfg)
	case DriverCloudSQL:
		dialector, err = newCloudSQL(cfg)
	case DriverPostgres:
		dialector = newPostgres(cfg)
	default:
		return nil, ErrDriverNotSupported
	}
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Client{db}, nil
}

func newMySQL(cfg *Config) (gorm.Dialector, error) {
	sqlDB, err := sql.Open("mysql", cfg.MySQLDSN())
	if err != nil {
		return nil, err
	}
	sqlDB.SetConnMaxLifetime(cfg.MaxConnLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.MaxConnIdleTime)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	return mysql.New(mysql.Config{
		Conn: sqlDB,
	}), nil
}

func newCloudSQL(cfg *Config) (gorm.Dialector, error) {
	dsn := cfg.CloudSQLDSN()
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetConnMaxLifetime(cfg.MaxConnLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.MaxConnIdleTime)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	return mysql.New(mysql.Config{
		Conn: sqlDB,
	}), nil
}

func newPostgres(cfg *Config) gorm.Dialector {
	return postgres.Open(cfg.PostgresDSN())
}

func (c *Client) Migrate() error {
	migrationDir, err := filepath.Abs(migrationDirectory)
	if err != nil {
		return err
	}

	if err := goose.SetDialect(c.DB.Dialector.Name()); err != nil {
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

	if err := goose.SetDialect(c.DB.Dialector.Name()); err != nil {
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
