package sql

import (
	"database/sql"
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

	switch cfg.Driver {
	case DriverMySQL:
		return newMySQL(cfg)
	case DriverPostgres:
		return newPostgres(cfg)
	default:
		return nil, ErrDriverNotSupported
	}
}

func newMySQL(cfg *Config) (*Client, error) {
	sqlDB, err := sql.Open("mysql", cfg.MySQLDSN())
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Client{db}, nil
}

func newPostgres(cfg *Config) (*Client, error) {
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Client{db}, nil
}
