package shareapp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/adapters/encryption"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/encryptionpartsmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/keychainmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/sharemockrepo"
	"go.openfort.xyz/shield/internal/applications/shamirjob"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/keychain"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/services/sharesvc"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/random"
)

func TestShareApplication_GetShare(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	keychainRepo := new(keychainmockrepo.MockKeychainRepository)
	shareSvc := sharesvc.New(shareRepo, keychainRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, keychainRepo, encryptionFactory, &shamirjob.Job{})
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf(key)
	}

	reconstructor := encryptionFactory.CreateReconstructionStrategy(true)
	storedPart, projectPart, err := reconstructor.Split(key)
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}

	cypher := encryptionFactory.CreateEncryptionStrategy(key)
	encryptedSecret, err := cypher.Encrypt("secret")
	if err != nil {
		t.Fatalf("failed to cypher secret: %v", err)
	}

	plainShare := &share.Share{
		Secret:  "secret",
		Entropy: share.EntropyNone,
	}
	encryptedShare := &share.Share{
		Secret:  encryptedSecret,
		Entropy: share.EntropyProject,
	}
	decryptedShare := &share.Share{
		Secret:  "secret",
		Entropy: share.EntropyProject,
	}

	testKeychainID := uuid.NewString()
	testKeychain := &keychain.Keychain{
		ID:     testKeychainID,
		UserID: "user_id",
	}

	tc := []struct {
		name    string
		opts    []Option
		wantErr error
		want    *share.Share
		mock    func()
	}{
		{
			name:    "success",
			wantErr: nil,
			want:    plainShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(plainShare, nil)
			},
		},
		{
			name:    "encrypted success",
			wantErr: nil,
			want:    decryptedShare,
			mock: func() {
				tmpEncryptedShare := *encryptedShare
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&tmpEncryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart(projectPart),
			},
		},
		{
			name:    "encrypted success with session",
			wantErr: nil,
			want:    decryptedShare,
			mock: func() {
				tmpEncryptedShare := *encryptedShare
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				encryptionPartsRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&tmpEncryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
				encryptionPartsRepo.On("Get", mock.Anything, "sessionID").Return(projectPart, nil)
				encryptionPartsRepo.On("Delete", mock.Anything, "sessionID").Return(nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)

			},
			opts: []Option{
				WithEncryptionSession("sessionID"),
			},
		},
		{
			name:    "encryption part required",
			wantErr: ErrEncryptionPartRequired,
			mock: func() {
				tmpEncryptedShare := *encryptedShare
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&tmpEncryptedShare, nil)
			},
		},
		{
			name:    "encryption not configured",
			wantErr: ErrEncryptionNotConfigured,
			mock: func() {
				tmpEncryptedShare := *encryptedShare
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&tmpEncryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domainErrors.ErrEncryptionPartNotFound)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)

			},
			opts: []Option{
				WithEncryptionPart(projectPart),
			},
		},
		{
			name:    "invalid encryption part",
			wantErr: ErrInvalidEncryptionPart,
			mock: func() {
				tmpEncryptedShare := *encryptedShare
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&tmpEncryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart("invalid-key"),
			},
		},
		{
			name:    "decryption error",
			wantErr: ErrInternal,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(decryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart(projectPart),
			},
		},
		{
			name:    "share not found",
			wantErr: ErrShareNotFound,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, domainErrors.ErrShareNotFound)
				shareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				keychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
			},
		},
		{
			name:    "get share repository error",
			wantErr: ErrInternal,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "get encryption part repository error",
			wantErr: ErrInternal,
			mock: func() {
				tmpEncryptedShare := *encryptedShare
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&tmpEncryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", errors.New("repository error"))
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart(projectPart),
			},
		},
		{
			name:    "encryption part not found",
			wantErr: ErrEncryptionNotConfigured,
			mock: func() {
				tmpEncryptedShare := *encryptedShare
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&tmpEncryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domainErrors.ErrEncryptionPartNotFound)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart(projectPart),
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			s, err := app.GetShare(ctx, tt.opts...)
			ass.ErrorIs(tt.wantErr, err)
			ass.Equal(tt.want, s)
		})
	}
}

