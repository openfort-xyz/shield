package projectsvc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestService_Create(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo, 60*time.Second)
	ctx := context.Background()
	testName := "test-project"

	tc := []struct {
		name    string
		wantErr bool
		param   string
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(nil)
			},
		},
		{
			name:    "repository error on create",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*project.Project")).Return(errors.New("repository error"))
			},
		},
		{
			name: "failed to encrypt secret",
			mock: func() {
				svc.(*service).cost = 9999
			},
			wantErr: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			result, err := svc.Create(ctx, testName, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Errorf("Create() expected a project but got nil")
			}
		})
	}
}

func TestService_SetEncryptionKey(t *testing.T) {
	mockRepo := new(projectmockrepo.MockProjectRepository)
	svc := New(mockRepo, 60*time.Second)
	ctx := context.Background()
	testProjectID := "test-project-id"
	testKey := &project.EncryptionKey{Part: "test-part"}

	tc := []struct {
		name    string
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetEncryptionKey", mock.Anything, testProjectID).Return(nil, domainErrors.ErrEncryptionPartNotFound)
				mockRepo.On("SetEncryptionKey", mock.Anything, testProjectID, testKey).Return(nil)
			},
		},
		{
			name:    "encryption part already exists",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetEncryptionKey", mock.Anything, testProjectID).Return(&project.EncryptionKey{Part: project.StoredPart("test-encryption-part")}, nil)
			},
			err: domainErrors.ErrEncryptionPartAlreadyExists,
		},
		{
			name:    "repository error on get encryption part",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetEncryptionKey", mock.Anything, testProjectID).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "repository error on set encryption part",
			wantErr: true,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetEncryptionKey", mock.Anything, testProjectID).Return(nil, domainErrors.ErrEncryptionPartNotFound)
				mockRepo.On("SetEncryptionKey", mock.Anything, testProjectID, testKey).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.SetEncryptionKey(ctx, testProjectID, testKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetEncryptionKey() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("SetEncryptionKey() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func TestService_CommitAPIKeyRotation(t *testing.T) {
	future := time.Now().Add(time.Minute)
	past := time.Now().Add(-time.Minute)
	pending := "pending-key"

	tests := []struct {
		name          string
		state         project.APIKeyRotation
		presented     string
		wantSwap      bool
		wantCommitted bool
		wantErr       error
	}{
		{"pending key commits", project.APIKeyRotation{APIKey: "old", PendingAPIKey: &pending, PendingExpiresAt: &future}, pending, true, true, nil},
		{"current key with nothing pending", project.APIKeyRotation{APIKey: "old"}, "old", false, false, nil},
		{"current key while pending", project.APIKeyRotation{APIKey: "old", PendingAPIKey: &pending, PendingExpiresAt: &future}, "old", false, false, domainErrors.ErrPendingAPIKeyNotProven},
		{"current key after expiry", project.APIKeyRotation{APIKey: "old", PendingAPIKey: &pending, PendingExpiresAt: &past}, "old", false, false, domainErrors.ErrAPIKeyRotationExpired},
		{"expired pending key", project.APIKeyRotation{APIKey: "old", PendingAPIKey: &pending, PendingExpiresAt: &past}, pending, false, false, domainErrors.ErrAPIKeyRotationExpired},
		{"replaced pending key", project.APIKeyRotation{APIKey: "old", PendingAPIKey: &pending, PendingExpiresAt: &future}, "earlier-pending", false, false, domainErrors.ErrNoPendingAPIKeyRotation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(projectmockrepo.MockProjectRepository)
			var swapped bool
			repo.On("CommitAPIKeyRotation", mock.Anything, "project", mock.Anything).Run(func(args mock.Arguments) {
				commit := args.Get(2).(project.CommitAPIKeyRotationFunc)
				var err error
				swapped, err = commit(tt.state)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("decision err = %v, want %v", err, tt.wantErr)
				}
			}).Return(tt.wantErr)

			_, committed, err := New(repo, time.Minute).CommitAPIKeyRotation(context.Background(), "project", tt.presented)

			if swapped != tt.wantSwap || committed != tt.wantCommitted || !errors.Is(err, tt.wantErr) {
				t.Errorf("swap=%v committed=%v err=%v, want %v %v %v", swapped, committed, err, tt.wantSwap, tt.wantCommitted, tt.wantErr)
			}
		})
	}
}

