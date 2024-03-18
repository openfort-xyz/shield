package projectapp

type ProviderOption func(*providerConfig)

func WithCustomProvider(url string) ProviderOption {
	return func(c *providerConfig) {
		c.jwkUrl = &url
	}
}

func WithOpenfortProvider(openfortProjectID string) ProviderOption {
	return func(c *providerConfig) {
		c.openfortProject = &openfortProjectID
	}
}

func WithSupabaseProvider(supabaseProjectReference string) ProviderOption {
	return func(c *providerConfig) {
		c.supabaseProject = &supabaseProjectReference
	}
}

type providerConfig struct {
	jwkUrl          *string
	openfortProject *string
	supabaseProject *string
}
