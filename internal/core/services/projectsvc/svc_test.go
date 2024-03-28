package projectsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/projectmockrepo"
)

func TestCreateProject(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testName := "test-project"

	tc := []struct {
		name    string
		wantErr bool
		param   string
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
			},
		},
		{
			name:    "repository error on create",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(errors.New("repository error"))
			},
		},
		{
			name: "failed to encrypt secret",
			mock: func() {
				svc.(*service).cost = 9999
			},
			wantErr: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			result, err := svc.Create(ctx, testName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Errorf("Create() expected a project but got nil")
			}
		})
	}
}

func TestGetProject(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProjectID := "get-test-project-id"

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Get", mock.Anything, testProjectID).Return(&project.Project{}, nil)
			},
		},
		{
			name:    "project not found",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Get", mock.Anything, testProjectID).Return(nil, errors.New("project not found"))
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Get", mock.Anything, testProjectID).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.Get(ctx, testProjectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProjectByAPIKey(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testAPIKey := "test-api-key"

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByAPIKey", mock.Anything, testAPIKey).Return(&project.Project{}, nil)
			},
		},
		{
			name:    "project not found",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByAPIKey", mock.Anything, testAPIKey).Return(nil, errors.New("project not found"))
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByAPIKey", mock.Anything, testAPIKey).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.GetByAPIKey(ctx, testAPIKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddAllowedOrigin(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProjectID := "add-origin-test-project-id"
	testOrigin := "test-origin"

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("AddAllowedOrigin", mock.Anything, testProjectID, testOrigin).Return(nil)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("AddAllowedOrigin", mock.Anything, testProjectID, testOrigin).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.AddAllowedOrigin(ctx, testProjectID, testOrigin)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddAllowedOrigin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveAllowedOrigin(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProjectID := "test-project-id"
	testOrigin := "test-origin"

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("RemoveAllowedOrigin", mock.Anything, testProjectID, testOrigin).Return(nil)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("RemoveAllowedOrigin", mock.Anything, testProjectID, testOrigin).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.RemoveAllowedOrigin(ctx, testProjectID, testOrigin)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveAllowedOrigin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAllowedOrigins(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProjectID := "test-project-id"

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetAllowedOrigins", mock.Anything, testProjectID).Return([]string{}, nil)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetAllowedOrigins", mock.Anything, testProjectID).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.GetAllowedOrigins(ctx, testProjectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllowedOrigins() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
