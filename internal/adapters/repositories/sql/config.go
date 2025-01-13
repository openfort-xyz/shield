package sql

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/caarlos0/env/v10"
	"github.com/go-sql-driver/mysql"
)

type Config struct {
	Host   string `env:"DB_HOST" envDefault:"localhost"`
	Port   int    `env:"DB_PORT" envDefault:"3306"`
	User   string `env:"DB_USER" envDefault:"root"`
	Pass   string `env:"DB_PASS" envDefault:"password"`
	DBName string `env:"DB_NAME" envDefault:"shield"`

	Charset   string `env:"DB_MYSQL_CHARSET" envDefault:"utf8mb4"`
	ParseTime bool   `env:"DB_MYSQL_PARSE_TIME" envDefault:"True"`
	Location  string `env:"DB_MYSQL_LOCATION" envDefault:"Local"`

	SSLRootCert string `env:"DB_SSL_ROOT_CERT"` // Path to server-ca.pem
	SSLCert     string `env:"DB_SSL_CERT"`      // Path to client-cert.pem
	SSLKey      string `env:"DB_SSL_KEY"`       // Path to client-key.pem

	SSLSkipVerify bool `env:"DB_SSL_SKIP_VERIFY" envDefault:"False"`
}

const migrationDirectory = "internal/adapters/repositories/sql/migrations"

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) mysqlDSN() (string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Pass, c.Host, c.Port, c.DBName)

	if c.SSLRootCert != "" && c.SSLCert != "" && c.SSLKey != "" {
		dsn = fmt.Sprintf("%s&tls=custom", dsn)
		if err := c.registerTLSConfig(); err != nil {
			return "", err
		}
	}

	return dsn, nil
}

func (c *Config) registerTLSConfig() error {
	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(c.SSLRootCert)
	if err != nil {
		return err
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("failed to append PEM")
	}

	certs, err := tls.LoadX509KeyPair(c.SSLCert, c.SSLKey)
	if err != nil {
		return err
	}

	return mysql.RegisterTLSConfig("custom", &tls.Config{
		InsecureSkipVerify: c.SSLSkipVerify, // nolint:gosec
		RootCAs:            rootCertPool,
		Certificates:       []tls.Certificate{certs},
		MinVersion:         tls.VersionTLS12,
	})
}
