package usersvc

import (
	"context"
	"errors"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/usermockedrepo"
	"go.openfort.xyz/shield/internal/core/domain/user"
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

	randomExternalUser := &user.ExternalUser{
		ID:             "external-user-id",
		UserID:         "user-id",
		ExternalUserID: "external-id",
		ProviderID:     "provider-id",
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
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(randomUser, nil)
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{randomExternalUser}, nil)
			},
		},
		{
			name:    "get failed external user",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, errors.New("random error"))
			},
		},
		{
			name:    "get failed to get user",
			wantErr: true,
			err:     domainErrors.ErrUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{randomExternalUser}, nil)
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrUserNotFound)
			},
		},
		{
			name:    "create success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, nil)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
				mockRepo.On("CreateExternal", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:    "create failed to create user",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, nil)
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("random error"))
			},
		},
		{
			name:    "create failed to create external user",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, nil)
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
