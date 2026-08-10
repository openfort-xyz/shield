package projectapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/encryption"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/encryptionpartsmockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/notificationsmockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/providermockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/sharemockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/usercontactmockrepo"
	"github.com/openfort-xyz/shield/internal/core/services/projectsvc"
	"github.com/openfort-xyz/shield/internal/core/services/providersvc"
	"github.com/openfort-xyz/shield/pkg/contexter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProjectApplication_DeleteProject(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	providerRepo := new(providermockrepo.MockProviderRepository)
	notificationsRepo := new(notificationsmockrepo.MockNotificationsRepository)
	userContactRepo := new(usercontactmockrepo.MockUserContactRepository)
	projectService := projectsvc.New(projectRepo, 60*time.Second)
	providerService := providersvc.New(providerRepo)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	rateLimiter := NewRequestTracker(&TestClock{})
	app := New(projectService, projectRepo, providerService, providerRepo, shareRepo, notificationsRepo, userContactRepo, encryptionFactory, encryptionPartsRepo, nil, nil, rateLimiter)

	tc := []struct {
		name    string
		wantErr bool
		mock    func()
	}{
		{
			name: "success",
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("HardDelete", mock.Anything, "project_id").Return(nil)
			},
		},
		{
			name:    "repository error",
			wantErr: true,
			mock: func() {
				projectRepo.ExpectedCalls = nil
				projectRepo.On("HardDelete", mock.Anything, "project_id").Return(errors.New("boom"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := app.DeleteProject(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			projectRepo.AssertCalled(t, "HardDelete", mock.Anything, "project_id")
		})
	}
}
