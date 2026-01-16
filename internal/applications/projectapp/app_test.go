package projectapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/adapters/encryption"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/encryptionpartsmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/notificationsmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/providermockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/sharemockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/usercontactmockrepo"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/services/projectsvc"
	"go.openfort.xyz/shield/internal/core/services/providersvc"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/random"
)

type TestClock struct {
	currentTime time.Time
}

func (tc *TestClock) Now() time.Time {
	return tc.currentTime
}

func (tc *TestClock) SetNewTime(t time.Time) {
	tc.currentTime = t
}

func TestProjectApplication_CreateProject(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	tc := []struct {
		name     string
		projName string
		options  []ProjectOption
		wantErr  error
		wantProj *project.Project
		mock     func()
	}{
		{
			name:     "success",
			projName: "project_name",
			wantProj: &project.Project{
				Name: "project_name",
			},
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
				projectRepo.On("SaveProjectRateLimits", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:     "success with encryption",
			projName: "project_name",
			options: []ProjectOption{
				WithEncryptionKey(),
			},
			wantProj: &project.Project{
				Name:           "project_name",
				EncryptionPart: "encryption_part",
			},
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
				projectRepo.On("SaveProjectRateLimits", mock.Anything, mock.Anything).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("", nil)
				projectRepo.On("SetEncryptionPart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				projectRepo.On("CreateMigration", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:     "create project error",
			projName: "project_name",
			wantErr:  ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(errors.New("repository error"))
				projectRepo.On("SaveProjectRateLimits", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:     "encryption part error",
			projName: "project_name",
			options: []ProjectOption{
				WithEncryptionKey(),
			},
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
				projectRepo.On("SaveProjectRateLimits", mock.Anything, mock.Anything).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("", errors.New("repository error"))
				projectRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:     "encryption part error and delete error",
			projName: "project_name",
			options: []ProjectOption{
				WithEncryptionKey(),
			},
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
				projectRepo.On("SaveProjectRateLimits", mock.Anything, mock.Anything).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("", errors.New("repository error"))
				projectRepo.On("Delete", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			proj, err := app.CreateProject(ctx, tt.projName, false, tt.options...)
			ass.Equal(tt.wantErr, err)
			if tt.wantErr == nil {
				ass.Equal(tt.wantProj.Name, proj.Name)
				ass.NotZero(proj.APIKey)
				ass.NotZero(proj.APISecret)
				if tt.wantProj.EncryptionPart != "" {
					ass.NotZero(proj.EncryptionPart)
				}
			}
		})
	}
}

func TestProjectApplication_GetProject(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)
	projOK := &project.Project{
		ID:             "project-id",
		Name:           "project name",
		APIKey:         "api-key",
		APISecret:      "XXXXX",
		EncryptionPart: "",
	}

	tc := []struct {
		name     string
		wantErr  error
		wantProj *project.Project
		mock     func()
	}{
		{
			name: "success",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, mock.Anything).Return(projOK, nil)
			},
			wantProj: projOK,
			wantErr:  nil,
		},
		{
			name: "project not found",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrProjectNotFound)
			},
			wantProj: nil,
			wantErr:  ErrProjectNotFound,
		},
		{
			name: "internal error",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("repository error"))
			},
			wantProj: nil,
			wantErr:  ErrInternal,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			proj, err := app.GetProject(ctx)
			ass.Equal(tt.wantErr, err)
			ass.Equal(tt.wantProj, proj)
		})
	}
}

