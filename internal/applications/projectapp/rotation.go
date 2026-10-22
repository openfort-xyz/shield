package projectapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/pkg/contexter"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/logger"
	"github.com/openfort-xyz/shield/pkg/random"
)

// rotationPendingTTL bounds how long a prepared rotation can be committed.
const rotationPendingTTL = 15 * time.Minute

// PrepareEncryptionKeyRotation generates a replacement encryption key and
// returns the project's half of it. Nothing is replaced until
// CommitEncryptionKeyRotation is called with that half, so a lost response
// only costs a repeated prepare. Preparing again replaces the pending key.
func (a *ProjectApplication) PrepareEncryptionKeyRotation(ctx context.Context, currentPart string) (string, error) {
	projectID := contexter.GetProjectID(ctx)
	a.logger.InfoContext(ctx, "preparing encryption key rotation", slog.String("project_id", projectID))

	var newPart project.ProjectPart
	err := a.projectRepo.PrepareEncryptionKeyRotation(ctx, projectID,
		func(state project.EncryptionKeyRotation, loadShares func() ([]project.EncryptedShare, error)) (*project.PreparedRotation, error) {
			if !state.Migrated {
				return nil, ErrProjectNotMigrated
			}
			currentKey, err := a.verifiedCurrentKey(ctx, state, project.ProjectPart(currentPart), loadShares)
			if err != nil {
				return nil, err
			}
			currentCheck, err := cypher.KeyCheck(currentKey)
			if err != nil {
				return nil, err
			}

			newKey, err := random.GenerateRandomString(32)
			if err != nil {
				return nil, err
			}
			storedPart, projectPart, err := a.encryptionFactory.CreateReconstructionStrategy(true).Split(newKey)
			if err != nil {
				return nil, err
			}
			newCheck, err := cypher.KeyCheck(newKey)
			if err != nil {
				return nil, err
			}

			newPart = projectPart
			return &project.PreparedRotation{
				Pending: project.PendingEncryptionKey{
					Part:      storedPart,
					KeyCheck:  project.KeyCheck(newCheck),
					ExpiresAt: time.Now().Add(rotationPendingTTL),
				},
				CurrentKeyCheck: project.KeyCheck(currentCheck),
			}, nil
		})
	if err != nil {
		return "", a.rotationError(ctx, "prepare", err)
	}

	return string(newPart), nil
}

// CommitEncryptionKeyRotation re-encrypts every project share under the
// prepared key and makes it current. It reports false when newPart is
// already the current key: a retry after a lost commit response.
func (a *ProjectApplication) CommitEncryptionKeyRotation(ctx context.Context, currentPart, newPart string) (committed bool, err error) {
	projectID := contexter.GetProjectID(ctx)
	a.logger.InfoContext(ctx, "committing encryption key rotation", slog.String("project_id", projectID))
	started := time.Now()

	var count int
	committed = false
	err = a.projectRepo.CommitEncryptionKeyRotation(ctx, projectID,
		func(state project.EncryptionKeyRotation, shares []project.EncryptedShare) (*project.CommittedRotation, error) {
			if !state.Migrated {
				return nil, ErrProjectNotMigrated
			}
			// A retry after a lost commit response: the new part is already current.
			if currentPart != newPart && a.matchesKey(state.Current.Part, project.ProjectPart(newPart), state.Current.KeyCheck) {
				return nil, nil
			}
			if state.Pending == nil {
				return nil, ErrNoPendingRotation
			}
			if time.Now().After(state.Pending.ExpiresAt) {
				return nil, ErrRotationExpired
			}

			strategy := a.encryptionFactory.CreateReconstructionStrategy(true)
			newKey, err := strategy.Reconstruct(state.Pending.Part, project.ProjectPart(newPart))
			if err != nil || !cypher.VerifyKeyCheck(newKey, string(state.Pending.KeyCheck)) {
				return nil, ErrInvalidEncryptionPart
			}
			currentKey, err := a.verifiedCurrentKey(ctx, state, project.ProjectPart(currentPart),
				func() ([]project.EncryptedShare, error) { return shares, nil })
			if err != nil {
				return nil, err
			}

			reencrypted, err := reencrypt(shares, currentKey, newKey)
			if err != nil {
				return nil, err
			}

			count = len(reencrypted)
			committed = true
			check := state.Pending.KeyCheck
			return &project.CommittedRotation{
				Key:    project.EncryptionKey{Part: state.Pending.Part, KeyCheck: &check},
				Shares: reencrypted,
			}, nil
		})
	if err != nil {
		return false, a.rotationError(ctx, "commit", err)
	}

	a.logger.InfoContext(ctx, "encryption key rotation finished",
		slog.Bool("committed", committed), slog.Int("shares", count), slog.Duration("duration", time.Since(started)))
	return committed, nil
}

