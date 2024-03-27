package usersvc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/user"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/usermockedrepo"
)

func TestCreateUser(t *testing.T) {
	mockRepo := new(usermockedrepo.MockUserRepository)
	svc := New(mockRepo)
	ctx := context.Background()

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:    "failure",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("random error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.Create(ctx, "fdsa")
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	mockRepo := new(usermockedrepo.MockUserRepository)
	svc := New(mockRepo)
	ctx := context.Background()

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
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(&user.User{}, nil)
			},
		},
		{
			name:    "not found",
			wantErr: true,
			err:     domain.ErrUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			},
		},
		{
			name:    "failure",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("random error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.Get(ctx, "fdsa")
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.err)
				return
			}
		})
	}
}

func TestGetUserByExternal(t *testing.T) {
	mockRepo := new(usermockedrepo.MockUserRepository)
	svc := New(mockRepo)
	ctx := context.Background()

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
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{{}}, nil)
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(&user.User{}, nil)
			},
		},
		{
			name:    "external not found",
			wantErr: true,
			err:     domain.ErrExternalUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return(nil, domain.ErrExternalUserNotFound)
			},
		},
		{
			name:    "external empty",
			wantErr: true,
			err:     domain.ErrExternalUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, nil)
			},
		},
		{
			name:    "user not found",
			wantErr: true,
			err:     domain.ErrUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{{}}, nil)
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			},
		},
		{
			name:    "failure",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return(nil, errors.New("random error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.GetByExternal(ctx, "fdsa", "fdsa")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByExternal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("GetByExternal() error = %v, wantErr %v", err, tt.err)
				return
			}
		})
	}
}

func TestCreateExternalUser(t *testing.T) {
	mockRepo := new(usermockedrepo.MockUserRepository)
	svc := New(mockRepo)
	ctx := context.Background()

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
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(&user.User{ProjectID: "project"}, nil)
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, nil)
				mockRepo.On("CreateExternal", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:    "user not found on repo",
			wantErr: true,
			err:     domain.ErrUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			},
		},
		{
			name:    "user empty on repo",
			wantErr: true,
			err:     domain.ErrUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(nil, nil)
			},
		},
		{
			name:    "user not found project mismatch",
			wantErr: true,
			err:     domain.ErrUserNotFound,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(&user.User{ProjectID: "noproject"}, nil)
			},
		},
		{
			name: "external user already exists",
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(&user.User{ProjectID: "project"}, nil)
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{{}}, nil)
			},
			wantErr: true,
			err:     domain.ErrExternalUserAlreadyExists,
		},
		{
			name:    "cant find external user",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(&user.User{ProjectID: "project"}, nil)
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, errors.New("random error"))
			},
		},
		{
			name:    "failure",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = []*mock.Call{}
				mockRepo.On("Get", mock.Anything, mock.Anything).Return(&user.User{ProjectID: "project"}, nil)
				mockRepo.On("FindExternalBy", mock.Anything, mock.Anything).Return([]*user.ExternalUser{}, nil)
				mockRepo.On("CreateExternal", mock.Anything, mock.Anything).Return(errors.New("random error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.CreateExternal(ctx, "project", "user", "external", "provider")
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExternal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("CreateExternal() error = %v, wantErr %v", err, tt.err)
				return
			}
		})
	}
}
