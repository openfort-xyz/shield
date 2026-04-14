package usersvc

import (
	"context"
	"errors"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/usermockedrepo"
	"github.com/openfort-xyz/shield/internal/core/domain/user"
	"github.com/stretchr/testify/mock"
)

func TestService_GetOrCreate(t *testing.T) {
	mockRepo := new(usermockedrepo.MockUserRepository)
	svc := New(mockRepo)
	ctx := context.Background()

	projectID := "project"
	providerID := "provider"
	externalUserID := "external"

	randomUser := &user.User{
		ID:        "user-id",
		ProjectID: "project-id",
	}

	tc := []struct {
		name    string
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:    "get success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindUserByExternalID", mock.Anything, externalUserID, providerID).Return(randomUser, nil)
			},
		},
		{
			name:    "get failed external user",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindUserByExternalID", mock.Anything, externalUserID, providerID).Return(nil, errors.New("random error"))
			},
		},
		{
			name:    "create success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindUserByExternalID", mock.Anything, externalUserID, providerID).Return(nil, domainErrors.ErrExternalUserNotFound)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
				mockRepo.On("CreateExternal", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:    "create failed to create user",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindUserByExternalID", mock.Anything, externalUserID, providerID).Return(nil, domainErrors.ErrExternalUserNotFound)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("random error"))
			},
		},
		{
			name:    "create failed to create external user",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindUserByExternalID", mock.Anything, externalUserID, providerID).Return(nil, domainErrors.ErrExternalUserNotFound)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
				mockRepo.On("CreateExternal", mock.Anything, mock.Anything).Return(errors.New("random error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.GetOrCreate(ctx, projectID, externalUserID, providerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("have error = %v, wantErr %v", err != nil, tt.wantErr)
				return
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("have error = %v, wantErr %v", err, tt.err)
				return
			}
		})
	}
}
