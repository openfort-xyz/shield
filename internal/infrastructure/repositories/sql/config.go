package sql

import (
	"fmt"
	"github.com/caarlos0/env/v10"
)

type Config struct {
	Driver Driver `env:"DB_DRIVER" envDefault:"mysql"`
	Host   string `env:"DB_HOST" envDefault:"localhost"`
	Port   int    `env:"DB_PORT" envDefault:"3306"`
	User   string `env:"DB_USER" envDefault:"user"`
	Pass   string `env:"DB_PASS" envDefault:"password"`
	DBName string `env:"DB_NAME" envDefault:"shield"`

	// MySQL
	Charset   string `env:"DB_MYSQL_CHARSET" envDefault:"utf8mb4"`
	ParseTime bool   `env:"DB_MYSQL_PARSE_TIME" envDefault:"True"`
	Location  string `env:"DB_MYSQL_LOCATION" envDefault:"Local"`

	// Postgres
	SSLMode  string `env:"DB_POSTGRES_SSL_MODE" envDefault:"disable"`
	TimeZone string `env:"DB_POSTGRES_TIME_ZONE" envDefault:"Europe/Madrid"`
}

const migrationDirectory = "internal/infrastructure/repositories/sql/migrations"

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.User, c.Pass, c.Host, c.Port, c.DBName, c.Charset, c.ParseTime, c.Location)
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.Host, c.User, c.Pass, c.DBName, c.Port, c.SSLMode, c.TimeZone)
}

type Driver string

const (
	DriverMySQL    Driver = "mysql"
	DriverPostgres Driver = "postgres"
)
