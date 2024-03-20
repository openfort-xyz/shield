package rest

import "github.com/caarlos0/env/v10"

type Config struct {
	Port int `envconfig:"PORT" envdefault:"8080"`
}

func GetConfigFromEnv() (*Config, error) {
	config := &Config{}
	err := env.Parse(config)
	return config, err
}
