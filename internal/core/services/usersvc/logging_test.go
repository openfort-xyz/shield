package usersvc

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/usermockedrepo"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/pkg/logger/logtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// GetOrCreate looks the external user up and creates one when it is missing, so
// a miss is the success path of a first-time signup and must not be reported as a
// failure. It stays available under LOG_LEVEL=debug, and a genuine repository
// failure keeps its error record.
func TestGetOrCreateOnFirstSignupLogsNoError(t *testing.T) {
	const (
		projectID      = "project"
		providerID     = "provider"
		externalUserID = "external"
	)

	tests := []struct {
		name         string
		lookupErr    error
		level        slog.Level
		wantErr      bool
		wantMessage  string
		wantSeverity string
	}{
		{
			name:      "first signup is silent at the default level",
			lookupErr: domainErrors.ErrExternalUserNotFound,
			level:     slog.LevelInfo,
		},
		{
			name:         "first signup is visible at debug",
			lookupErr:    domainErrors.ErrExternalUserNotFound,
			level:        slog.LevelDebug,
			wantMessage:  "external user not found",
			wantSeverity: "DEBUG",
		},
		{
			name:         "an unexpected repository error is still an error",
			lookupErr:    errors.New("connection refused"),
			level:        slog.LevelInfo,
			wantErr:      true,
			wantMessage:  "failed to get user by external ID",
			wantSeverity: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.Start(tt.level)
			defer rec.Stop()

			repo := new(usermockedrepo.MockUserRepository)
			repo.On("FindUserByExternalID", mock.Anything, externalUserID, providerID).Return(nil, tt.lookupErr)
			repo.On("Create", mock.Anything, mock.Anything).Return(nil)
			repo.On("CreateExternal", mock.Anything, mock.Anything).Return(nil)

			// Constructed inside the recorder, since New resolves its destination
			// when it runs.
			svc := New(repo)

			_, err := svc.GetOrCreate(context.Background(), projectID, externalUserID, providerID)

			if tt.wantErr {
				require.ErrorIs(t, err, tt.lookupErr, "the returned error must not change")
			} else {
				require.NoError(t, err)
				assert.NotContains(t, rec.Severities(), "ERROR",
					"a first-time signup must not log an error")
			}

			if tt.wantMessage == "" {
				_, found := rec.Find("external user not found")
				assert.False(t, found, "the outcome must be filtered out at this level")
				return
			}

			got, found := rec.Find(tt.wantMessage)
			require.True(t, found, "expected a %q record, got %v", tt.wantMessage, rec.Records())
			assert.Equal(t, tt.wantSeverity, got.Severity)
			assert.Equal(t, "user_service", got.Logger)

			if tt.wantSeverity == "ERROR" {
				assert.Equal(t, tt.lookupErr.Error(), got.Attr("error"),
					"the error record must still carry the underlying cause")
			} else {
				assert.Equal(t, externalUserID, got.Attr("external_user_id"))
				assert.Equal(t, providerID, got.Attr("provider_id"))
			}
		})
	}
}
