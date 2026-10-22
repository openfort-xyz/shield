package projectsvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type cacheEntry struct {
	project   *project.Project
	expiresAt time.Time
}

type service struct {
	repo     repositories.ProjectRepository
	logger   *slog.Logger
	cost     int
	cache    map[string]*cacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
	// evictions counts API-key evictions; a lookup that started before one
	// must not cache what it read, or it could re-cache a revoked key.
	evictions uint64
}

var _ services.ProjectService = (*service)(nil)

func New(repo repositories.ProjectRepository, cacheTTL time.Duration) services.ProjectService {
	return &service{
		repo:     repo,
		logger:   logger.New("project_service"),
		cost:     bcrypt.DefaultCost,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: cacheTTL,
	}
}

func (s *service) Create(ctx context.Context, name string, enable2fa bool) (*project.Project, error) {
	s.logger.InfoContext(ctx, "creating project", slog.String("name", name))
	apiSecretBytes := make([]byte, 32)
	_, err := rand.Read(apiSecretBytes)
	if err != nil {
		// Shouldn't happen (error is returned for legacy reasons and it's unlikely we'll ever see it)
		return nil, err
	}
	encryptedSecret, err := bcrypt.GenerateFromPassword(apiSecretBytes, s.cost)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to encrypt secret", logger.Error(err))
		return nil, err
	}

	proj := &project.Project{
		Name:      name,
		APIKey:    uuid.NewString(),
		APISecret: string(encryptedSecret),
		Enable2FA: enable2fa,
	}

	err = s.repo.Create(ctx, proj)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create project", logger.Error(err))
		return nil, err
	}

	proj.APISecret = hex.EncodeToString(apiSecretBytes)
	return proj, nil
}

func (s *service) GetByAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	s.logger.InfoContext(ctx, "getting project by api key")

	s.cacheMu.RLock()
	if entry, ok := s.cache[apiKey]; ok && time.Now().Before(entry.expiresAt) {
		s.cacheMu.RUnlock()
		return entry.project, nil
	}
	evictions := s.evictions
	s.cacheMu.RUnlock()

	proj, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get project by api key", logger.Error(err))
		return nil, err
	}

	s.cacheMu.Lock()
	if s.evictions == evictions {
		s.cache[apiKey] = &cacheEntry{
			project:   proj,
			expiresAt: time.Now().Add(s.cacheTTL),
		}
	}
	s.cacheMu.Unlock()

	return proj, nil
}

func (s *service) SaveProjectRateLimits(ctx context.Context, projectID string, smsRateLimit int64, emailRateLimit int64) error {
	s.logger.InfoContext(ctx, "save project rate limits", slog.String("project", projectID))

	projectRateLimit := &project.RateLimit{
		ProjectID:            projectID,
		SMSRequestsPerHour:   smsRateLimit,
		EmailRequestsPerHour: emailRateLimit,
	}

	err := s.repo.SaveProjectRateLimits(ctx, projectRateLimit)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to save project rate limits", logger.Error(err))
	}

	return err
}

func (s *service) SetEncryptionKey(ctx context.Context, projectID string, key *project.EncryptionKey) error {
	s.logger.InfoContext(ctx, "setting encryption key", slog.String("project_id", projectID))
	ep, err := s.repo.GetEncryptionKey(ctx, projectID)
	if err != nil && !errors.Is(err, domainErrors.ErrEncryptionPartNotFound) {
		s.logger.ErrorContext(ctx, "failed to get encryption part", logger.Error(err))
		return err
	}

	if ep != nil {
		s.logger.Warn("encryption part already exists", slog.String("project_id", projectID))
		return domainErrors.ErrEncryptionPartAlreadyExists
	}

	err = s.repo.SetEncryptionKey(ctx, projectID, key)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to set encryption part", logger.Error(err))
		return err
	}

	return nil
}

func (s *service) PrepareAPIKeyRotation(ctx context.Context, projectID, presentedKey string, expiresAt time.Time) (string, error) {
	s.logger.InfoContext(ctx, "preparing API key rotation", slog.String("project_id", projectID))
	apiKey := uuid.NewString()
	if err := s.repo.SetPendingAPIKey(ctx, projectID, presentedKey, apiKey, expiresAt); err != nil {
		s.logger.ErrorContext(ctx, "failed to set pending API key", logger.Error(err))
		return "", err
	}
	return apiKey, nil
}

func (s *service) CommitAPIKeyRotation(ctx context.Context, projectID, presentedKey string) (string, bool, error) {
	s.logger.InfoContext(ctx, "committing API key rotation", slog.String("project_id", projectID))

	var oldKey string
	committed := false
	err := s.repo.CommitAPIKeyRotation(ctx, projectID, func(state project.APIKeyRotation) (bool, error) {
		pending := state.PendingAPIKey != nil && state.PendingExpiresAt != nil
		expired := pending && time.Now().After(*state.PendingExpiresAt)
		switch {
		case pending && !expired && *state.PendingAPIKey == presentedKey:
			oldKey, committed = state.APIKey, true
			return true, nil
		case expired && *state.PendingAPIKey == presentedKey:
			return false, domainErrors.ErrAPIKeyRotationExpired
		case presentedKey != state.APIKey:
			return false, domainErrors.ErrNoPendingAPIKeyRotation
		case pending && !expired:
			return false, domainErrors.ErrPendingAPIKeyNotProven
		case expired:
			return false, domainErrors.ErrAPIKeyRotationExpired
		default:
			return false, nil
		}
	})
	if err != nil {
		return "", false, err
	}

	if committed {
		// The cache is per process: other replicas keep accepting the old
		// key on cached routes until their entry expires (cacheTTL).
		s.cacheMu.Lock()
		delete(s.cache, oldKey)
		s.evictions++
		s.cacheMu.Unlock()
	}
	return presentedKey, committed, nil
}
