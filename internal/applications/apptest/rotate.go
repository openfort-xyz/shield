package apptest

import (
	"context"
	"sync/atomic"
	"testing"

	sssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/random"
	"gorm.io/gorm"
)

// RotateOutOfBand replaces the project's encryption key the way a rotation
// commit does, in one transaction holding FOR UPDATE on the part row:
// re-encrypt every project share, then update the part and key check in
// place. It returns the new project part.
func (a Apps) RotateOutOfBand(t *testing.T, p dbtest.Project, oldProjectPart string) string {
	t.Helper()
	strategy := sssrec.NewSSSReconstructionStrategy()

	newKey, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	newStored, newProjectPart, err := strategy.Split(newKey)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	newCheck, err := cypher.KeyCheck(newKey)
	if err != nil {
		t.Fatalf("key check: %v", err)
	}

	err = a.DB.Transaction(func(tx *gorm.DB) error {
		var oldStored string
		if err := tx.Raw(`SELECT part FROM shld_encryption_parts WHERE project_id = ? FOR UPDATE`, p.ID).Scan(&oldStored).Error; err != nil {
			return err
		}
		oldKey, err := strategy.Reconstruct(project.StoredPart(oldStored), project.ProjectPart(oldProjectPart))
		if err != nil {
			return err
		}
		type row struct{ ID, Data string }
		var rows []row
		if err := tx.Raw(`SELECT s.id, s.data FROM shld_shares s JOIN shld_users u ON s.user_id = u.id WHERE u.project_id = ? AND s.entropy = 'project' AND s.deleted_at IS NULL`, p.ID).Scan(&rows).Error; err != nil {
			return err
		}
		for _, r := range rows {
			plain, err := cypher.Decrypt(r.Data, oldKey)
			if err != nil {
				return err
			}
			enc, err := cypher.Encrypt(plain, newKey)
			if err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE shld_shares SET data = ? WHERE id = ?`, enc, r.ID).Error; err != nil {
				return err
			}
		}
		return tx.Exec(`UPDATE shld_encryption_parts SET part = ?, key_check = ? WHERE project_id = ?`, string(newStored), newCheck, p.ID).Error
	})
	if err != nil {
		t.Fatalf("out-of-band rotation: %v", err)
	}
	return string(newProjectPart)
}

// ParkingShareRepo pauses Create, Update and BulkUpdate once armed: each
// signals Parked and waits for Release before reaching the database.
type ParkingShareRepo struct {
	repositories.ShareRepository
	Armed   atomic.Bool
	Parked  chan struct{}
	Release chan struct{}
}

// NewParkingShareRepo returns an unarmed parking decorator.
func NewParkingShareRepo() *ParkingShareRepo {
	return &ParkingShareRepo{Parked: make(chan struct{}, 1), Release: make(chan struct{})}
}

// Wrap is the function to pass to NewWithShareRepo.
func (r *ParkingShareRepo) Wrap(inner repositories.ShareRepository) repositories.ShareRepository {
	r.ShareRepository = inner
	return r
}

func (r *ParkingShareRepo) park() {
	if r.Armed.Load() {
		r.Parked <- struct{}{}
		<-r.Release
	}
}

func (r *ParkingShareRepo) Create(ctx context.Context, shr *share.Share) error {
	r.park()
	return r.ShareRepository.Create(ctx, shr)
}

func (r *ParkingShareRepo) Update(ctx context.Context, shr *share.Share) error {
	r.park()
	return r.ShareRepository.Update(ctx, shr)
}

func (r *ParkingShareRepo) BulkUpdate(ctx context.Context, shrs []*share.Share) error {
	r.park()
	return r.ShareRepository.BulkUpdate(ctx, shrs)
}