// A lookup that read the old key before a commit evicted it must not put it
// back into the cache afterwards.
func TestService_GetByAPIKey_DoesNotRecacheAcrossEviction(t *testing.T) {
	repo := new(projectmockrepo.MockProjectRepository)
	svc := New(repo, time.Minute)
	pending := "new-key"
	future := time.Now().Add(time.Minute)
	proj := &project.Project{ID: "project", APIKey: "old-key"}

	reading, release := make(chan struct{}), make(chan struct{})
	repo.On("GetByAPIKey", mock.Anything, "old-key").Run(func(mock.Arguments) {
		close(reading)
		<-release
	}).Return(proj, nil).Once()
	repo.On("CommitAPIKeyRotation", mock.Anything, "project", mock.Anything).Run(func(args mock.Arguments) {
		_, _ = args.Get(2).(project.CommitAPIKeyRotationFunc)(project.APIKeyRotation{APIKey: "old-key", PendingAPIKey: &pending, PendingExpiresAt: &future})
	}).Return(nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.GetByAPIKey(context.Background(), "old-key")
	}()
	<-reading
	if _, committed, err := svc.CommitAPIKeyRotation(context.Background(), "project", pending); err != nil || !committed {
		t.Fatalf("commit: committed=%v err=%v", committed, err)
	}
	close(release)
	<-done

	repo.On("GetByAPIKey", mock.Anything, "old-key").Return(nil, domainErrors.ErrProjectNotFound).Once()
	if _, err := svc.GetByAPIKey(context.Background(), "old-key"); !errors.Is(err, domainErrors.ErrProjectNotFound) {
		t.Errorf("old key after rotation: err = %v, want %v (served from a re-populated cache)", err, domainErrors.ErrProjectNotFound)
	}
}

// secretAndHash returns a hex-encoded secret and its bcrypt hash, as stored.
func secretAndHash(t *testing.T, seed byte) (string, string) {
	t.Helper()
	raw := bytes.Repeat([]byte{seed}, 32)
	hash, err := bcrypt.GenerateFromPassword(raw, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(raw), string(hash)
}

func TestService_CommitAPISecretRotation(t *testing.T) {
	future := time.Now().Add(time.Minute)
	past := time.Now().Add(-time.Minute)
	current, currentHash := secretAndHash(t, 1)
	pending, pendingHash := secretAndHash(t, 2)
	other, _ := secretAndHash(t, 3)

	withPending := func(expires *time.Time) project.APISecretRotation {
		return project.APISecretRotation{APIKey: "key", APISecretHash: currentHash, PendingHash: &pendingHash, PendingExpiresAt: expires}
	}
	tests := []struct {
		name      string
		state     project.APISecretRotation
		key       string
		presented string
		wantSwap  bool
		wantErr   error
	}{
		{"pending secret commits", withPending(&future), "key", pending, true, nil},
		{"current secret with nothing pending", project.APISecretRotation{APIKey: "key", APISecretHash: currentHash}, "key", current, false, nil},
		{"current secret while pending", withPending(&future), "key", current, false, domainErrors.ErrPendingAPISecretNotProven},
		{"current secret after expiry", withPending(&past), "key", current, false, domainErrors.ErrAPISecretRotationExpired},
		{"expired pending secret", withPending(&past), "key", pending, false, domainErrors.ErrAPISecretRotationExpired},
		{"replaced pending secret", withPending(&future), "key", other, false, domainErrors.ErrNoPendingAPISecretRotation},
		{"key rotated since authentication", withPending(&future), "old-key", pending, false, domainErrors.ErrAPIKeyRevoked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(projectmockrepo.MockProjectRepository)
			var swapped bool
			repo.On("CommitAPISecretRotation", mock.Anything, "project", mock.Anything).Run(func(args mock.Arguments) {
				commit := args.Get(2).(project.CommitAPISecretRotationFunc)
				var err error
				swapped, err = commit(tt.state)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("decision err = %v, want %v", err, tt.wantErr)
				}
			}).Return(tt.wantErr)

			committed, err := New(repo, time.Minute).CommitAPISecretRotation(context.Background(), "project", tt.key, tt.presented)

			if swapped != tt.wantSwap || committed != tt.wantSwap || !errors.Is(err, tt.wantErr) {
				t.Errorf("swap=%v committed=%v err=%v, want %v %v", swapped, committed, err, tt.wantSwap, tt.wantErr)
			}
		})
	}
}

func TestService_PrepareAPISecretRotation_StoresHashNotSecret(t *testing.T) {
	repo := new(projectmockrepo.MockProjectRepository)
	var stored string
	repo.On("SetPendingAPISecret", mock.Anything, "project", "key", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		stored = args.String(3)
	}).Return(nil)

	secret, err := New(repo, time.Minute).PrepareAPISecretRotation(context.Background(), "project", "key", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	raw, err := hex.DecodeString(secret)
	if err != nil || len(raw) != 32 {
		t.Fatalf("secret %q is not hex-encoded 32 bytes", secret)
	}
	if stored == secret || bcrypt.CompareHashAndPassword([]byte(stored), raw) != nil {
		t.Fatalf("stored %q is not a bcrypt hash of the returned secret", stored)
	}
}

func TestService_PrepareAPISecretRotation_RevokedKey(t *testing.T) {
	repo := new(projectmockrepo.MockProjectRepository)
	repo.On("SetPendingAPISecret", mock.Anything, "project", "old-key", mock.Anything, mock.Anything).Return(domainErrors.ErrAPIKeyRevoked)

	secret, err := New(repo, time.Minute).PrepareAPISecretRotation(context.Background(), "project", "old-key", time.Now().Add(time.Minute))
	if secret != "" || !errors.Is(err, domainErrors.ErrAPIKeyRevoked) {
		t.Fatalf("got %q, %v; want no secret and ErrAPIKeyRevoked", secret, err)
	}
}
