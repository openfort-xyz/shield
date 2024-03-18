package sql

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Client struct {
	*gorm.DB
}

// dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
// dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
type Config struct {
	Driver Driver `env:"DB_DRIVER" envDefault:"mysql"`
	Host   string `env:"DB_HOST" envDefault:"localhost"`
	Port   int    `env:"DB_PORT" envDefault:"3306"`
	User   string `env:"DB_USER,required"`
	Pass   string `env:"DB_PASS,required"`
	DBName string `env:"DB_NAME,required"`

	// MySQL
	Charset   string `env:"DB_MYSQL_CHARSET" envDefault:"utf8mb4"`
	ParseTime bool   `env:"DB_MYSQL_PARSE_TIME" envDefault:"True"`
	Location  string `env:"DB_MYSQL_LOCATION" envDefault:"Local"`

	// Postgres
	SSLMode  string `env:"DB_POSTGRES_SSL_MODE" envDefault:"disable"`
	TimeZone string `env:"DB_POSTGRES_TIME_ZONE" envDefault:"Europe/Madrid"`
}

func (c *Config) DSN() string {
	switch c.Driver {
	case DriverMySQL:
		return c.MySQLDSN()
	case DriverPostgres:
		return c.PostgresDSN()
	default:
		return ""
	}
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

func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, ErrMissingConfig
	}

	if cfg.Driver == "" {
		return nil, ErrMissingDriver
	}

	var dialect gorm.Dialector
	switch cfg.Driver {
	case DriverMySQL:
		sqlDB, err := sql.Open("mysql", cfg.MySQLDSN())
		if err != nil {
			return nil, err
		}

		dialect = mysql.New(mysql.Config{
			Conn: sqlDB,
		})
	case DriverPostgres:
		dialect = postgres.Open(cfg.PostgresDSN())
	default:
		return nil, ErrDriverNotSupported
	}

	db, err := gorm.Open(dialect)
	if err != nil {
		return nil, err
	}

	return &Client{db}, nil
}
