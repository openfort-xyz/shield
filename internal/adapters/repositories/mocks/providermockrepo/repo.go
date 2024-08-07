package providermockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type MockProviderRepository struct {
	mock.Mock
}

var _ repositories.ProviderRepository = (*MockProviderRepository)(nil)

func (m *MockProviderRepository) Create(ctx context.Context, prov *provider.Provider) error {
	args := m.Mock.Called(ctx, prov)
	return args.Error(0)
}

func (m *MockProviderRepository) GetByProjectAndType(ctx context.Context, projectID string, providerType provider.Type) (*provider.Provider, error) {
	args := m.Mock.Called(ctx, projectID, providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.Provider), args.Error(1)
}

func (m *MockProviderRepository) GetByAPIKeyAndType(ctx context.Context, apiKey string, providerType provider.Type) (*provider.Provider, error) {
	args := m.Mock.Called(ctx, apiKey, providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*provider.Provider), args.Error(1)
}

func (m *MockProviderRepository) Get(ctx context.Context, id string) (*provider.Provider, error) {
	args := m.Mock.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.Provider), args.Error(1)
}

func (m *MockProviderRepository) List(ctx context.Context, projectID string) ([]*provider.Provider, error) {
	args := m.Mock.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*provider.Provider), args.Error(1)
}

func (m *MockProviderRepository) Delete(ctx context.Context, providerID string) error {
	args := m.Mock.Called(ctx, providerID)
	return args.Error(0)
}

func (m *MockProviderRepository) CreateCustom(ctx context.Context, prov *provider.CustomConfig) error {
	args := m.Mock.Called(ctx, prov)
	return args.Error(0)
}

func (m *MockProviderRepository) GetCustom(ctx context.Context, providerID string) (*provider.CustomConfig, error) {
	args := m.Mock.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.CustomConfig), args.Error(1)
}

func (m *MockProviderRepository) UpdateCustom(ctx context.Context, prov *provider.CustomConfig) error {
	args := m.Mock.Called(ctx, prov)
	return args.Error(0)
}

func (m *MockProviderRepository) CreateOpenfort(ctx context.Context, prov *provider.OpenfortConfig) error {
	args := m.Mock.Called(ctx, prov)
	return args.Error(0)
}

func (m *MockProviderRepository) GetOpenfort(ctx context.Context, providerID string) (*provider.OpenfortConfig, error) {
	args := m.Mock.Called(ctx, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.OpenfortConfig), args.Error(1)
}

func (m *MockProviderRepository) UpdateOpenfort(ctx context.Context, prov *provider.OpenfortConfig) error {
	args := m.Mock.Called(ctx, prov)
	return args.Error(0)
}
