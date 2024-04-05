package projectapp

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/services/projectsvc"
	"go.openfort.xyz/shield/internal/core/services/providersvc"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/projectmockrepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/providermockrepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/sharemockrepo"
	"go.openfort.xyz/shield/pkg/contexter"
	"testing"
)

func TestProjectApplication_CreateProject(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

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
				projectRepo.Calls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
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
				projectRepo.Calls = nil
				providerRepo.Calls = nil
				shareRepo.Calls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("", nil)
				projectRepo.On("SetEncryptionPart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:     "create project error",
			projName: "project_name",
			wantErr:  ErrInternal,
			mock: func() {
				projectRepo.Calls = nil
				providerRepo.Calls = nil
				shareRepo.Calls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(errors.New("repository error"))
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
				projectRepo.Calls = nil
				providerRepo.Calls = nil
				shareRepo.Calls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
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
				projectRepo.Calls = nil
				providerRepo.Calls = nil
				shareRepo.Calls = nil
				projectRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, mock.Anything).Return("", errors.New("repository error"))
				projectRepo.On("Delete", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			proj, err := app.CreateProject(ctx, tt.projName, tt.options...)
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
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name     string
		wantErr  error
		wantProj *project.Project
		mock     func()
	}{
		{},
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
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name          string
		wantErr       error
		options       []ProviderOption
		wantProviders []*provider.Provider
		mock          func()
	}{
		{},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			providers, err := app.AddProviders(ctx, tt.options...)
			ass.Equal(tt.wantErr, err)
			ass.Equal(tt.wantProviders, providers)
		})
	}
}

func TestProjectApplication_GetProviders(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name          string
		wantErr       error
		wantProviders []*provider.Provider
		mock          func()
	}{
		{},
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
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name       string
		providerID string
		wantProv   *provider.Provider
		wantErr    error
		mock       func()
	}{
		{},
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
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name       string
		providerID string
		options    []ProviderOption
		wantErr    error
		mock       func()
	}{
		{},
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
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name       string
		providerID string
		wantErr    error
		mock       func()
	}{
		{},
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

func TestProjectApplication_AddAllowedOrigin(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name    string
		origin  string
		wantErr error
		mock    func()
	}{
		{},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			err := app.AddAllowedOrigin(ctx, tt.origin)
			ass.Equal(tt.wantErr, err)
		})
	}
}

func TestProjectApplication_RemoveAllowedOrigin(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name    string
		origin  string
		wantErr error
		mock    func()
	}{
		{},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			err := app.RemoveAllowedOrigin(ctx, tt.origin)
			ass.Equal(tt.wantErr, err)
		})
	}
}

func TestProjectApplication_GetAllowedOrigins(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name        string
		wantOrigins []string
		wantErr     error
		mock        func()
	}{
		{},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			origins, err := app.GetAllowedOrigins(ctx)
			ass.Equal(tt.wantErr, err)
			ass.Equal(tt.wantOrigins, origins)
		})
	}
}

func TestProjectApplication_EncryptProjectShares(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name         string
		externalPart string
		wantErr      error
		mock         func()
	}{
		{},
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
	projectService := projectsvc.New(projectRepo)
	providerService := providersvc.New(providerRepo)
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo)

	tc := []struct {
		name             string
		wantExternalPart string
		wantErr          error
		mock             func()
	}{
		{},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			externalPart, err := app.RegisterEncryptionKey(ctx)
			ass.Equal(tt.wantErr, err)
			ass.Equal(tt.wantExternalPart, externalPart)
		})
	}
}
