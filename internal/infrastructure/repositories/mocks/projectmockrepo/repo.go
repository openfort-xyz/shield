package projectrepomock

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
	args := m.Called(ctx, proj)
	return args.Error(0)
}

func (m *MockProjectRepository) Get(ctx context.Context, projectID string) (*project.Project, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

func (m *MockProjectRepository) GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}