func TestProjectApplication_AddProviders(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)
	validPEM := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1ljaGMp9BrY6KQtUIWhw\ng2weyyF65zzNFR9VCxyxk7M/NCTvash6nJO4HwZ+/51YO6kZFr0JDdIMrMmNu/pE\na4FfvmAQJ+vDdc8LSwS7IWAp9y04MZVVFLEQzbToQ3kqkaJV5KsbKuADjm3JCXng\nkeOvuS04AeO4W2lB5BqQ+wX5TjAZ9P7xusJUd2ovk1kWVKeJDTxpAImpVhK2nLZ3\nFV/TWWVYutYFU1wmoRRyOeypTP4ZSPhKB5s6PqQuyl9KPqiWz7ESL9zAW3/yxONb\nEPc9pB8w/qXcW++g6iCYN66xH4punt7KuismzQwGysgnMyK6UnNuOJyJznPzAvB+\nQwIDAQAB\n-----END PUBLIC KEY-----\n"

	tc := []struct {
		name          string
		wantErr       error
		options       []ProviderOption
		wantProviders int
		mock          func()
	}{
		{
			name: "success",
			options: []ProviderOption{
				WithOpenfort("publishableKey"),
				WithCustomJWK("ur"),
			},
			wantErr:       nil,
			wantProviders: 2,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeOpenfort).Return(nil, domainErrors.ErrProviderNotFound)
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(nil, domainErrors.ErrProviderNotFound)
				providerRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				providerRepo.On("CreateOpenfort", mock.Anything, mock.AnythingOfType("*provider.OpenfortConfig")).Return(nil)
				providerRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(nil)
			},
		},
		{
			name: "success with pem",
			options: []ProviderOption{
				WithCustomPEM(validPEM, provider.KeyTypeRSA),
			},
			wantErr:       nil,
			wantProviders: 1,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(nil, domainErrors.ErrProviderNotFound)
				providerRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				providerRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(nil)
			},
		},
		{
			name: "error with bogus pem",
			options: []ProviderOption{
				WithCustomPEM("---BEGIN RACCOON KEY---\n0xTRASHCONTAINER\n---END RACCOON KEY---", provider.KeyTypeRSA),
			},
			wantErr:       ErrInvalidPemCertificate,
			wantProviders: 0,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(nil, domainErrors.ErrProviderNotFound)
				providerRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(nil)
				providerRepo.On("CreateCustom", mock.Anything, mock.AnythingOfType("*provider.CustomConfig")).Return(nil)
			},
		},
		{
			name:    "no providers",
			wantErr: ErrNoProviderSpecified,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
			},
		},
		{
			name: "openfort provider already exists",
			options: []ProviderOption{
				WithOpenfort("publishableKey"),
			},
			wantErr: ErrProviderAlreadyExists,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeOpenfort).Return(&provider.Provider{}, nil)
			},
		},
		{
			name: "custom provider already exists",
			options: []ProviderOption{
				WithCustomJWK("ur"),
			},
			wantErr: ErrProviderAlreadyExists,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(&provider.Provider{}, nil)
			},
		},
		{
			name: "custom provider already exists",
			options: []ProviderOption{
				WithCustomPEM(validPEM, provider.KeyTypeRSA),
			},
			wantErr: ErrProviderAlreadyExists,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(&provider.Provider{}, nil)
			},
		},
		{
			name: "custom provider conflict config",
			options: []ProviderOption{
				WithCustomJWK("ur"),
				WithCustomPEM(validPEM, provider.KeyTypeRSA),
			},
			wantErr: ErrJWKPemConflict,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
			},
		},
		{
			name: "error getting openfort provider",
			options: []ProviderOption{
				WithOpenfort("publishableKey"),
			},
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeOpenfort).Return(nil, errors.New("repository error"))
			},
		},
		{
			name: "error getting custom provider",
			options: []ProviderOption{
				WithCustomJWK("ur"),
			},
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(nil, errors.New("repository error"))
			},
		},
		{
			name: "error getting custom provider",
			options: []ProviderOption{
				WithCustomPEM(validPEM, provider.KeyTypeRSA),
			},
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(nil, errors.New("repository error"))
			},
		},
		{
			name: "error configuring provider",
			options: []ProviderOption{
				WithOpenfort("publishableKey"),
				WithCustomJWK("ur"),
			},
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeOpenfort).Return(nil, domainErrors.ErrProviderNotFound)
				providerRepo.On("GetByProjectAndType", mock.Anything, mock.Anything, provider.TypeCustom).Return(nil, domainErrors.ErrProviderNotFound)
				providerRepo.On("Create", mock.Anything, mock.AnythingOfType("*provider.Provider")).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			providers, err := app.AddProviders(ctx, tt.options...)
			ass.Equal(tt.wantErr, err)
			ass.Equal(tt.wantProviders, len(providers))
		})
	}
}

