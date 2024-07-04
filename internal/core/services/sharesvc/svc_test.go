package sharesvc

import (
	"context"
	"errors"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/cypher"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/sharemockrepo"
	"go.openfort.xyz/shield/internal/core/domain/share"
)

func TestCreateShare(t *testing.T) {
	mockRepo := new(sharemockrepo.MockShareRepository)
	svc := New(mockRepo)
	ctx := context.Background()
	testUserID := "test-user"
	testData := "test-data"
	testShare := &share.Share{
		UserID: testUserID,
		Secret: testData,
	}
	testEncryptionShare := &share.Share{
		UserID: testUserID,
		Secret: testData,
		EncryptionParameters: &share.EncryptionParameters{
			Entropy: share.EntropyProject,
		},
	}
	storedPart, externalPart, err := cypher.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}
	encryptionKey, err := cypher.ReconstructEncryptionKey(storedPart, externalPart)
	if err != nil {
		t.Fatalf("failed to reconstruct encryption key: %v", err)
	}

	tc := []struct {
		name    string
		share   *share.Share
		opts    []services.ShareOption
		wantErr bool
		err     error
		mock    func()
	}{
		{
			name:    "success",
			wantErr: false,
			share:   testShare,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
			},
		},
		{
			name:    "encryption success",
			share:   testEncryptionShare,
			wantErr: false,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(nil)
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
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
			},
			err: domainErrors.ErrEncryptionPartRequired,
		},
		{
			name:    "encryption error",
			wantErr: true,
			share:   testEncryptionShare,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
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
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(&share.Share{}, nil)
			},
		},
		{
			name:    "repository error on get",
			wantErr: true,
			share:   testShare,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, errors.New("repository error"))
			},
		},
		{
			name:    "repository error on create",
			wantErr: true,
			share:   testShare,
			mock: func() {
				mockRepo.ExpectedCalls = nil
				mockRepo.On("GetByUserID", mock.Anything, testUserID).Return(nil, domainErrors.ErrShareNotFound)
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*share.Share")).Return(errors.New("repository error"))
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := svc.Create(ctx, tt.share, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("Create() error = %v, expected error %v", err, tt.err)
			}
		})
	}
}
