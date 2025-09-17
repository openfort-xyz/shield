package rest

import (
	"time"

	env "github.com/caarlos0/env/v10"
)

// Config holds the configuration for the REST server.
// The default values are used if the environment variables are not set.
// The environment variables are:
// - PORT: the port the server listens on
// - REQUESTS_PER_SECOND: the number of requests per second the server can handle (if 0, the rate limiter is disabled)
// - READ_TIMEOUT: the read timeout for the server (if 0, no timeout is set)
// - WRITE_TIMEOUT: the write timeout for the server (if 0, no timeout is set)
// - IDLE_TIMEOUT: the idle timeout for the server (if 0, no timeout is set)
// - CORS_MAX_AGE: the max age for the CORS header
// - CORS_EXTRA_ALLOWED_HEADERS: the extra allowed headers for the CORS header (comma separated)
type Config struct {
	Port                    int           `env:"PORT" envDefault:"8080"`
	MetricsPort             int           `env:"METRICS_PORT" envDefault:"9090"`
	RPS                     int           `env:"REQUESTS_PER_SECOND" envDefault:"100"`
	ReadTimeout             time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout            time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout             time.Duration `env:"IDLE_TIMEOUT" envDefault:"15s"`
	CORSMaxAge              int           `env:"CORS_MAX_AGE" envDefault:"86400"`
	CORSExtraAllowedHeaders string        `env:"CORS_EXTRA_ALLOWED_HEADERS" envDefault:""`
}

// GetConfigFromEnv gets the configuration from the environment variables.
func GetConfigFromEnv() (*Config, error) {
	config := &Config{}
	err := env.Parse(config)
	return config, err
}
