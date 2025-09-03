package ofidty

import env "github.com/caarlos0/env/v10"

type Config struct {
	OpenfortBaseURL string `env:"OPENFORT_BASE_URL" envDefault:"http://localhost:3000"`
}

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	return cfg, err
}
