package shareapp

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/adapters/encryption"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/encryptionpartsmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/projectmockrepo"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/sharemockrepo"
	"go.openfort.xyz/shield/internal/applications/shamirjob"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/services/sharesvc"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/random"
	"testing"
)

func TestShareApplication_GetShare(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	shareSvc := sharesvc.New(shareRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, encryptionFactory, &shamirjob.Job{})
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

func TestShareApplication_RegisterShare(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
	encryptionFactory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	shareSvc := sharesvc.New(shareRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, encryptionFactory, &shamirjob.Job{})
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
	shareSvc := sharesvc.New(shareRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, encryptionFactory, &shamirjob.Job{})

	tc := []struct {
		name    string
		wantErr error
		mock    func()
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
			name:    "share not found",
			wantErr: ErrShareNotFound,
			mock: func() {
				shareRepo.ExpectedCalls = nil
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
			ass := assert.New(t)
			err := app.DeleteShare(ctx)
			ass.ErrorIs(tt.wantErr, err)
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
	shareSvc := sharesvc.New(shareRepo, encryptionFactory)
	app := New(shareSvc, shareRepo, projectRepo, encryptionFactory, &shamirjob.Job{})
	updates := &share.Share{
		ID:                   "share-id",
		Secret:               "secret",
		UserID:               "user_id",
		EncryptionParameters: nil,
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
			_, err := app.UpdateShare(ctx, tt.updates)
			ass.ErrorIs(tt.wantErr, err)
		})
	}
}
