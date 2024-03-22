package providersmgr

import "github.com/caarlos0/env/v10"

type Config struct {
	OpenfortBaseURL string `env:"OPENFORT_BASE_URL" envDefault:"https://api.openfort.xyz"`
}

func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	return cfg, err
}
