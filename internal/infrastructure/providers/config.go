package providers

import "github.com/caarlos0/env/v10"

type openfortConfig struct {
	OpenfortBaseURL string `env:"OPENFORT_BASE_URL" envDefault:"https://api.openfort.xyz"`
}

type supabaseConfig struct {
	SupabaseAPIKey  string `env:"SUPABASE_API_KEY"`
	SupabaseBaseURL string `env:"SUPABASE_BASE_URL"`
}

type Config struct {
	openfortConfig
	supabaseConfig
}

func GetConfigFromEnv() (*Config, error) {
	ofCf := openfortConfig{}
	err := env.Parse(&ofCf)
	if err != nil {
		return nil, err
	}
	supCf := supabaseConfig{}
	err = env.Parse(&supCf)
	if err != nil {
		return nil, err
	}
	return &Config{
		openfortConfig: ofCf,
		supabaseConfig: supCf,
	}, nil
}