func TestProjectApplication_GetProviders(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)
	providers := []*provider.Provider{
		{
			ID:        "provider-id",
			ProjectID: "project-id",
			Type:      provider.TypeOpenfort,
			Config: &provider.OpenfortConfig{
				ProviderID:     "provider-id",
				PublishableKey: "publishable-key",
			},
		},
	}

	tc := []struct {
		name          string
		wantErr       error
		wantProviders []*provider.Provider
		mock          func()
	}{
		{
			name:    "success",
			wantErr: nil,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("List", mock.Anything, mock.Anything).Return(providers, nil)
			},
			wantProviders: providers,
		},
		{
			name:    "no providers",
			wantErr: nil,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("List", mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantProviders: nil,
		},
		{
			name:    "error listing providers",
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("repository error"))
			},
			wantProviders: nil,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			providers, err := app.GetProviders(ctx)
			ass.Equal(tt.wantErr, err)
			ass.Equal(tt.wantProviders, providers)
		})
	}
}

func TestProjectApplication_GetProviderDetail(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	prov := &provider.Provider{
		ID:        "provider-id",
		ProjectID: "project_id",
		Type:      provider.TypeOpenfort,
		Config: &provider.OpenfortConfig{
			ProviderID:     "provider-id",
			PublishableKey: "publishable-key",
		},
	}

	tc := []struct {
		name       string
		providerID string
		wantProv   *provider.Provider
		wantErr    error
		mock       func()
	}{
		{
			name:       "success",
			providerID: "provider-id",
			wantProv:   prov,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(prov, nil)
				providerRepo.On("GetOpenfort", mock.Anything, mock.Anything).Return(prov.Config, nil)
			},
		},
		{
			name:       "provider not found",
			providerID: "provider-id",
			wantProv:   nil,
			wantErr:    ErrProviderNotFound,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrProviderNotFound)
			},
		},
		{
			name:       "error getting provider",
			providerID: "provider-id",
			wantProv:   nil,
			wantErr:    ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:       "unauthorized provider",
			providerID: "provider-id",
			wantProv:   nil,
			wantErr:    ErrProviderNotFound,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(&provider.Provider{ProjectID: "other-project"}, nil)
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			prov, err := app.GetProviderDetail(ctx, tt.providerID)
			ass.Equal(tt.wantErr, err)
			ass.Equal(tt.wantProv, prov)
		})
	}
}

