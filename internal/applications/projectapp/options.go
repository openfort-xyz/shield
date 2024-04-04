package projectapp

type ProviderOption func(*providerConfig)

func WithCustom(url string) ProviderOption {
	return func(c *providerConfig) {
		c.jwkURL = &url
	}
}

func WithOpenfort(openfortProjectID string) ProviderOption {
	return func(c *providerConfig) {
		c.openfortPublishableKey = &openfortProjectID
	}
}

type providerConfig struct {
	jwkURL                 *string
	openfortPublishableKey *string
}

type ProjectOption func(options *projectOptions)

type projectOptions struct {
	generateEncryptionKey bool
}

func WithEncryptionKey() ProjectOption {
	return func(o *projectOptions) {
		o.generateEncryptionKey = true
	}
}
