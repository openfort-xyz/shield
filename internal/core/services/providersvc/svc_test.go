package providersvc

import (
	"context"
	"errors"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/providermockrepo"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
	"github.com/stretchr/testify/mock"
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
				mockRepo.On("Create", mock.Anything, projectID, []*provider.Provider{customProvider}, mock.Anything).Return(nil)
			},
		},
		{
			name:     "configure Openfort provider success",
			provider: openfortProvider,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Create", mock.Anything, projectID, []*provider.Provider{openfortProvider}, mock.Anything).Return(nil)
			},
		},
		{
			name:     "invalid provider type",
			provider: unknownProvider,
			wantErr:  true,
			mock:     func() {},
			err:      domainErrors.ErrUnknownProviderType,
		},
		{
			name:     "invalid custom provider config",
			provider: fakeCustomProvider,
			wantErr:  true,
			mock:     func() {},
			err:      domainErrors.ErrInvalidProviderConfig,
		},
		{
			name:     "invalid openfort provider config",
			provider: fakeOpenfortProvider,
			wantErr:  true,
			mock:     func() {},
			err:      domainErrors.ErrInvalidProviderConfig,
		},
		{
			name:     "failed to create provider",
			provider: customProvider,
			wantErr:  true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Create", mock.Anything, projectID, []*provider.Provider{customProvider}, mock.Anything).Return(domainErrors.ErrAPISecretRevoked)
			},
			err: domainErrors.ErrAPISecretRevoked,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.Configure(ctx, projectID, []*provider.Provider{tt.provider}, project.StillCurrent("key", "secret"))
			if (err != nil) != tt.wantErr {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}
