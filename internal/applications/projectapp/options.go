package projectapp

import "go.openfort.xyz/shield/internal/core/domain/provider"

type ProviderOption func(*providerConfig)

func WithCustomJWK(url string) ProviderOption {
	return func(c *providerConfig) {
		c.jwkURL = &url
	}
}

func WithCustomPEM(pem string, keyType provider.KeyType) ProviderOption {
	return func(c *providerConfig) {
		c.pem = &pem
		c.keyType = keyType
	}
}

func WithOpenfort(openfortProjectID string) ProviderOption {
	return func(c *providerConfig) {
		c.openfortPublishableKey = &openfortProjectID
	}
}

type providerConfig struct {
	jwkURL                 *string
	pem                    *string
	keyType                provider.KeyType
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
