package shareapp

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/encryption"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/encryptionpartsmockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/keychainmockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/sharemockrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/usermockedrepo"
	"github.com/openfort-xyz/shield/internal/applications/shamirjob"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/keychain"
	"github.com/openfort-xyz/shield/internal/core/services/sharesvc"
	"github.com/openfort-xyz/shield/pkg/contexter"
	"github.com/openfort-xyz/shield/pkg/logger/logtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// At the application layer a missing share is what the caller receives as a 404,
// rather than a step inside a larger operation, so it stays visible at the
// default level as a warning. A genuine repository failure keeps its error
// record.
func TestGetKeychainSharesLogsNotFoundAsWarning(t *testing.T) {
	const (
		userID    = "user_id"
		projectID = "project_id"
		reference = "test-reference"
	)

	tests := []struct {
		name         string
		lookupErr    error
		wantErr      error
		wantMessage  string
		wantSeverity string
	}{
		{
			name:         "missing share is a warning",
			lookupErr:    domainErrors.ErrShareNotFound,
			wantErr:      ErrShareNotFound,
			wantMessage:  "keychain share not found by reference",
			wantSeverity: "WARNING",
		},
		{
			name:         "an unexpected repository error is still an error",
			lookupErr:    errors.New("connection refused"),
			wantErr:      ErrInternal,
			wantMessage:  "failed to get share by reference",
			wantSeverity: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.Start(slog.LevelInfo)
			defer rec.Stop()

			keychainID := uuid.NewString()

			shareRepo := new(sharemockrepo.MockShareRepository)
			keychainRepo := new(keychainmockrepo.MockKeychainRepository)
			projectRepo := new(projectmockrepo.MockProjectRepository)
			userRepo := new(usermockedrepo.MockUserRepository)

			// The user already has a keychain and no legacy share, so the call
			// reaches the lookup under test without migrating anything.
			keychainRepo.On("GetByUserID", mock.Anything, userID).
				Return(&keychain.Keychain{ID: keychainID, UserID: userID}, nil)
			shareRepo.On("GetByUserID", mock.Anything, userID).Return(nil, domainErrors.ErrShareNotFound)
			shareRepo.On("GetByReferenceAndKeychain", mock.Anything, reference, keychainID).Return(nil, tt.lookupErr)

			encryptionFactory := encryption.NewEncryptionFactory(
				new(encryptionpartsmockrepo.MockEncryptionPartsRepository), projectRepo)

			// Constructed inside the recorder, since New resolves its destination
			// when it runs.
			app := New(
				sharesvc.New(shareRepo, keychainRepo, encryptionFactory),
				shareRepo, projectRepo, userRepo, keychainRepo, encryptionFactory, &shamirjob.Job{},
			)

			ctx := contexter.WithProjectID(context.Background(), projectID)
			ctx = contexter.WithUserID(ctx, userID)

			ref := reference
			shares, err := app.GetKeychainShares(ctx, &ref)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, shares)

			got, found := rec.Find(tt.wantMessage)
			require.True(t, found, "expected a %q record, got %v", tt.wantMessage, rec.Records())
			assert.Equal(t, tt.wantSeverity, got.Severity)
			assert.Equal(t, "share_application", got.Logger)

			if tt.wantSeverity == "ERROR" {
				assert.Equal(t, tt.lookupErr.Error(), got.Attr("error"),
					"the error record must still carry the underlying cause")
				return
			}

			assert.NotContains(t, rec.Severities(), "ERROR",
				"a missing share must not produce an error record")
			// Both application-layer sites used to log the same text under the same
			// logger name, which made them indistinguishable in a query.
			assert.Equal(t, reference, got.Attr("reference"))
			assert.Equal(t, keychainID, got.Attr("keychain_id"))
		})
	}
}

// The sibling site, which has no keychain in play.
func TestGetShareByReferenceLogsNotFoundAsWarning(t *testing.T) {
	const (
		externalUserID = "external_user_id"
		projectID      = "project_id"
		reference      = "test-reference"
	)

	rec := logtest.Start(slog.LevelInfo)
	defer rec.Stop()

	shareRepo := new(sharemockrepo.MockShareRepository)
	keychainRepo := new(keychainmockrepo.MockKeychainRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	userRepo := new(usermockedrepo.MockUserRepository)
	shareRepo.On("GetByReference", mock.Anything, reference).Return(nil, domainErrors.ErrShareNotFound)

	encryptionFactory := encryption.NewEncryptionFactory(
		new(encryptionpartsmockrepo.MockEncryptionPartsRepository), projectRepo)

	app := New(
		sharesvc.New(shareRepo, keychainRepo, encryptionFactory),
		shareRepo, projectRepo, userRepo, keychainRepo, encryptionFactory, &shamirjob.Job{},
	)

	ctx := contexter.WithProjectID(context.Background(), projectID)
	ctx = contexter.WithExternalUserID(ctx, externalUserID)

	shr, err := app.GetShareByReference(ctx, reference)
	require.ErrorIs(t, err, ErrShareNotFound)
	assert.Nil(t, shr)

	got, found := rec.Find("share not found by reference")
	require.True(t, found, "expected a warning record, got %v", rec.Records())
	assert.Equal(t, "WARNING", got.Severity)
	assert.Equal(t, "share_application", got.Logger)
	assert.Equal(t, reference, got.Attr("reference"))
	assert.NotContains(t, rec.Severities(), "ERROR",
		"a missing share must not produce an error record")
}
