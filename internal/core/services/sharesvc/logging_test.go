package sharesvc

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
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/keychain"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/logger/logtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Registering a share looks the reference up first and expects it to be missing,
// so the miss is the success path and must not be reported as a failure. It stays
// available under LOG_LEVEL=debug, and a genuine repository failure keeps its
// error record.
func TestCreateWithFreeReferenceLogsNoError(t *testing.T) {
	const (
		userID    = "test-user"
		reference = "test-reference"
	)

	keychainID := uuid.NewString()

	tests := []struct {
		name         string
		lookupErr    error
		level        slog.Level
		wantErr      bool
		wantMessage  string
		wantSeverity string
	}{
		{
			name:      "free reference is silent at the default level",
			lookupErr: domainErrors.ErrShareNotFound,
			level:     slog.LevelInfo,
		},
		{
			name:         "free reference is visible at debug",
			lookupErr:    domainErrors.ErrShareNotFound,
			level:        slog.LevelDebug,
			wantMessage:  "share not found by reference",
			wantSeverity: "DEBUG",
		},
		{
			name:         "an unexpected repository error is still an error",
			lookupErr:    errors.New("connection refused"),
			level:        slog.LevelInfo,
			wantErr:      true,
			wantMessage:  "failed to get share by reference",
			wantSeverity: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := logtest.Start(tt.level)
			defer rec.Stop()

			shareRepo := new(sharemockrepo.MockShareRepository)
			keychainRepo := new(keychainmockrepo.MockKeychainRepository)
			shareRepo.On("GetByReference", mock.Anything, reference).Return(nil, tt.lookupErr)
			shareRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
			keychainRepo.On("Get", mock.Anything, keychainID).
				Return(&keychain.Keychain{ID: keychainID, UserID: userID}, nil)

			// Constructed inside the recorder, since New resolves its destination
			// when it runs.
			svc := New(shareRepo, keychainRepo, encryption.NewEncryptionFactory(
				new(encryptionpartsmockrepo.MockEncryptionPartsRepository),
				new(projectmockrepo.MockProjectRepository),
			))

			ref := reference
			err := svc.Create(context.Background(), &share.Share{
				UserID:     userID,
				Secret:     "test-data",
				KeychainID: &keychainID,
				Reference:  &ref,
			})

			if tt.wantErr {
				require.ErrorIs(t, err, tt.lookupErr, "the returned error must not change")
			} else {
				require.NoError(t, err)
				assert.NotContains(t, rec.Severities(), "ERROR",
					"registering a share with a free reference must not log an error")
			}

			if tt.wantMessage == "" {
				_, found := rec.Find("share not found by reference")
				assert.False(t, found, "the outcome must be filtered out at this level")
				return
			}

			got, found := rec.Find(tt.wantMessage)
			require.True(t, found, "expected a %q record, got %v", tt.wantMessage, rec.Records())
			assert.Equal(t, tt.wantSeverity, got.Severity)
			assert.Equal(t, "share_service", got.Logger)

			if tt.wantSeverity == "ERROR" {
				assert.Equal(t, tt.lookupErr.Error(), got.Attr("error"),
					"the error record must still carry the underlying cause")
			} else {
				assert.Equal(t, reference, got.Attr("reference"))
			}
		})
	}
}
