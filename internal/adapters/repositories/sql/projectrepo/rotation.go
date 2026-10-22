package projectrepo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *repository) PrepareEncryptionKeyRotation(ctx context.Context, projectID string, prepare project.PrepareRotationFunc) error {
	r.logger.InfoContext(ctx, "preparing encryption key rotation", slog.String("project_id", projectID))

	return r.db.Transaction(func(tx *gorm.DB) error {
		state, err := lockRotationState(tx, projectID)
		if err != nil {
			return err
		}

		prepared, err := prepare(*state, func() ([]project.EncryptedShare, error) {
			return listProjectShares(tx, projectID, false)
		})
		if err != nil {
			return err
		}

		return tx.Model(&EncryptionPart{}).Where("project_id = ?", projectID).Updates(map[string]any{
			"key_check":          string(prepared.CurrentKeyCheck),
			"pending_part":       string(prepared.Pending.Part),
			"pending_key_check":  string(prepared.Pending.KeyCheck),
			"pending_expires_at": prepared.Pending.ExpiresAt,
		}).Error
	})
}

func (r *repository) CommitEncryptionKeyRotation(ctx context.Context, projectID string, commit project.CommitRotationFunc) error {
	r.logger.InfoContext(ctx, "committing encryption key rotation", slog.String("project_id", projectID))

	return r.db.Transaction(func(tx *gorm.DB) error {
		state, err := lockRotationState(tx, projectID)
		if err != nil {
			return err
		}
		shares, err := listProjectShares(tx, projectID, true)
		if err != nil {
			return err
		}

		committed, err := commit(*state, shares)
		if err != nil || committed == nil {
			return err
		}

		for _, shr := range committed.Shares {
			res := tx.Exec(`UPDATE shld_shares SET data = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, shr.Secret, shr.ID)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return fmt.Errorf("re-encrypting share %s: %d rows updated, want 1", shr.ID, res.RowsAffected)
			}
		}

		// In place, never DELETE+INSERT: a share write blocked on FOR SHARE
		// must see the new part, not zero rows (sharerepo TestPartRowLockSemantics).
		var keyCheck *string
		if committed.Key.KeyCheck != nil {
			check := string(*committed.Key.KeyCheck)
			keyCheck = &check
		}
		return tx.Model(&EncryptionPart{}).Where("project_id = ?", projectID).Updates(map[string]any{
			"part":               string(committed.Key.Part),
			"key_check":          keyCheck,
			"pending_part":       nil,
			"pending_key_check":  nil,
			"pending_expires_at": nil,
		}).Error
	})
}

// lockRotationState takes FOR UPDATE on every encryption part row of the
// project (a project may hold duplicates; GetEncryptionKey reads the first)
// and reads the rotation state under that lock.
func lockRotationState(tx *gorm.DB, projectID string) (*project.EncryptionKeyRotation, error) {
	var parts []EncryptionPart
	err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("project_id = ?", projectID).Order("id").Find(&parts).Error
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, domainErrors.ErrEncryptionPartNotFound
	}

	var migrations int64
	err = tx.Model(&Migration{}).Where("project_id = ? AND success = ?", projectID, true).Count(&migrations).Error
	if err != nil {
		return nil, err
	}

	row := parts[0]
	state := &project.EncryptionKeyRotation{
		Current:  *toEncryptionKey(&row),
		Migrated: migrations > 0,
	}
	if row.PendingPart != nil && row.PendingKeyCheck != nil && row.PendingExpiresAt != nil {
		state.Pending = &project.PendingEncryptionKey{
			Part:      project.StoredPart(*row.PendingPart),
			KeyCheck:  project.KeyCheck(*row.PendingKeyCheck),
			ExpiresAt: *row.PendingExpiresAt,
		}
	}
	return state, nil
}

// listProjectShares returns every project-entropy share of the project,
// soft-deleted ones included so they stay restorable after a rotation. With
// lock, the rows are held FOR UPDATE until the transaction ends: a write that
// moves a share out of project entropy (and so carries no guard) then waits
// for the commit and lands after it, instead of being overwritten by it.
func listProjectShares(tx *gorm.DB, projectID string, lock bool) ([]project.EncryptedShare, error) {
	query := `SELECT s.id, s.data AS secret FROM shld_shares s
		JOIN shld_users u ON s.user_id = u.id
		WHERE u.project_id = ? AND s.entropy = 'project'
		ORDER BY s.id`
	if lock {
		query += ` FOR UPDATE OF s`
	}
	var shares []project.EncryptedShare
	err := tx.Raw(query, projectID).Scan(&shares).Error
	return shares, err
}

func (r *repository) SetPendingAPIKey(ctx context.Context, projectID, currentKey, apiKey string, expiresAt time.Time) error {
	r.logger.InfoContext(ctx, "setting pending API key", slog.String("project_id", projectID))

	res := r.db.Model(&Project{}).Where("id = ? AND api_key = ?", projectID, currentKey).Updates(map[string]any{
		"pending_api_key":            apiKey,
		"pending_api_key_expires_at": expiresAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return domainErrors.ErrAPIKeyRevoked
	}
	return nil
}

func (r *repository) GetByPendingAPIKey(ctx context.Context, apiKey string) (*project.Project, error) {
	r.logger.InfoContext(ctx, "getting project by pending API key")

	dbProj := &Project{}
	err := r.db.Where("pending_api_key = ? AND pending_api_key_expires_at > ?", apiKey, time.Now()).First(dbProj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainErrors.ErrProjectNotFound
		}
		return nil, err
	}

	return r.parser.toDomain(dbProj), nil
}

func (r *repository) CommitAPIKeyRotation(ctx context.Context, projectID string, commit project.CommitAPIKeyRotationFunc) error {
	r.logger.InfoContext(ctx, "committing API key rotation", slog.String("project_id", projectID))

	return r.db.Transaction(func(tx *gorm.DB) error {
		dbProj, err := lockProject(tx, projectID)
		if err != nil {
			return err
		}

		swap, err := commit(project.APIKeyRotation{
			APIKey:           dbProj.APIKey,
			PendingAPIKey:    dbProj.PendingAPIKey,
			PendingExpiresAt: dbProj.PendingAPIKeyExpiresAt,
		})
		if err != nil || !swap {
			return err
		}

		return tx.Model(&Project{}).Where("id = ?", projectID).Updates(map[string]any{
			"api_key":                    gorm.Expr("pending_api_key"),
			"pending_api_key":            nil,
			"pending_api_key_expires_at": nil,
		}).Error
	})
}

func (r *repository) SetPendingAPISecret(ctx context.Context, projectID, secretHash string, expiresAt time.Time, check project.CheckAPISecretFunc) error {
	r.logger.InfoContext(ctx, "setting pending API secret", slog.String("project_id", projectID))

	return r.db.Transaction(func(tx *gorm.DB) error {
		dbProj, err := lockProject(tx, projectID)
		if err != nil {
			return err
		}

		if err := check(project.APISecretRotation{APIKey: dbProj.APIKey, APISecretHash: dbProj.APISecret}); err != nil {
			return err
		}

		return tx.Model(&Project{}).Where("id = ?", projectID).Updates(map[string]any{
			"pending_api_secret":            secretHash,
			"pending_api_secret_expires_at": expiresAt,
		}).Error
	})
}

func (r *repository) CommitAPISecretRotation(ctx context.Context, projectID string, commit project.CommitAPISecretRotationFunc) error {
	r.logger.InfoContext(ctx, "committing API secret rotation", slog.String("project_id", projectID))

	return r.db.Transaction(func(tx *gorm.DB) error {
		dbProj, err := lockProject(tx, projectID)
		if err != nil {
			return err
		}

		swap, err := commit(project.APISecretRotation{
			APIKey:           dbProj.APIKey,
			APISecretHash:    dbProj.APISecret,
			PendingHash:      dbProj.PendingAPISecret,
			PendingExpiresAt: dbProj.PendingAPISecretExpiresAt,
		})
		if err != nil || !swap {
			return err
		}

		return tx.Model(&Project{}).Where("id = ?", projectID).Updates(map[string]any{
			"api_secret":                    gorm.Expr("pending_api_secret"),
			"pending_api_secret":            nil,
			"pending_api_secret_expires_at": nil,
		}).Error
	})
}

// lockProject reads the project row under FOR UPDATE. API key and API secret
// commits both take it, so they serialize; each updates only its own columns.
func lockProject(tx *gorm.DB, projectID string) (*Project, error) {
	dbProj := &Project{}
	err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Where("id = ?", projectID).First(dbProj).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainErrors.ErrProjectNotFound
	}
	return dbProj, err
}