func TestShareApplication_GetShareEncryption(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")

	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	keychainRepo := new(keychainmockrepo.MockKeychainRepository)
	shareSvc := sharesvc.New(shareRepo, keychainRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, keychainRepo, encryptionFactory, &shamirjob.Job{})

	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf(key)
	}

	// Create a project-based share (no encryption details)
	projectShare := &share.Share{
		Secret:  "project-secret",
		Entropy: share.EntropyProject,
	}
	// Create a user-based share (encryption details must include pbkdf2 config details)

	encryptionInfo := share.EncryptionParameters{
		Digest:     "sha256",
		Length:     256,
		Salt:       "ipebre",
		Iterations: 1337,
	}
	userShare := &share.Share{
		Secret:               "user-secret",
		Entropy:              share.EntropyUser,
		EncryptionParameters: &encryptionInfo,
	}

	// Create a none-entropy share
	noneShare := &share.Share{
		Secret:  "none-secret",
		Entropy: share.EntropyNone,
	}

	tc := []struct {
		name        string
		wantErr     error
		wantEntropy share.Entropy
		wantDetails *share.EncryptionParameters
		mock        func()
	}{
		{
			name:        "Get project-based share",
			wantErr:     nil,
			wantEntropy: share.EntropyProject,
			wantDetails: nil,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(projectShare, nil)
			},
		},
		{
			name:        "Get user-based share",
			wantErr:     nil,
			wantEntropy: share.EntropyUser,
			wantDetails: &encryptionInfo,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(userShare, nil)
			},
		},
		{
			name:        "Get none-entropy share",
			wantErr:     nil,
			wantEntropy: share.EntropyNone,
			wantDetails: nil,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(noneShare, nil)
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			ent, det, err := app.GetShareEncryption(ctx)
			ass.ErrorIs(tt.wantErr, err)
			ass.Equal(tt.wantEntropy, ent)
			ass.Equal(tt.wantDetails, det)
		})
	}
}

func TestShareApplication_RegisterShare(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	keychainRepo := new(keychainmockrepo.MockKeychainRepository)
	shareSvc := sharesvc.New(shareRepo, keychainRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, keychainRepo, encryptionFactory, &shamirjob.Job{})
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf(key)
	}

	storedPart, projectPart, err := encryptionFactory.CreateReconstructionStrategy(true).Split(key)
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}

	cypher := encryptionFactory.CreateEncryptionStrategy(key)
	encryptedSecret, err := cypher.Encrypt("secret")
	if err != nil {
		t.Fatalf("failed to cypher secret: %v", err)
	}

	plainShare := &share.Share{
		Secret:  "secret",
		UserID:  "user_id",
		Entropy: share.EntropyNone,
	}
	encryptedShare := &share.Share{
		Secret:  encryptedSecret,
		UserID:  "user_id",
		Entropy: share.EntropyProject,
	}

	testKeychainID := uuid.NewString()
	testKeychain := &keychain.Keychain{
		ID:     testKeychainID,
		UserID: "user_id",
	}

	tc := []struct {
		name    string
		opts    []Option
		share   *share.Share
		wantErr error
		mock    func()
	}{
		{
			name:    "success",
			wantErr: nil,
			share:   plainShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				keychainRepo.ExpectedCalls = nil
				shareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				keychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				shareRepo.On("Create", mock.Anything, plainShare).Return(nil)
			},
		},
		{
			name:    "encrypted success",
			wantErr: nil,
			share:   encryptedShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				keychainRepo.ExpectedCalls = nil
				shareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				keychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				shareRepo.On("Create", mock.Anything, encryptedShare).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart(projectPart),
			},
		},
		{
			name:    "encryption part required",
			wantErr: ErrEncryptionPartRequired,
			share:   encryptedShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
			},
		},
		{
			name:    "encryption not configured",
			wantErr: ErrEncryptionNotConfigured,
			share:   encryptedShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domainErrors.ErrEncryptionPartNotFound)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart(projectPart),
			},
		},
		{
			name:    "invalid encryption part",
			wantErr: ErrInvalidEncryptionPart,
			share:   encryptedShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
				projectRepo.On("HasSuccessfulMigration", mock.Anything, "project_id").Return(true, nil)
			},
			opts: []Option{
				WithEncryptionPart("invalid-key"),
			},
		},
		{
			name:    "share already exists",
			wantErr: ErrShareAlreadyExists,
			share:   plainShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				keychainRepo.ExpectedCalls = nil
				shareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(plainShare, nil)
				shareRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
				keychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(plainShare, nil)
			},
		},
		{
			name:    "repository error",
			wantErr: ErrInternal,
			share:   plainShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			err := app.RegisterShare(ctx, tt.share, tt.opts...)
			ass.ErrorIs(tt.wantErr, err)
		})
	}
}

