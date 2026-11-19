package usersvc

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/usermockedrepo"
	"github.com/openfort-xyz/shield/internal/core/domain/user"
	"github.com/openfort-xyz/shield/pkg/logger/logtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// A first-time signup is resolved inside the repository, so the service logs an
// error only when resolution itself fails, and keeps the underlying cause.
func TestGetOrCreateLogsOnlyFailures(t *testing.T) {
	const (
		projectID      = "project"
		providerID     = "provider"
		externalUserID = "external"
	)
	resolved := &user.User{ID: "user-id", ProjectID: projectID}

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "resolved user is returned without an error record"},
		{name: "a repository failure is returned and logged", repoErr: errors.New("connection refused")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.Start(slog.LevelInfo)
			defer rec.Stop()

			repo := new(usermockedrepo.MockUserRepository)
			if tt.repoErr != nil {
				repo.On("GetOrCreate", mock.Anything, projectID, externalUserID, providerID).Return(nil, tt.repoErr)
			} else {
				repo.On("GetOrCreate", mock.Anything, projectID, externalUserID, providerID).Return(resolved, nil)
			}

			// Constructed inside the recorder, since New resolves its destination
			// when it runs.
			svc := New(repo)

			got, err := svc.GetOrCreate(context.Background(), projectID, externalUserID, providerID)

			if tt.repoErr == nil {
				require.NoError(t, err)
				assert.Equal(t, resolved, got)
				assert.NotContains(t, rec.Severities(), "ERROR")
				return
			}

			require.ErrorIs(t, err, tt.repoErr, "the returned error must not change")
			record, found := rec.Find("failed to get or create user")
			require.True(t, found, "expected an error record, got %v", rec.Records())
			assert.Equal(t, "ERROR", record.Severity)
			assert.Equal(t, "user_service", record.Logger)
			assert.Equal(t, tt.repoErr.Error(), record.Attr("error"))
		})
	}
}
