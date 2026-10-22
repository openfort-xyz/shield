package projecthdl

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/openfort-xyz/shield/internal/applications/projectapp"
)

// Clients branch on the status and code of rotation errors.
func TestFromApplicationError_Rotation(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{projectapp.ErrInvalidEncryptionPart, http.StatusBadRequest, "EC_INVALID"},
		{projectapp.ErrNoPendingRotation, http.StatusConflict, "EC_NO_PENDING"},
		{projectapp.ErrRotationExpired, http.StatusConflict, "EC_ROTATION_EXPIRED"},
		{projectapp.ErrProjectNotMigrated, http.StatusConflict, "EC_NOT_MIGRATED"},
		{projectapp.ErrUnverifiableEncryptionKey, http.StatusConflict, "EC_UNVERIFIABLE"},
		{projectapp.ErrPendingAPIKeyNotProven, http.StatusConflict, "A_PENDING_NOT_PROVEN"},
		{projectapp.ErrAPIKeyRotationExpired, http.StatusConflict, "A_ROTATION_EXPIRED"},
		{projectapp.ErrNoPendingAPIKeyRotation, http.StatusConflict, "A_NO_PENDING"},
		{projectapp.ErrEncryptionPartChanged, http.StatusConflict, "EC_ROTATED"},
		{projectapp.ErrEncryptionNotConfigured, http.StatusConflict, "EC_MISSING"},
		{fmt.Errorf("wrapped: %w", projectapp.ErrRotationExpired), http.StatusConflict, "EC_ROTATION_EXPIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			got := fromApplicationError(tt.err)
			if got.Status != tt.status || got.Code != tt.code {
				t.Errorf("fromApplicationError(%v) = %d %s, want %d %s", tt.err, got.Status, got.Code, tt.status, tt.code)
			}
		})
	}
}
