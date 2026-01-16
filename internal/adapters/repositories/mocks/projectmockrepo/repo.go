package projectmockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type MockProjectRepository struct {
	mock.Mock
}

var _ repositories.ProjectRepository = (*MockProjectRepository)(nil)

func (m *MockProjectRepository) Create(ctx context.Context, proj *project.Project) error {
	args := m.Mock.Called(ctx, proj)
	return args.Error(0)
}

func (m *MockProjectRepository) SaveProjectRateLimits(ctx context.Context, rateLimits *project.RateLimit) error {
	args := m.Mock.Called(ctx, rateLimits)
	return args.Error(0)
}

func (m *MockProjectRepository) Get(ctx context.Context, projectID string) (*project.Project, error) {
	args := m.Mock.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

func (m *MockProjectRepository) GetWithRateLimit(ctx context.Context, projectID string) (*project.WithRateLimit, error) {
	args := m.Mock.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.WithRateLimit), args.Error(1)
}

func (m *MockProjectRepository) UpdateAPISecret(ctx context.Context, projectID, encryptedSecret string) error {
	args := m.Mock.Called(ctx, projectID, encryptedSecret)
	return args.Error(0)
}

func (m *MockProjectRepository) Update2FA(ctx context.Context, projectID string, enable2FA bool) error {
	args := m.Mock.Called(ctx, projectID, enable2FA)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	args := m.Mock.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

func (m *MockProjectRepository) Delete(ctx context.Context, projectID string) error {
	args := m.Mock.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockProjectRepository) GetEncryptionPart(ctx context.Context, projectID string) (string, error) {
	args := m.Mock.Called(ctx, projectID)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}

func (m *MockProjectRepository) SetEncryptionPart(ctx context.Context, projectID, part string) error {
	args := m.Mock.Called(ctx, projectID, part)
	return args.Error(0)
}

func (m *MockProjectRepository) CreateMigration(ctx context.Context, projectID string, success bool) error {
	args := m.Mock.Called(ctx, projectID, success)
	return args.Error(0)
}

func (m *MockProjectRepository) HasSuccessfulMigration(ctx context.Context, projectID string) (bool, error) {
	args := m.Mock.Called(ctx, projectID)
	return args.Bool(0), args.Error(1)
}