func TestProjectApplication_UpdateProvider(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)
	validPEM := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1ljaGMp9BrY6KQtUIWhw\ng2weyyF65zzNFR9VCxyxk7M/NCTvash6nJO4HwZ+/51YO6kZFr0JDdIMrMmNu/pE\na4FfvmAQJ+vDdc8LSwS7IWAp9y04MZVVFLEQzbToQ3kqkaJV5KsbKuADjm3JCXng\nkeOvuS04AeO4W2lB5BqQ+wX5TjAZ9P7xusJUd2ovk1kWVKeJDTxpAImpVhK2nLZ3\nFV/TWWVYutYFU1wmoRRyOeypTP4ZSPhKB5s6PqQuyl9KPqiWz7ESL9zAW3/yxONb\nEPc9pB8w/qXcW++g6iCYN66xH4punt7KuismzQwGysgnMyK6UnNuOJyJznPzAvB+\nQwIDAQAB\n-----END PUBLIC KEY-----\n"

	openfortProvider := &provider.Provider{
		ID:        "provider-id",
		ProjectID: "project_id",
		Type:      provider.TypeOpenfort,
		Config: &provider.OpenfortConfig{
			ProviderID:     "provider-id",
			PublishableKey: "publishable-key",
		},
	}

	customProvider := &provider.Provider{
		ID:        "provider-id",
		ProjectID: "project_id",
		Type:      provider.TypeCustom,
		Config: &provider.CustomConfig{
			ProviderID: "provider-id",
			JWK:        "url",
		},
	}

	tc := []struct {
		name       string
		providerID string
		options    []ProviderOption
		wantErr    error
		mock       func()
	}{
		{
			name:       "success openfort",
			providerID: "provider-id",
			wantErr:    nil,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(openfortProvider, nil)
				providerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				providerRepo.On("UpdateOpenfort", mock.Anything, mock.Anything).Return(nil)
			},
			options: []ProviderOption{
				WithOpenfort("publishable-key"),
			},
		},
		{
			name: "success custom jwk",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(customProvider, nil)
				providerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				providerRepo.On("UpdateCustom", mock.Anything, mock.Anything).Return(nil)
			},
			options: []ProviderOption{
				WithCustomJWK("url"),
			},
		},
		{
			name: "success custom pem",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(customProvider, nil)
				providerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				providerRepo.On("UpdateCustom", mock.Anything, mock.Anything).Return(nil)
			},
			options: []ProviderOption{
				WithCustomPEM(validPEM, provider.KeyTypeRSA),
			},
		},
		{
			name:       "provider not found",
			providerID: "provider-id",
			wantErr:    ErrProviderNotFound,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrProviderNotFound)
			},
		},
		{
			name:       "error getting provider",
			providerID: "provider-id",
			wantErr:    ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:       "unauthorized provider",
			providerID: "provider-id",
			wantErr:    ErrProviderNotFound,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(&provider.Provider{ProjectID: "other-project"}, nil)
			},
		},
		{
			name:       "error provider mismatch",
			providerID: "provider-id",
			wantErr:    ErrProviderMismatch,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(&provider.Provider{ProjectID: "project_id", Type: provider.TypeCustom}, nil)
			},
			options: []ProviderOption{
				WithOpenfort("publishable-key"),
			},
		},
		{
			name:    "error provider mismatch",
			wantErr: ErrProviderMismatch,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(&provider.Provider{ProjectID: "project_id", Type: provider.TypeOpenfort}, nil)
			},
			options: []ProviderOption{
				WithCustomJWK("ur"),
			},
		},
		{
			name:    "error provider mismatch",
			wantErr: ErrProviderMismatch,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(&provider.Provider{ProjectID: "project_id", Type: provider.TypeOpenfort}, nil)
			},
			options: []ProviderOption{
				WithCustomPEM("pem", provider.KeyTypeECDSA),
			},
		},
		{
			name:       "error key not specified",
			providerID: "provider-id",
			wantErr:    ErrKeyTypeNotSpecified,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(&provider.Provider{ProjectID: "project_id", Type: provider.TypeCustom, Config: &provider.CustomConfig{}}, nil)
			},
			options: []ProviderOption{
				WithCustomPEM(validPEM, provider.KeyTypeUnknown),
			},
		},
		{
			name:       "error updating openfort provider",
			providerID: "provider-id",
			wantErr:    ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(openfortProvider, nil)
				providerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				providerRepo.On("UpdateOpenfort", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
			options: []ProviderOption{
				WithOpenfort("publishable-key"),
			},
		},
		{
			name:       "error updating custom provider",
			providerID: "provider-id",
			wantErr:    ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(customProvider, nil)
				providerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				providerRepo.On("UpdateCustom", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
			options: []ProviderOption{
				WithCustomJWK("ur"),
			},
		},
		{
			name:       "error updating custom provider (different key type)",
			providerID: "provider-id",
			wantErr:    ErrInvalidPemCertificate,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(customProvider, nil)
				providerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				providerRepo.On("UpdateCustom", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
			options: []ProviderOption{
				WithCustomPEM(validPEM, provider.KeyTypeECDSA),
			},
		},
		{
			name:       "error updating custom provider (invalid PEM)",
			providerID: "provider-id",
			wantErr:    ErrInvalidPemCertificate,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(customProvider, nil)
				providerRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				providerRepo.On("UpdateCustom", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
			options: []ProviderOption{
				WithCustomPEM("---BEGIN RACCOON KEY---\n0xTRASHCONTAINER\n---END RACCOON KEY---\n", provider.KeyTypeRSA),
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			err := app.UpdateProvider(ctx, tt.providerID, tt.options...)
			ass.Equal(tt.wantErr, err)
		})
	}
}

func TestProjectApplication_RemoveProvider(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	openfortProvider := &provider.Provider{
		ID:        "provider-id",
		ProjectID: "project_id",
		Type:      provider.TypeOpenfort,
		Config: &provider.OpenfortConfig{
			ProviderID:     "provider-id",
			PublishableKey: "publishable-key",
		},
	}

	tc := []struct {
		name       string
		providerID string
		wantErr    error
		mock       func()
	}{
		{
			name:       "success",
			providerID: "provider-id",
			wantErr:    nil,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(openfortProvider, nil)
				providerRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:       "provider not found",
			providerID: "provider-id",
			wantErr:    ErrProviderNotFound,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrProviderNotFound)
			},
		},
		{
			name:       "error getting provider",
			providerID: "provider-id",
			wantErr:    ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:       "unauthorized provider",
			providerID: "provider-id",
			wantErr:    ErrProviderNotFound,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(&provider.Provider{ProjectID: "other-project"}, nil)
			},
		},
		{
			name:       "error deleting provider",
			providerID: "provider-id",
			wantErr:    ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				providerRepo.ExpectedCalls = nil
				providerRepo.On("Get", mock.Anything, mock.Anything).Return(openfortProvider, nil)
				providerRepo.On("Delete", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			err := app.RemoveProvider(ctx, tt.providerID)
			ass.Equal(tt.wantErr, err)
		})
	}
}

