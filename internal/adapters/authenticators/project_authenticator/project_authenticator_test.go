package projauth

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func secretAndHash(t *testing.T, seed byte) (string, *string) {
	t.Helper()
	raw := bytes.Repeat([]byte{seed}, 32)
	hash, err := bcrypt.GenerateFromPassword(raw, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	h := string(hash)
	return hex.EncodeToString(raw), &h
}

func TestProjectAuthenticators_PendingSecret(t *testing.T) {
	current, currentHash := secretAndHash(t, 1)
	pending, pendingHash := secretAndHash(t, 2)
	future, past := time.Now().Add(time.Minute), time.Now().Add(-time.Minute)

	proj := func(expires *time.Time) *project.Project {
		return &project.Project{ID: "project", APIKey: "key", APISecret: *currentHash, PendingAPISecret: pendingHash, PendingAPISecretExpiresAt: expires}
	}
	tests := []struct {
		name    string
		proj    *project.Project
		pending bool // use the pending-secret authenticator
		secret  string
		wantOK  bool
	}{
		{"current authenticator, current secret", proj(&future), false, current, true},
		{"current authenticator rejects the pending secret", proj(&future), false, pending, false},
		{"pending authenticator, pending secret", proj(&future), true, pending, true},
		{"pending authenticator rejects the current secret", proj(&future), true, current, false},
		{"pending authenticator rejects an expired pending secret", proj(&past), true, pending, false},
		{"pending authenticator with nothing pending", &project.Project{ID: "project", APIKey: "key", APISecret: *currentHash}, true, pending, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(projectmockrepo.MockProjectRepository)
			repo.On("GetByAPIKey", mock.Anything, "key").Return(tt.proj, nil)

			newAuth := NewProjectAuthenticator
			if tt.pending {
				newAuth = NewPendingSecretProjectAuthenticator
			}
			auth, err := newAuth(repo, "key", tt.secret).Authenticate(context.Background())

			if ok := err == nil && auth != nil && auth.ProjectID == "project"; ok != tt.wantOK {
				t.Errorf("authenticated = %v (err %v), want %v", ok, err, tt.wantOK)
			}
		})
	}
}

func TestProjectAuthenticators_DisabledProject(t *testing.T) {
	secret, hash := secretAndHash(t, 1)
	other, _ := secretAndHash(t, 2)
	now := time.Now()
	repo := new(projectmockrepo.MockProjectRepository)
	repo.On("GetByAPIKey", mock.Anything, "key").Return(&project.Project{ID: "project", APIKey: "key", APISecret: *hash, DisabledAt: &now}, nil)

	if _, err := NewProjectAuthenticator(repo, "key", secret).Authenticate(context.Background()); !errors.Is(err, domainErrors.ErrProjectDisabled) {
		t.Errorf("disabled project: err = %v, want ErrProjectDisabled", err)
	}
	if _, err := NewProjectAuthenticator(repo, "key", other).Authenticate(context.Background()); err == nil || errors.Is(err, domainErrors.ErrProjectDisabled) {
		t.Errorf("disabled project, wrong secret: err = %v, want a credential error", err)
	}
	if auth, err := NewProjectAuthenticatorAllowingDisabled(repo, "key", secret).Authenticate(context.Background()); err != nil || auth.ProjectID != "project" {
		t.Errorf("allowing disabled: auth = %v, err = %v, want project", auth, err)
	}
}
