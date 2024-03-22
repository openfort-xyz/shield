package services

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type ProviderService interface {
	Configure(ctx context.Context, projectID string, config ProviderConfig) (*provider.Provider, error)
	Get(ctx context.Context, providerID string) (*provider.Provider, error)
	List(ctx context.Context, projectID string) ([]*provider.Provider, error)
	UpdateConfig(ctx context.Context, config interface{}) error
	Remove(ctx context.Context, projectID string, providerID string) error
}

type ProviderConfig interface {
	GetType() provider.Type
	GetConfig() interface{}
}

type CustomProviderConfig struct {
	JWKUrl string
}

func (c *CustomProviderConfig) GetType() provider.Type {
	return provider.TypeCustom
}

func (c *CustomProviderConfig) GetConfig() interface{} {
	return c
}

type OpenfortProviderConfig struct {
	OpenfortProject string
}

func (o *OpenfortProviderConfig) GetType() provider.Type {
	return provider.TypeOpenfort
}

func (o *OpenfortProviderConfig) GetConfig() interface{} {
	return o
}