func TestProjectApplication_EncryptProjectShares(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf(key)
	}

	reconstructor := encryptionFactory.CreateReconstructionStrategy(true)
	storedPart, projectPart, err := reconstructor.Split(key)
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}

	encryptedShare := &share.Share{
		ID:      "encrypted_share_id",
		Secret:  "djksalfjadsfds",
		UserID:  "user_id",
		Entropy: share.EntropyUser,
		EncryptionParameters: &share.EncryptionParameters{
			Salt:       "somesalt",
			Iterations: 1000,
			Length:     256,
			Digest:     "SHA-256",
		},
	}

	plainShare := &share.Share{
		ID:      "share_id",
		Secret:  "secret",
		UserID:  "user_id",
		Entropy: share.EntropyNone,
	}

	plainShare2 := &share.Share{
		ID:      "share_id",
		Secret:  "secret",
		UserID:  "user_id",
		Entropy: share.EntropyNone,
	}

	tc := []struct {
		name         string
		externalPart string
		wantErr      error
		mock         func()
	}{
		{
			name:         "success",
			externalPart: projectPart,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				shareRepo.ExpectedCalls = nil
				shareRepo.On("ListProjectIDAndEntropy", mock.Anything, mock.Anything, mock.Anything).Return([]*share.Share{plainShare, encryptedShare}, nil)
				shareRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return(storedPart, nil)
				shareRepo.On("UpdateProjectEncryption", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, mock.Anything).Return(true, nil)
			},
		},
		{
			name:         "encryption part not found",
			externalPart: projectPart,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("", domainErrors.ErrEncryptionPartNotFound)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: ErrEncryptionNotConfigured,
		},
		{
			name:         "error getting encryption part",
			externalPart: projectPart,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("", errors.New("repository error"))
				projectRepo.On("HasSuccessfulMigration", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: ErrInternal,
		},
		{
			name:         "error reconstructing encryption key",
			externalPart: "invalid",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("invalid", nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: ErrInvalidEncryptionPart,
		},
		{
			name:         "error listing shares",
			externalPart: projectPart,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				shareRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return(storedPart, nil)
				shareRepo.On("ListProjectIDAndEntropy", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("repository error"))
				projectRepo.On("HasSuccessfulMigration", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: ErrInternal,
		},
		{
			name:         "error updating share",
			externalPart: projectPart,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return(storedPart, nil)
				shareRepo.ExpectedCalls = nil
				shareRepo.On("ListProjectIDAndEntropy", mock.Anything, mock.Anything, mock.Anything).Return([]*share.Share{plainShare2}, nil)
				shareRepo.On("UpdateProjectEncryption", mock.Anything, "share_id", mock.Anything).Return(errors.New("repository error"))
				projectRepo.On("HasSuccessfulMigration", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: ErrInternal,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			err := app.EncryptProjectShares(ctx, tt.externalPart)
			ass.Equal(tt.wantErr, err)
		})
	}
}

func TestProjectApplication_RegisterEncryptionKey(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	tc := []struct {
		name    string
		wantErr error
		mock    func()
	}{
		{
			name:    "success",
			wantErr: nil,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domainErrors.ErrEncryptionPartNotFound)
				projectRepo.On("SetEncryptionPart", mock.Anything, "project_id", mock.Anything).Return(nil)
				projectRepo.On("CreateMigration", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "encryption part already exists",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("encryption_part", nil)
			},
			wantErr: ErrEncryptionPartAlreadyExists,
		},
		{
			name:    "error getting encryption part",
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", errors.New("repository error"))
			},
		},
		{
			name:    "error setting encryption part",
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domainErrors.ErrEncryptionPartNotFound)
				projectRepo.On("SetEncryptionPart", mock.Anything, "project_id", mock.Anything).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			_, err := app.RegisterEncryptionKey(ctx)
			ass.Equal(tt.wantErr, err)
		})
	}
}

