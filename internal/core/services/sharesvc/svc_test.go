package sharesvc

import (
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/sharemockrepo"
	"testing"
)

func TestCreateShare(t *testing.T) {
	mockRepo := new(sharemockrepo.MockShareRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testUserID := "test-user"
	testData := "test-data"

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
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domain.ErrShareNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
			},
		},
		{
			name:    "share already exists",
			wantErr: true,
			err:     domain.ErrShareAlreadyExists,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(&share.Share{}, nil)
			},
		},
		{
			name:    "repository error on get",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "repository error on create",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domain.ErrShareNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.Create(ctx, testUserID, testData)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Create() error = %v, expected error %v", err, tt.err)
			}
		})
	}
}

func TestGetShareByUserID(t *testing.T) {
	mockRepo := new(sharemockrepo.MockShareRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testUserID := "test-user"

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
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(&share.Share{}, nil)
			},
		},
		{
			name:    "share not found",
			wantErr: true,
			err:     domain.ErrShareNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domain.ErrShareNotFound)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			result, err := svc.GetByUserID(ctx, testUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByUserID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("GetByUserID() error = %v, expected error %v", err, tt.err)
			}
			if !tt.wantErr && result == nil {
				t.Errorf("GetByUserID() expected a result but got nil")
			}
		})
	}
}
