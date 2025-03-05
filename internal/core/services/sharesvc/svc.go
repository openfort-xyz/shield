package sharesvc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/internal/core/domain/keychain"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/factories"

	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/logger"
)

type service struct {
	repo              repositories.ShareRepository
	keychainRepo      repositories.KeychainRepository
	logger            *slog.Logger
	encryptionFactory factories.EncryptionFactory
}

var _ services.ShareService = (*service)(nil)

func New(repo repositories.ShareRepository, keychainRepo repositories.KeychainRepository, encryptionFactory factories.EncryptionFactory) services.ShareService {
	return &service{
		repo:              repo,
		keychainRepo:      keychainRepo,
		logger:            logger.New("share_service"),
		encryptionFactory: encryptionFactory,
	}
}

func (s *service) Create(ctx context.Context, shr *share.Share, opts ...services.ShareOption) error {
	s.logger.InfoContext(ctx, "creating share", slog.String("user_id", shr.UserID))

	shrRepo, err := s.Find(ctx, shr.UserID, shr.KeychainID, shr.Reference)
	if err != nil && !errors.Is(err, domainErrors.ErrShareNotFound) {
		s.logger.ErrorContext(ctx, "failed to get share", logger.Error(err))
		return err
	}

	if shrRepo != nil && shrRepo.Reference == shr.Reference {
		s.logger.ErrorContext(ctx, "share already exists", slog.String("user_id", shr.UserID))
		return domainErrors.ErrShareAlreadyExists
	}

	var o services.ShareOptions
	for _, opt := range opts {
		opt(&o)
	}

	if shr.RequiresEncryption() {
		if o.EncryptionKey == nil {
			return domainErrors.ErrEncryptionPartRequired
		}

		cypher := s.encryptionFactory.CreateEncryptionStrategy(*o.EncryptionKey)
		shr.Secret, err = cypher.Encrypt(shr.Secret)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to encrypt secret", logger.Error(err))
			return err
		}
	}

	err = s.validateKeychain(ctx, shr)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to validate keychain", logger.Error(err))
		return err
	}

	if shr.Reference == nil {
		ref := share.DefaultReference
		shr.Reference = &ref
	}

	err = s.repo.Create(ctx, shr)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create share", logger.Error(err))
		return err
	}

	return nil
}

func (s *service) Find(ctx context.Context, userID string, keychainID, reference *string) (*share.Share, error) {
	s.logger.InfoContext(ctx, "finding share", slog.String("user_id", userID))

	if keychainID == nil {
		shr, err := s.repo.GetByUserID(ctx, userID)
		if err != nil && !errors.Is(err, domainErrors.ErrShareNotFound) {
			s.logger.ErrorContext(ctx, "failed to get share by user ID", logger.Error(err))
			return nil, err
		}

		if err == nil {
			return shr, nil
		}

		userKeychain, err := s.getOrCreateKeychain(ctx, userID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get or create keychain", logger.Error(err))
			return nil, err
		}

		keychainID = &userKeychain.ID
	}

	if reference == nil {
		return s.repo.GetByReference(ctx, share.DefaultReference, *keychainID)
	}

	return s.repo.GetByReference(ctx, *reference, *keychainID)
}

func (s *service) validateKeychain(ctx context.Context, shr *share.Share) error {
	if shr.KeychainID == nil {
		userKeychain, err := s.getOrCreateKeychain(ctx, shr.UserID)
		if err != nil {
			return err
		}

		shr.KeychainID = &userKeychain.ID
		return nil
	}

	usrKeychain, err := s.keychainRepo.Get(ctx, *shr.KeychainID)
	if err != nil {
		return err
	}

	if usrKeychain.UserID != shr.UserID {
		return domainErrors.ErrKeychainNotFound
	}

	return nil
}

func (s *service) getOrCreateKeychain(ctx context.Context, userID string) (*keychain.Keychain, error) {
	userKeychain, err := s.keychainRepo.GetByUserID(ctx, userID)
	if err == nil {
		return userKeychain, nil
	}

	if !errors.Is(err, domainErrors.ErrKeychainNotFound) {
		return nil, err
	}

	userKeychain = &keychain.Keychain{
		ID:     uuid.NewString(),
		UserID: userID,
	}

	err = s.keychainRepo.Create(ctx, userKeychain)
	if err != nil {
		return nil, err
	}

	return userKeychain, nil
}
