package services

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type ProviderService interface {
	Configure(ctx context.Context, projectID string, config ProviderConfig) (*provider.Provider, error)
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

type SupabaseProviderConfig struct {
	SupabaseProject string
}

func (s *SupabaseProviderConfig) GetType() provider.Type {
	return provider.TypeSupabase
}

func (s *SupabaseProviderConfig) GetConfig() interface{} {
	return s
}
