package projectmockrepo

import (
	"context"
	"time"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/stretchr/testify/mock"
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

func (m *MockProjectRepository) HardDelete(ctx context.Context, projectID string) error {
	args := m.Mock.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockProjectRepository) GetEncryptionKey(ctx context.Context, projectID string) (*project.EncryptionKey, error) {
	args := m.Mock.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.EncryptionKey), args.Error(1)
}

func (m *MockProjectRepository) SetEncryptionKey(ctx context.Context, projectID string, key *project.EncryptionKey) error {
	args := m.Mock.Called(ctx, projectID, key)
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

func (m *MockProjectRepository) PrepareEncryptionKeyRotation(ctx context.Context, projectID string, prepare project.PrepareRotationFunc) error {
	args := m.Mock.Called(ctx, projectID, prepare)
	return args.Error(0)
}

func (m *MockProjectRepository) CommitEncryptionKeyRotation(ctx context.Context, projectID string, commit project.CommitRotationFunc) error {
	args := m.Mock.Called(ctx, projectID, commit)
	return args.Error(0)
}

func (m *MockProjectRepository) SetPendingAPIKey(ctx context.Context, projectID, currentKey, apiKey string, expiresAt time.Time) error {
	args := m.Mock.Called(ctx, projectID, currentKey, apiKey, expiresAt)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByPendingAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	args := m.Mock.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

func (m *MockProjectRepository) CommitAPIKeyRotation(ctx context.Context, projectID string, commit project.CommitAPIKeyRotationFunc) error {
	args := m.Mock.Called(ctx, projectID, commit)
	return args.Error(0)
}
