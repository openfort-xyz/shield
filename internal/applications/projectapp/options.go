package projectapp

type ProviderOption func(*providerConfig)

func WithCustom(url string) ProviderOption {
	return func(c *providerConfig) {
		c.jwkUrl = &url
	}
}

func WithOpenfort(openfortProjectID string) ProviderOption {
	return func(c *providerConfig) {
		c.openfortPublishableKey = &openfortProjectID
	}
}

type providerConfig struct {
	jwkUrl                 *string
	openfortPublishableKey *string
}
