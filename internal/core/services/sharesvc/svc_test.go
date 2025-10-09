package sharesvc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/adapters/encryption"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/encryptionpartsmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/keychainmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/sharemockrepo"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/keychain"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/random"
)

func TestCreateShare(t *testing.T) {
	mockShareRepo := new(sharemockrepo.MockShareRepository)
	mockKeychainRepo := new(keychainmockrepo.MockKeychainRepository)
	mockProjectRepo := new(projectmockrepo.MockProjectRepository)
	mockEncryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)

	encryptionFactory := encryption.NewEncryptionFactory(mockEncryptionPartsRepo, mockProjectRepo)

	svc := New(mockShareRepo, mockKeychainRepo, encryptionFactory)

	ctx := context.Background()
	testUserID := "test-user"
	testData := "test-data"
	testKeychainID := uuid.NewString()
	testReference := "test-reference"

	testKeychain := &keychain.Keychain{
		ID:     testKeychainID,
		UserID: testUserID,
	}

	testShare := share.Share{
		UserID: testUserID,
		Secret: testData,
	}

	testShareWithoutKeychain := share.Share{
		UserID: testUserID,
		Secret: testData,
	}

	testShareWithKeychain := share.Share{
		UserID:     testUserID,
		Secret:     testData,
		KeychainID: &testKeychainID,
		Reference:  &testReference,
	}

	testEncryptionShare := share.Share{
		UserID:  testUserID,
		Secret:  testData,
		Entropy: share.EntropyProject,
	}

	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("Failed to generate random string: %v", err)
	}

	reconstructor := encryptionFactory.CreateReconstructionStrategy(true)
	storedPart, projectPart, err := reconstructor.Split(key)
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}

	encryptionKey, err := reconstructor.Reconstruct(storedPart, projectPart)
	if err != nil {
		t.Fatalf("failed to reconstruct encryption key: %v", err)
	}

	// Test cases
	tc := []struct {
		name    string
		share   share.Share
		opts    []services.ShareOption
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:    "success - without keychain",
			wantErr: false,
			share:   testShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, testUserID).Return(testKeychain, nil)
				mockShareRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
			},
		},
		{
			name:    "success - with keychain",
			wantErr: false,
			share:   testShareWithKeychain,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("Get", mock.Anything, testKeychainID).Return(testKeychain, nil)
				mockShareRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
			},
		},
		{
			name:    "success - keychain creation required",
			wantErr: false,
			share:   testShareWithoutKeychain,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrKeychainNotFound)
				mockKeychainRepo.On("Create", mock.Anything, mock.AnythingOfType("*keychain.Keychain")).Return(nil).Run(func(args mock.Arguments) {
					kc := args.Get(1).(*keychain.Keychain)
					if kc.UserID != testUserID {
						t.Errorf("Expected keychain UserID %s, got %s", testUserID, kc.UserID)
					}
				})
				mockShareRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
			},
		},
		{
			name:    "encryption success",
			share:   testEncryptionShare,
			wantErr: false,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, testUserID).Return(testKeychain, nil)
				mockShareRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
			},
			opts: []services.ShareOption{
				services.WithEncryptionKey(encryptionKey),
			},
		},
		{
			name:    "encryption part required",
			wantErr: true,
			share:   testEncryptionShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
			},
			err: domainErrors.ErrEncryptionPartRequired,
		},
		{
			name:    "encryption error",
			wantErr: true,
			share:   testEncryptionShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)

			},
			opts: []services.ShareOption{
				services.WithEncryptionKey("invalid-key"),
			},
		},
		{
			name:    "share already exists",
			wantErr: true,
			share:   testShare,
			err:     domainErrors.ErrShareAlreadyExists,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(&share.Share{}, nil)
			},
		},
		{
			name:    "keychain validation error - wrong user",
			wantErr: true,
			share:   testShareWithKeychain,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				wrongUserKeychain := &keychain.Keychain{
					ID:     testKeychainID,
					UserID: "different-user-id",
				}
				mockKeychainRepo.On("Get", mock.Anything, testKeychainID).Return(wrongUserKeychain, nil)
				mockShareRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)

			},
			err: domainErrors.ErrKeychainNotFound,
		},
		{
			name:    "repository error on keychain get",
			wantErr: true,
			share:   testShareWithKeychain,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockKeychainRepo.On("Get", mock.Anything, testKeychainID).Return(nil, errors.New("repository error"))
				mockShareRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)

			},
		},
		{
			name:    "repository error on share get",
			wantErr: true,
			share:   testShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "repository error on keychain create",
			wantErr: true,
			share:   testShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrKeychainNotFound)
				mockKeychainRepo.On("Create", mock.Anything, mock.AnythingOfType("*keychain.Keychain")).Return(errors.New("repository error"))
			},
		},
		{
			name:    "repository error on share create",
			wantErr: true,
			share:   testShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(errors.New("repository error"))
				mockShareRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)

			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			shr := tt.share
			err := svc.Create(ctx, &shr, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Create() error = %v, expected error %v", err, tt.err)
			}
		})
	}
}

func TestFindShare(t *testing.T) {
	mockShareRepo := new(sharemockrepo.MockShareRepository)
	mockKeychainRepo := new(keychainmockrepo.MockKeychainRepository)
	mockProjectRepo := new(projectmockrepo.MockProjectRepository)
	mockEncryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(mockEncryptionPartsRepo, mockProjectRepo)

	svc := New(mockShareRepo, mockKeychainRepo, encryptionFactory)

	ctx := context.Background()
	testUserID := "test-user"
	testKeychainID := uuid.NewString()
	testKeychain := &keychain.Keychain{
		ID:     testKeychainID,
		UserID: testUserID,
	}
	testReference := "test-reference"

	testShare := &share.Share{
		UserID: testUserID,
		Secret: "test-secret",
	}

	tc := []struct {
		name       string
		userID     string
		keychainID *string
		reference  *string
		wantErr    bool
		err        error
		expected   *share.Share
		mock       func()
	}{
		{
			name:       "find by user ID",
			userID:     testUserID,
			keychainID: nil,
			reference:  nil,
			wantErr:    false,
			expected:   testShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(testShare, nil)
			},
		},
		{
			name:       "find by keychain ID with default reference",
			userID:     testUserID,
			keychainID: &testKeychainID,
			reference:  nil,
			wantErr:    false,
			expected:   testShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(testShare, nil)
			},
		},
		{
			name:       "find by keychain ID and custom reference",
			userID:     testUserID,
			keychainID: &testKeychainID,
			reference:  &testReference,
			wantErr:    false,
			expected:   testShare,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(testShare, nil)
			},
		},
		{
			name:       "share not found",
			userID:     testUserID,
			keychainID: nil,
			reference:  nil,
			wantErr:    true,
			err:        domainErrors.ErrShareNotFound,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockKeychainRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockShareRepo.On("GetByReferenceAndKeychain", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				mockKeychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
			},
		},
		{
			name:       "repository error",
			userID:     testUserID,
			keychainID: nil,
			reference:  nil,
			wantErr:    true,
			mock: func() {
				mockShareRepo.ExpectedCalls = nil
				mockShareRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			result, err := svc.Find(ctx, tt.userID, tt.keychainID, tt.reference)
			if (err != nil) != tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Find() error = %v, expected error %v", err, tt.err)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("Find() got = %v, want %v", result, tt.expected)
			}
		})
	}
}
