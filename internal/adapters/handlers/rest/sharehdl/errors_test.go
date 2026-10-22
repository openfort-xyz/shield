package sharehdl

import (
	"net/http"
	"testing"

	"github.com/openfort-xyz/shield/internal/applications/shareapp"
)

// A share write that raced a key rotation must tell the client to retry
// with the current part, not look like a server failure.
func TestFromApplicationError_EncryptionPartChanged(t *testing.T) {
	got := fromApplicationError(shareapp.ErrEncryptionPartChanged)
	if got.Status != http.StatusConflict || got.Code != "EC_ROTATED" {
		t.Errorf("fromApplicationError(ErrEncryptionPartChanged) = %d %s, want 409 EC_ROTATED", got.Status, got.Code)
	}
}

func TestFromApplicationError_ProjectEntropyImport(t *testing.T) {
	got := fromApplicationError(shareapp.ErrProjectEntropyImport)
	if got.Status != http.StatusBadRequest || got.Code != "SH_INVALID" {
		t.Errorf("fromApplicationError(ErrProjectEntropyImport) = %d %s, want 400 SH_INVALID", got.Status, got.Code)
	}
}
