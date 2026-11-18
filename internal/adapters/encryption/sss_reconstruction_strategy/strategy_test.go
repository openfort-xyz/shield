package sssrec

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/pkg/random"
)

func split(t *testing.T) (key string, stored, proj []byte) {
	t.Helper()
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatal(err)
	}
	storedPart, projectPart, err := NewSSSReconstructionStrategy().Split(key)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	return key, decode(t, string(storedPart)), decode(t, string(projectPart))
}

func decode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func reconstruct(t *testing.T, stored, proj []byte) (key string, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Reconstruct panicked: %v", r)
		}
	}()
	return NewSSSReconstructionStrategy().Reconstruct(project.StoredPart(encode(stored)), project.ProjectPart(encode(proj)))
}

func TestReconstruct_RejectsMalformedProjectPart(t *testing.T) {
	_, stored, proj := split(t)

	sameCoordinate := bytes.Clone(proj)
	sameCoordinate[0] = stored[0]

	tests := []struct {
		name string
		part []byte
	}{
		{"coordinate of the stored part", sameCoordinate},
		{"longer than a part", append(bytes.Clone(proj), 0)},
		{"shorter than a part", proj[:20]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := reconstruct(t, stored, tt.part); !errors.Is(err, domainErrors.ErrInvalidEncryptionPart) {
				t.Fatalf("err = %v, want ErrInvalidEncryptionPart", err)
			}
		})
	}
}

func TestReconstruct_ValidParts(t *testing.T) {
	key, stored, proj := split(t)

	tests := []struct {
		name   string
		stored []byte
		proj   []byte
	}{
		{"current parts", stored, proj},
		{"legacy 32-byte parts", stored[1:], proj[1:]},
		{"legacy stored part, current project part", stored[1:], proj},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reconstruct(t, tt.stored, tt.proj)
			if err != nil || got != key {
				t.Fatalf("Reconstruct = %q, %v, want the split key", got, err)
			}
		})
	}
}
