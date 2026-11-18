package depsssrec

import (
	"encoding/base64"
	"errors"
	"testing"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/random"
)

// legacyParts splits a 32-byte key the way projects registered before the
// shamir migration were split, and returns the key those projects encrypt
// under.
func legacyParts(t *testing.T) (key, stored, proj string) {
	t.Helper()
	raw, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatal(err)
	}
	stored, proj, err = cypher.SplitEncryptionKey(raw)
	if err != nil {
		t.Fatalf("SplitEncryptionKey: %v", err)
	}
	key, err = cypher.ReconstructEncryptionKey(stored, proj)
	if err != nil {
		t.Fatalf("ReconstructEncryptionKey: %v", err)
	}
	return key, stored, proj
}

func reconstruct(t *testing.T, stored, proj string) (key string, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Reconstruct panicked: %v", r)
		}
	}()
	return NewSSSReconstructionStrategy().Reconstruct(project.StoredPart(stored), project.ProjectPart(proj))
}

func TestReconstruct_RejectsPartOfAnotherLength(t *testing.T) {
	_, stored, _ := legacyParts(t)

	for _, size := range []int{1, 31, 33} {
		part := base64.StdEncoding.EncodeToString(make([]byte, size))
		if _, err := reconstruct(t, stored, part); !errors.Is(err, domainErrors.ErrInvalidEncryptionPart) {
			t.Fatalf("%d-byte part: err = %v, want ErrInvalidEncryptionPart", size, err)
		}
	}
}

func TestReconstruct_LegacyParts(t *testing.T) {
	key, stored, proj := legacyParts(t)

	got, err := reconstruct(t, stored, proj)
	if err != nil || got != key {
		t.Fatalf("Reconstruct = %q, %v, want the legacy key", got, err)
	}
}
