package projectsvc

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/projectmockrepo"
)

func TestService_Create(t *testing.T) {
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

func TestService_SetEncryptionPart(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testProjectID := "test-project-id"
	testPart := "test-part"

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
				mockRepo.On("GetEncryptionPart", mock.Anything, testProjectID).Return("", domain.ErrEncryptionPartNotFound)
				mockRepo.On("SetEncryptionPart", mock.Anything, testProjectID, testPart).Return(nil)
			},
		},
		{
			name:    "encryption part already exists",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetEncryptionPart", mock.Anything, testProjectID).Return("test-encryption-part", nil)
			},
			err: domain.ErrEncryptionPartAlreadyExists,
		},
		{
			name:    "repository error on get encryption part",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetEncryptionPart", mock.Anything, testProjectID).Return("", errors.New("repository error"))
			},
		},
		{
			name:    "repository error on set encryption part",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetEncryptionPart", mock.Anything, testProjectID).Return("", domain.ErrEncryptionPartNotFound)
				mockRepo.On("SetEncryptionPart", mock.Anything, testProjectID, testPart).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.SetEncryptionPart(ctx, testProjectID, testPart)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetEncryptionPart() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("SetEncryptionPart() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}
