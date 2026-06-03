package sql

import (
	"fmt"
	"net/url"
	"strconv"

	env "github.com/caarlos0/env/v10"
)

type Config struct {
	Host   string `env:"DB_HOST" envDefault:"localhost"`
	Port   int    `env:"DB_PORT" envDefault:"5432"`
	User   string `env:"DB_USER" envDefault:"postgres"`
	Pass   string `env:"DB_PASS" envDefault:"password"`
	DBName string `env:"DB_NAME" envDefault:"shield"`

	// SSLMode controls the libpq sslmode parameter.
	// Valid values: disable, allow, prefer, require, verify-ca, verify-full.
	SSLMode string `env:"DB_SSL_MODE" envDefault:"disable"`

	SSLRootCert string `env:"DB_SSL_ROOT_CERT"` // Path to server-ca.pem
	SSLCert     string `env:"DB_SSL_CERT"`      // Path to client-cert.pem
	SSLKey      string `env:"DB_SSL_KEY"`       // Path to client-key.pem

	TimeZone string `env:"DB_TIMEZONE" envDefault:"UTC"`
}

const migrationDirectory = "internal/adapters/repositories/sql/migrations"

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) postgresDSN() (string, error) {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}

	q := url.Values{}
	q.Set("sslmode", c.SSLMode)
	if c.TimeZone != "" {
		q.Set("TimeZone", c.TimeZone)
	}
	if c.SSLRootCert != "" {
		q.Set("sslrootcert", c.SSLRootCert)
	}
	if c.SSLCert != "" {
		q.Set("sslcert", c.SSLCert)
	}
	if c.SSLKey != "" {
		q.Set("sslkey", c.SSLKey)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Pass),
		Host:     c.Host + ":" + strconv.Itoa(c.Port),
		Path:     "/" + c.DBName,
		RawQuery: q.Encode(),
	}

	if c.Host == "" || c.DBName == "" {
		return "", fmt.Errorf("invalid postgres config: host and dbname are required")
	}

	return u.String(), nil
}