func TestProjectApplication_RegisterEncryptionSession(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	tc := []struct {
		name    string
		wantErr error
		mock    func()
	}{
		{
			name:    "success",
			wantErr: nil,
			mock: func() {
				encryptionPartsRepo.ExpectedCalls = nil
				encryptionPartsRepo.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, "project_id").Return(&project.Project{ID: "project_id"}, nil)
			},
		},
		{
			name:    "error setting encryption session",
			wantErr: ErrInternal,
			mock: func() {
				encryptionPartsRepo.ExpectedCalls = nil
				encryptionPartsRepo.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			_, err := app.RegisterEncryptionSession(ctx, "encryptionPart", "irrelevant", nil)
			ass.Equal(tt.wantErr, err)
		})
	}
}

func TestProjectApplication_Enable2FA(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	tc := []struct {
		name    string
		wantErr error
		mock    func()
	}{
		{
			name:    "success",
			wantErr: nil,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, "project_id").Return(&project.Project{ID: "project_id", Enable2FA: false}, nil)
				projectRepo.On("Update2FA", mock.Anything, "project_id", true).Return(nil)
			},
		},
		{
			name:    "2FA already enabled",
			wantErr: ErrProject2FAAlreadyEnabled,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, "project_id").Return(&project.Project{ID: "project_id", Enable2FA: true}, nil)
			},
		},
		{
			name:    "project not found",
			wantErr: ErrProjectNotFound,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, "project_id").Return(nil, domainErrors.ErrProjectNotFound)
			},
		},
		{
			name:    "error getting project",
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, "project_id").Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "error updating 2FA",
			wantErr: ErrInternal,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("Get", mock.Anything, "project_id").Return(&project.Project{ID: "project_id", Enable2FA: false}, nil)
				projectRepo.On("Update2FA", mock.Anything, "project_id", true).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			err := app.Enable2FA(ctx)
			ass.Equal(tt.wantErr, err)
		})
	}
}
