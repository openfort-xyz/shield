package providersvc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/providermockrepo"
)

func TestConfigureProvider(t *testing.T) {
	mockRepo := new(providermockrepo.MockProviderRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	projectID := "test-project"
	jwkURL := "http://jwk.url"
	openfortProject := "openfort-project"

	tc := []struct {
		name    string
		config  services.ProviderConfig
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:   "configure custom provider success",
			config: &services.CustomProviderConfig{JWKUrl: jwkURL},
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(nil)
			},
		},
		{
			name:   "configure Openfort provider success",
			config: &services.OpenfortProviderConfig{OpenfortProject: openfortProject},
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateOpenfort", mock.Anything, mock.AnythingOfType("*provider.OpenfortConfig")).Return(nil)
			},
		},
		{
			name:    "invalid provider config",
			config:  nil,
			wantErr: true,
			mock:    func() {},
			err:     domain.ErrNoProviderConfig,
		},
		{
			name:    "invalid provider type",
			config:  &unknownProviderConfig{},
			wantErr: true,
			mock:    func() {},
			err:     domain.ErrUnknownProviderType,
		},
		{
			name:    "invalid custom provider config",
			config:  &fakeCustomProviderConfig{},
			wantErr: true,
			mock:    func() {},
			err:     domain.ErrInvalidProviderConfig,
		},
		{
			name:    "invalid openfort provider config",
			config:  &fakeOpenfortProviderConfig{},
			wantErr: true,
			mock:    func() {},
			err:     domain.ErrInvalidProviderConfig,
		},
		{
			name: "failed to get custom provider",
			config: &services.CustomProviderConfig{
				JWKUrl: jwkURL,
			},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, errors.New("repository error"))
			},
		},
		{
			name: "custom provider already exists",
			config: &services.CustomProviderConfig{
				JWKUrl: jwkURL,
			},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(&provider.Provider{}, nil)
			},
		},
		{
			name: "failed to create custom provider",
			config: &services.CustomProviderConfig{
				JWKUrl: jwkURL,
			},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(errors.New("repository error"))
			},
		},
		{
			name: "failed to create custom provider config and provider is deleted successfully",
			config: &services.CustomProviderConfig{
				JWKUrl: jwkURL,
			},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(errors.New("repository error"))
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)
			},
		},
		{
			name: "failed to create custom provider config and provider is not deleted",
			config: &services.CustomProviderConfig{
				JWKUrl: jwkURL,
			},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(errors.New("repository error"))
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("repository error"))
			},
		},
		{
			name:    "failed to get openfort provider",
			config:  &services.OpenfortProviderConfig{OpenfortProject: openfortProject},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "openfort provider already exists",
			config:  &services.OpenfortProviderConfig{OpenfortProject: openfortProject},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(&provider.Provider{}, nil)
			},
		},
		{
			name:    "failed to create openfort provider",
			config:  &services.OpenfortProviderConfig{OpenfortProject: openfortProject},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(errors.New("repository error"))
			},
		},
		{
			name:    "failed to create openfort provider config and provider is deleted successfully",
			config:  &services.OpenfortProviderConfig{OpenfortProject: openfortProject},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateOpenfort", mock.Anything, mock.AnythingOfType("*provider.OpenfortConfig")).Return(errors.New("repository error"))
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)
			},
		},
		{
			name:    "failed to create openfort provider config and provider is not deleted",
			config:  &services.OpenfortProviderConfig{OpenfortProject: openfortProject},
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateOpenfort", mock.Anything, mock.AnythingOfType("*provider.OpenfortConfig")).Return(errors.New("repository error"))
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.Configure(ctx, projectID, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func TestGetProvider(t *testing.T) {
	mockRepo := new(providermockrepo.MockProviderRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProviderID := "test-provider-id"

	tc := []struct {
		name    string
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(&provider.Provider{}, nil)
			},
		},
		{
			name:    "provider not found",
			wantErr: true,
			err:     domain.ErrProviderNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, domain.ErrProviderNotFound)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.Get(ctx, testProviderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func TestListProviders(t *testing.T) {
	mockRepo := new(providermockrepo.MockProviderRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProjectID := "test-project-id"

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "success with providers",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("List", mock.Anything, mock.AnythingOfType("string")).Return([]*provider.Provider{{}}, nil)
			},
		},
		{
			name:    "success with no providers",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("List", mock.Anything, mock.AnythingOfType("string")).Return([]*provider.Provider{}, nil)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("List", mock.Anything, mock.AnythingOfType("string")).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			result, err := svc.List(ctx, testProjectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Errorf("List() expected a result but got nil")
			}
		})
	}
}

func TestUpdateConfig(t *testing.T) {
	mockRepo := new(providermockrepo.MockProviderRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	customConfig := &provider.CustomConfig{ProviderID: "custom-id", JWK: "http://jwk.url"}
	openfortConfig := &provider.OpenfortConfig{ProviderID: "openfort-id", PublishableKey: "openfort-project"}

	tc := []struct {
		name    string
		config  interface{}
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:   "update custom config success",
			config: customConfig,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("UpdateCustom", mock.Anything, customConfig).Return(nil)
			},
		},
		{
			name:   "update openfort config success",
			config: openfortConfig,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("UpdateOpenfort", mock.Anything, openfortConfig).Return(nil)
			},
		},
		{
			name:    "custom config repository error",
			config:  customConfig,
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("UpdateCustom", mock.Anything, customConfig).Return(errors.New("repository error"))
			},
		},
		{
			name:    "openfort config repository error",
			config:  openfortConfig,
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("UpdateOpenfort", mock.Anything, openfortConfig).Return(errors.New("repository error"))
			},
		},
		{
			name:    "invalid config type",
			config:  unknownProviderConfig{},
			wantErr: true,
			err:     domain.ErrInvalidProviderConfig,
			mock: func() {
				mockRepo.ExpectedCalls = nil
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.UpdateConfig(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("UpdateConfig() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func TestRemoveProvider(t *testing.T) {
	mockRepo := new(providermockrepo.MockProviderRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProviderID := "test-provider-id"
	testProjectID := "test-project-id"

	tc := []struct {
		name    string
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:    "successful removal",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Delete", mock.Anything, testProviderID).Return(nil)
			},
		},
		{
			name:    "provider not found",
			wantErr: true,
			err:     domain.ErrProviderNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Delete", mock.Anything, testProviderID).Return(domain.ErrProviderNotFound)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Delete", mock.Anything, testProviderID).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.Remove(ctx, testProjectID, testProviderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

type unknownProviderConfig struct{}

func (f *unknownProviderConfig) GetConfig() interface{} { return nil }

func (f *unknownProviderConfig) GetType() provider.Type {
	return provider.TypeUnknown
}

type fakeCustomProviderConfig struct{}

func (f *fakeCustomProviderConfig) GetConfig() interface{} { return nil }

func (f *fakeCustomProviderConfig) GetType() provider.Type {
	return provider.TypeCustom
}

type fakeOpenfortProviderConfig struct{}

func (f *fakeOpenfortProviderConfig) GetConfig() interface{} { return nil }

func (f *fakeOpenfortProviderConfig) GetType() provider.Type {
	return provider.TypeOpenfort
}
