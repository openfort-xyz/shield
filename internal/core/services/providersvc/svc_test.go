package providersvc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/providermockrepo"
)

func TestConfigureProvider(t *testing.T) {
	mockRepo := new(providermockrepo.MockProviderRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	projectID := "test-project"
	jwkURL := "http://jwk.url"
	openfortPublishableKey := "openfort-project"
	customProvider := &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeCustom,
		Config: &provider.CustomConfig{
			ProviderID: "custom-provider",
			JWK:        jwkURL,
		},
	}

	openfortProvider := &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeOpenfort,
		Config: &provider.OpenfortConfig{
			ProviderID:     "openfort-provider",
			PublishableKey: openfortPublishableKey,
		},
	}

	unknownProvider := &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeUnknown,
	}

	fakeCustomProvider := &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeCustom,
		Config:    &struct{}{},
	}

	fakeOpenfortProvider := &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeOpenfort,
		Config:    &struct{}{},
	}

	tc := []struct {
		name     string
		provider *provider.Provider
		wantErr  bool
		err      error
		mock     func()
	}{
		{
			name:     "configure custom provider success",
			provider: customProvider,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(nil)
			},
		},
		{
			name:     "configure Openfort provider success",
			provider: openfortProvider,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateOpenfort", mock.Anything, mock.AnythingOfType("*provider.OpenfortConfig")).Return(nil)
			},
		},
		{
			name:     "invalid provider type",
			provider: unknownProvider,
			wantErr:  true,
			mock:     func() {},
			err:      domain.ErrUnknownProviderType,
		},
		{
			name:     "invalid custom provider config",
			provider: fakeCustomProvider,
			wantErr:  true,
			mock:     func() {},
			err:      domain.ErrInvalidProviderConfig,
		},
		{
			name:     "invalid openfort provider config",
			provider: fakeOpenfortProvider,
			wantErr:  true,
			mock:     func() {},
			err:      domain.ErrInvalidProviderConfig,
		},
		{
			name:     "failed to create custom provider",
			provider: customProvider,
			wantErr:  true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(errors.New("repository error"))
			},
		},
		{
			name:     "failed to create custom provider config and provider is deleted successfully",
			provider: customProvider,
			wantErr:  true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(errors.New("repository error"))
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)
			},
		},
		{
			name:     "failed to create custom provider config and provider is not deleted",
			provider: customProvider,
			wantErr:  true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeCustom).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(errors.New("repository error"))
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("repository error"))
			},
		},
		{
			name:     "failed to create openfort provider",
			provider: openfortProvider,
			wantErr:  true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(errors.New("repository error"))
			},
		},
		{
			name:     "failed to create openfort provider config and provider is deleted successfully",
			provider: openfortProvider,
			wantErr:  true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByProjectAndType", mock.Anything, projectID, provider.TypeOpenfort).Return(nil, domain.ErrProviderNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				mockRepo.On("CreateOpenfort", mock.Anything, mock.AnythingOfType("*provider.OpenfortConfig")).Return(errors.New("repository error"))
				mockRepo.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)
			},
		},
		{
			name:     "failed to create openfort provider config and provider is not deleted",
			provider: openfortProvider,
			wantErr:  true,
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
			err := svc.Configure(ctx, tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}
