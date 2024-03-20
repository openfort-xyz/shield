package providers

import "github.com/caarlos0/env/v10"

type openfortConfig struct {
	OpenfortBaseURL string `envconfig:"OPENFORT_BASE_URL" envDefault:"https://api.openfort.xyz"`
}

type supabaseConfig struct {
	SupabaseAPIKey  string `envconfig:"SUPABASE_API_KEY"`
	SupabaseBaseURL string `envconfig:"SUPABASE_BASE_URL"`
}

type Config struct {
	openfortConfig
	supabaseConfig
}

func GetConfigFromEnv() (*Config, error) {
	config := &Config{}
	err := env.Parse(config)
	return config, err
}