func TestShareApplication_DeleteShare(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	keychainRepo := new(keychainmockrepo.MockKeychainRepository)
	shareSvc := sharesvc.New(shareRepo, keychainRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, keychainRepo, encryptionFactory, &shamirjob.Job{})

	testKeychain := &keychain.Keychain{
		ID:     "test_keychain",
		UserID: "user_id",
	}

	reference := "some-reference"

	tc := []struct {
		name      string
		wantErr   error
		mock      func()
		reference *string
	}{
		{
			name:    "success",
			wantErr: nil,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&share.Share{ID: "share-id"}, nil)
				shareRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:    "success (by reference)",
			wantErr: nil,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&share.Share{ID: "share-id", KeychainID: new(string)}, nil)
				shareRepo.On("GetByReference", mock.Anything, reference, mock.Anything).Return(&share.Share{ID: "share-id"}, nil)
				shareRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
			},
			reference: &reference,
		},
		{
			name:    "share not found",
			wantErr: ErrShareNotFound,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				keychainRepo.ExpectedCalls = nil
				shareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				keychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, domainErrors.ErrShareNotFound)
			},
		},
		{
			name:    "share not found (by reference)",
			wantErr: ErrShareNotFound,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				keychainRepo.ExpectedCalls = nil
				shareRepo.On("GetByReference", mock.Anything, reference, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				keychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, domainErrors.ErrShareNotFound)
			},
			reference: &reference,
		},
		{
			name:    "repository error",
			wantErr: ErrInternal,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "delete error",
			wantErr: ErrInternal,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&share.Share{ID: "share-id"}, nil)
				shareRepo.On("Delete", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := app.DeleteShare(ctx, tt.reference)
			assert.ErrorIs(t, tt.wantErr, err)
		})
	}
}

func TestShareApplication_UpdateShare(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	keychainRepo := new(keychainmockrepo.MockKeychainRepository)
	shareSvc := sharesvc.New(shareRepo, keychainRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, keychainRepo, encryptionFactory, &shamirjob.Job{})
	updates := &share.Share{
		ID:                   "share-id",
		Secret:               "secret",
		UserID:               "user_id",
		EncryptionParameters: nil,
	}

	testKeychain := &keychain.Keychain{
		ID:     "test_keychain",
		UserID: "user_id",
	}

	tc := []struct {
		name    string
		wantErr error
		mock    func()
		updates *share.Share
	}{
		{
			name:    "success",
			wantErr: nil,
			updates: updates,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&share.Share{ID: "share-id"}, nil)
				shareRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:    "share not found",
			wantErr: ErrShareNotFound,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				keychainRepo.ExpectedCalls = nil
				shareRepo.On("GetByReference", mock.Anything, mock.Anything, mock.Anything).Return(nil, domainErrors.ErrShareNotFound)
				keychainRepo.On("GetByUserID", mock.Anything, mock.Anything).Return(testKeychain, nil)
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, domainErrors.ErrShareNotFound)
			},
		},
		{
			name:    "repository error",
			wantErr: ErrInternal,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "delete error",
			updates: updates,
			wantErr: ErrInternal,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(&share.Share{ID: "share-id"}, nil)
				shareRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			ass := assert.New(t)
			_, err := app.UpdateShare(ctx, tt.updates, "default")
			ass.ErrorIs(tt.wantErr, err)
		})
	}
}
