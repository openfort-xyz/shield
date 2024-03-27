package sql

import (
	"context"
	"fmt"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/caarlos0/env/v10"
	"github.com/go-sql-driver/mysql"
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

	// CloudSQL
	InstanceConnectionName string `env:"INSTANCE_CONNECTION_NAME"`
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

func (c *Config) CloudSQLDSN() (string, error) {
	d, err := cloudsqlconn.NewDialer(context.Background())
	if err != nil {
		return "", fmt.Errorf("cloudsqlconn.NewDialer: %w", err)
	}
	var opts []cloudsqlconn.DialOption
	mysql.RegisterDialContext("cloudsqlconn",
		func(ctx context.Context, addr string) (net.Conn, error) { // nolint
			return d.Dial(ctx, c.InstanceConnectionName, opts...)
		})

	return fmt.Sprintf("%s:%s@cloudsqlconn(localhost:3306)/%s?parseTime=true",
		c.User, c.Pass, c.DBName), nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.Host, c.User, c.Pass, c.DBName, c.Port, c.SSLMode, c.TimeZone)
}

type Driver string

const (
	DriverMySQL    Driver = "mysql"
	DriverCloudSQL Driver = "cloudsql"
	DriverPostgres Driver = "postgres"
)
