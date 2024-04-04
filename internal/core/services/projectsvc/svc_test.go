package projectsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
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
