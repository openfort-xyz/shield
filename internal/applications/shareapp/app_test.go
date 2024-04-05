package shareapp

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/services/sharesvc"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/projectmockrepo"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/mocks/sharemockrepo"
	"go.openfort.xyz/shield/pkg/contexter"
	"go.openfort.xyz/shield/pkg/cypher"
	"testing"
)

func TestShareApplication_GetShare(t *testing.T) {
	ctx := contexter.WithProjectID(context.Background(), "project_id")
	ctx = contexter.WithUserID(ctx, "user_id")
	shareRepo := new(sharemockrepo.MockShareRepository)
	projectRepo := new(projectmockrepo.MockProjectRepository)
	shareSvc := sharesvc.New(shareRepo)
	app := New(shareSvc, shareRepo, projectRepo)
	storedPart, externalPart, err := cypher.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}
	encryptionKey, err := cypher.ReconstructEncryptionKey(storedPart, externalPart)
	if err != nil {
		t.Fatalf("failed to reconstruct encryption key: %v", err)
	}

	encryptedSecret, err := cypher.Encrypt("secret", encryptionKey)
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}

	plainShare := &share.Share{
		Secret: "secret",
		EncryptionParameters: &share.EncryptionParameters{
			Entropy: share.EntropyNone,
		},
	}
	encryptedShare := &share.Share{
		Secret: encryptedSecret,
		EncryptionParameters: &share.EncryptionParameters{
			Entropy: share.EntropyProject,
		},
	}
	decryptedShare := &share.Share{
		Secret: "secret",
		EncryptionParameters: &share.EncryptionParameters{
			Entropy: share.EntropyProject,
		},
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
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(encryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
			},
			opts: []Option{
				WithEncryptionPart(externalPart),
			},
		},
		{
			name:    "encryption part required",
			wantErr: ErrEncryptionPartRequired,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(encryptedShare, nil)
			},
		},
		{
			name:    "encryption not configured",
			wantErr: ErrEncryptionNotConfigured,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(encryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domain.ErrEncryptionPartNotFound)
			},
			opts: []Option{
				WithEncryptionPart(externalPart),
			},
		},
		{
			name:    "invalid encryption part",
			wantErr: ErrInvalidEncryptionPart,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(encryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
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
			},
			opts: []Option{
				WithEncryptionPart(externalPart),
			},
		},
		{
			name:    "share not found",
			wantErr: ErrShareNotFound,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(nil, domain.ErrShareNotFound)
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
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(encryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", errors.New("repository error"))
			},
			opts: []Option{
				WithEncryptionPart(externalPart),
			},
		},
		{
			name:    "encryption part not found",
			wantErr: ErrEncryptionNotConfigured,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, "user_id").Return(encryptedShare, nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domain.ErrEncryptionPartNotFound)
			},
			opts: []Option{
				WithEncryptionPart(externalPart),
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
	shareSvc := sharesvc.New(shareRepo)
	app := New(shareSvc, shareRepo, projectRepo)
	storedPart, externalPart, err := cypher.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}
	encryptionKey, err := cypher.ReconstructEncryptionKey(storedPart, externalPart)
	if err != nil {
		t.Fatalf("failed to reconstruct encryption key: %v", err)
	}

	encryptedSecret, err := cypher.Encrypt("secret", encryptionKey)
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}

	plainShare := &share.Share{
		Secret: "secret",
		UserID: "user_id",
		EncryptionParameters: &share.EncryptionParameters{
			Entropy: share.EntropyNone,
		},
	}
	encryptedShare := &share.Share{
		Secret: encryptedSecret,
		UserID: "user_id",
		EncryptionParameters: &share.EncryptionParameters{
			Entropy: share.EntropyProject,
		},
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
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrShareNotFound)
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
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrShareNotFound)
				shareRepo.On("Create", mock.Anything, encryptedShare).Return(nil)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
			},
			opts: []Option{
				WithEncryptionPart(externalPart),
			},
		},
		{
			name:    "encryption part required",
			wantErr: ErrEncryptionPartRequired,
			share:   encryptedShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrShareNotFound)
			},
		},
		{
			name:    "encryption not configured",
			wantErr: ErrEncryptionNotConfigured,
			share:   encryptedShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrShareNotFound)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return("", domain.ErrEncryptionPartNotFound)
			},
			opts: []Option{
				WithEncryptionPart(externalPart),
			},
		},
		{
			name:    "invalid encryption part",
			wantErr: ErrInvalidEncryptionPart,
			share:   encryptedShare,
			mock: func() {
				shareRepo.ExpectedCalls = nil
				projectRepo.ExpectedCalls = nil
				shareRepo.On("GetByUserID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrShareNotFound)
				projectRepo.On("GetEncryptionPart", mock.Anything, "project_id").Return(storedPart, nil)
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