// verifiedCurrentKey reconstructs the current key from the caller's part and
// proves it: against the key check, or for keys without one by decrypting
// every project share (a project with no shares has nothing to protect).
func (a *ProjectApplication) verifiedCurrentKey(ctx context.Context, state project.EncryptionKeyRotation, part project.ProjectPart, loadShares func() ([]project.EncryptedShare, error)) (string, error) {
	key, err := a.encryptionFactory.CreateReconstructionStrategy(true).Reconstruct(state.Current.Part, part)
	if err != nil {
		return "", ErrInvalidEncryptionPart
	}
	if state.Current.KeyCheck != nil {
		if !cypher.VerifyKeyCheck(key, string(*state.Current.KeyCheck)) {
			return "", ErrInvalidEncryptionPart
		}
		return key, nil
	}

	shares, err := loadShares()
	if err != nil {
		return "", err
	}
	if len(shares) == 0 {
		return "", ErrUnverifiableEncryptionKey
	}
	for _, shr := range shares {
		if _, err := cypher.Decrypt(shr.Secret, key); err != nil {
			if errors.Is(err, cypher.ErrAuthenticationFailed) {
				a.logger.WarnContext(ctx, "encryption part does not decrypt the project's shares", slog.String("share_id", shr.ID))
				return "", ErrInvalidEncryptionPart
			}
			return "", err
		}
	}
	return key, nil
}

func (a *ProjectApplication) matchesKey(stored project.StoredPart, part project.ProjectPart, check *project.KeyCheck) bool {
	if check == nil {
		return false
	}
	key, err := a.encryptionFactory.CreateReconstructionStrategy(true).Reconstruct(stored, part)
	return err == nil && cypher.VerifyKeyCheck(key, string(*check))
}

// reencrypt moves shares from a verified current key to the new key. A share
// that does not decrypt under the verified key is corrupt data, not a wrong
// part, so it fails the rotation as an internal error naming the share.
func reencrypt(shares []project.EncryptedShare, fromKey, toKey string) ([]project.EncryptedShare, error) {
	out := make([]project.EncryptedShare, 0, len(shares))
	for _, shr := range shares {
		plain, err := cypher.Decrypt(shr.Secret, fromKey)
		if err != nil {
			return nil, fmt.Errorf("share %s does not decrypt under the verified current key: %w", shr.ID, err)
		}
		secret, err := cypher.Encrypt(plain, toKey)
		if err != nil {
			return nil, err
		}
		out = append(out, project.EncryptedShare{ID: shr.ID, Secret: secret})
	}
	return out, nil
}

func (a *ProjectApplication) rotationError(ctx context.Context, phase string, err error) error {
	switch {
	case errors.Is(err, ErrInvalidEncryptionPart), errors.Is(err, ErrNoPendingRotation),
		errors.Is(err, ErrRotationExpired), errors.Is(err, ErrProjectNotMigrated),
		errors.Is(err, ErrUnverifiableEncryptionKey):
		a.logger.WarnContext(ctx, "encryption key rotation rejected", slog.String("phase", phase), logger.Error(err))
		return err
	case errors.Is(err, domainErrors.ErrEncryptionPartNotFound):
		return ErrEncryptionNotConfigured
	default:
		a.logger.ErrorContext(ctx, "encryption key rotation failed", slog.String("phase", phase), logger.Error(err))
		return ErrInternal
	}
}

// PrepareAPIKeyRotation returns a new pending API key. The current key keeps
// working until CommitAPIKeyRotation is called with the new one.
func (a *ProjectApplication) PrepareAPIKeyRotation(ctx context.Context) (string, error) {
	apiKey, err := a.projectSvc.PrepareAPIKeyRotation(ctx, contexter.GetProjectID(ctx), contexter.GetAPIKey(ctx), time.Now().Add(rotationPendingTTL))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to prepare API key rotation", logger.Error(err))
		return "", fromDomainError(err)
	}
	return apiKey, nil
}

// CommitAPIKeyRotation makes the pending API key current. It must be
// authenticated with that pending key; it reports false when the presented
// key is already current with nothing pending (a retry after a lost response).
func (a *ProjectApplication) CommitAPIKeyRotation(ctx context.Context) (string, bool, error) {
	apiKey, committed, err := a.projectSvc.CommitAPIKeyRotation(ctx, contexter.GetProjectID(ctx), contexter.GetAPIKey(ctx))
	if err != nil {
		a.logger.WarnContext(ctx, "API key rotation commit rejected", logger.Error(err))
		return "", false, fromDomainError(err)
	}
	a.logger.InfoContext(ctx, "API key rotation finished", slog.Bool("committed", committed))
	return apiKey, committed, nil
}

// PrepareAPISecretRotation returns a new pending API secret. The current
// secret keeps working until CommitAPISecretRotation is called with the new one.
func (a *ProjectApplication) PrepareAPISecretRotation(ctx context.Context) (string, error) {
	secret, err := a.projectSvc.PrepareAPISecretRotation(ctx, contexter.GetProjectID(ctx), contexter.GetAPIKey(ctx), contexter.GetAPISecret(ctx), time.Now().Add(rotationPendingTTL))
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to prepare API secret rotation", logger.Error(err))
		return "", fromDomainError(err)
	}
	return secret, nil
}

// CommitAPISecretRotation makes the pending API secret current. It must be
// authenticated with that pending secret; it reports false when the presented
// secret is already current with nothing pending (a retry after a lost response).
func (a *ProjectApplication) CommitAPISecretRotation(ctx context.Context) (bool, error) {
	committed, err := a.projectSvc.CommitAPISecretRotation(ctx, contexter.GetProjectID(ctx), contexter.GetAPIKey(ctx), contexter.GetAPISecret(ctx))
	if err != nil {
		a.logger.WarnContext(ctx, "API secret rotation commit rejected", logger.Error(err))
		return false, fromDomainError(err)
	}
	a.logger.InfoContext(ctx, "API secret rotation finished", slog.Bool("committed", committed))
	return committed, nil
}
