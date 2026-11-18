package apptest

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// retireLegacyKey runs the README's "Retiring pre-migration keys" statements,
// each limited to the project so that tests sharing the database stay apart,
// and returns how many migration rows the insert added.
func retireLegacyKey(t *testing.T, apps Apps, p dbtest.Project) int64 {
	t.Helper()
	var inserted int64
	err := apps.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Exec(`SELECT ep.id FROM shld_encryption_parts ep
			WHERE NOT EXISTS (SELECT 1 FROM shld_shamir_migrations m WHERE m.project_id = ep.project_id AND m.success)
			AND ep.project_id = ?
			FOR UPDATE OF ep`, p.ID).Error
		if err != nil {
			return err
		}
		res := tx.Exec(`INSERT INTO shld_shamir_migrations (id, project_id, success)
			SELECT gen_random_uuid()::text, ep.project_id, true
			FROM (SELECT DISTINCT project_id FROM shld_encryption_parts) ep
			WHERE NOT EXISTS (SELECT 1 FROM shld_shamir_migrations m WHERE m.project_id = ep.project_id AND m.success)
			AND NOT EXISTS (SELECT 1 FROM shld_shares s JOIN shld_users u ON s.user_id = u.id
				WHERE u.project_id = ep.project_id AND s.entropy = 'project')
			AND ep.project_id = ?`, p.ID)
		inserted = res.RowsAffected
		return res.Error
	})
	if err != nil {
		t.Fatalf("retire pre-migration key: %v", err)
	}
	return inserted
}

func readShare(apps Apps, p dbtest.Project, reference string, opts ...shareapp.Option) (string, error) {
	shrs, err := apps.Share.GetKeychainShares(UserContext(p), &reference, opts...)
	if err != nil {
		return "", err
	}
	if len(shrs) != 1 {
		return "", errors.New("not exactly one share")
	}
	return shrs[0].Secret, nil
}

// An unmigrated project with no project share has nothing under its
// pre-migration key, so recording it as migrated moves it onto the current
// reconstruction of its own two parts, the same as a migrated project whose
// key predates key checks.
func TestRetireLegacyKey_ProjectWithoutProjectShares(t *testing.T) {
	for _, withOtherShares := range []bool{false, true} {
		name := "no shares"
		if withOtherShares {
			name = "plain and user-entropy shares"
		}
		t.Run(name, func(t *testing.T) {
			apps := New(t)
			p, part, _ := seedLegacyProject(t, apps, false)
			ctx := ProjectContext(p)
			others := map[string]string{}
			if withOtherShares {
				for entropy, data := range map[string]string{"none": "plain-share", "user": "user-share"} {
					others[shareReference(t, apps, dbtest.InsertShare(t, apps.DB, p, entropy, data))] = data
				}
			}

			if n := retireLegacyKey(t, apps, p); n != 1 {
				t.Fatalf("migration rows inserted = %d, want 1", n)
			}
			for reference, want := range others {
				if got, err := readShare(apps, p, reference); err != nil || got != want {
					t.Errorf("non-project share %s = %q, %v; want %q", reference, got, err, want)
				}
			}
			if _, err := apps.Project.PrepareEncryptionKeyRotation(ctx, part); !errors.Is(err, projectapp.ErrUnverifiableEncryptionKey) {
				t.Errorf("prepare before any project share: err = %v, want %v", err, projectapp.ErrUnverifiableEncryptionKey)
			}

			wallet := "wallet-" + p.ID[:8]
			err := apps.Share.RegisterShare(UserContext(p), &share.Share{Secret: "wallet", Entropy: share.EntropyProject, Reference: &wallet},
				shareapp.WithEncryptionPart(part))
			if err != nil {
				t.Fatalf("RegisterShare with the project's part: %v", err)
			}
			if got, err := readShare(apps, p, wallet, shareapp.WithEncryptionPart(part)); err != nil || got != "wallet" {
				t.Errorf("read with the project's part = %q, %v; want %q", got, err, "wallet")
			}
			assertSharesDecrypt(t, apps, p, part, 1)

			otherPart := randomPart(t)
			if _, err := readShare(apps, p, wallet, shareapp.WithEncryptionPart(otherPart)); !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
				t.Errorf("read with another part: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
			}
			err = apps.Share.RegisterShare(UserContext(p), &share.Share{Secret: "x", Entropy: share.EntropyProject, Reference: ptr("x-" + p.ID[:8])},
				shareapp.WithEncryptionPart(otherPart))
			if !errors.Is(err, shareapp.ErrEncryptionPartUnproven) {
				t.Errorf("write with another part: err = %v, want %v", err, shareapp.ErrEncryptionPartUnproven)
			}

			if _, err := apps.Project.PrepareEncryptionKeyRotation(ctx, otherPart); !errors.Is(err, projectapp.ErrInvalidEncryptionPart) {
				t.Errorf("prepare with another part: err = %v, want %v", err, projectapp.ErrInvalidEncryptionPart)
			}
			newPart, err := apps.Project.PrepareEncryptionKeyRotation(ctx, part)
			if err != nil {
				t.Fatalf("prepare with the project's part: %v", err)
			}
			if committed, err := apps.Project.CommitEncryptionKeyRotation(ctx, part, newPart); err != nil || !committed {
				t.Fatalf("commit: committed = %v, err = %v", committed, err)
			}
			if got, err := readShare(apps, p, wallet, shareapp.WithEncryptionPart(newPart)); err != nil || got != "wallet" {
				t.Errorf("read with the rotated part = %q, %v; want %q", got, err, "wallet")
			}
			for reference, want := range others {
				if got, err := readShare(apps, p, reference); err != nil || got != want {
					t.Errorf("non-project share %s after rotation = %q, %v; want %q", reference, got, err, want)
				}
			}
		})
	}
}

// A project share of an unmigrated project is under the pre-migration key,
// which the current reconstruction does not rebuild: the statement skips such
// a project, soft-deleted shares included, and a success row inserted for it
// anyway leaves the share unreadable with the project's part. Deleting the
// row restores reads only on a process that has not yet seen it, since each
// process caches a project's migrated status once it observes it.
func TestRetireLegacyKey_ProjectWithProjectShare(t *testing.T) {
	apps := New(t)

	t.Run("soft-deleted share", func(t *testing.T) {
		p, _, key := seedLegacyProject(t, apps, false)
		id := insertProjectShare(t, apps, p, key, "wallet")
		dbtest.Exec(t, apps.DB, `UPDATE shld_shares SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`, id)

		if n := retireLegacyKey(t, apps, p); n != 0 {
			t.Errorf("migration rows inserted = %d, want 0", n)
		}
	})

	t.Run("live share", func(t *testing.T) {
		p, part, key := seedLegacyProject(t, apps, false)
		reference := shareReference(t, apps, insertProjectShare(t, apps, p, key, "wallet"))

		if n := retireLegacyKey(t, apps, p); n != 0 {
			t.Errorf("migration rows inserted = %d, want 0", n)
		}
		if got, err := readShare(apps, p, reference, shareapp.WithEncryptionPart(part)); err != nil || got != "wallet" {
			t.Fatalf("read before a direct insert = %q, %v; want %q", got, err, "wallet")
		}

		row := uuid.NewString()
		dbtest.Exec(t, apps.DB, `INSERT INTO shld_shamir_migrations (id, project_id, success) VALUES (?, ?, true)`, row, p.ID)
		if _, err := readShare(apps, p, reference, shareapp.WithEncryptionPart(part)); !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
			t.Errorf("read with the project's part after a direct insert: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
		}

		dbtest.Exec(t, apps.DB, `DELETE FROM shld_shamir_migrations WHERE id = ?`, row)
		if _, err := readShare(apps, p, reference, shareapp.WithEncryptionPart(part)); !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
			t.Errorf("read on the same process after deleting the row: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
		}
		restarted := New(t)
		if got, err := readShare(restarted, p, reference, shareapp.WithEncryptionPart(part)); err != nil || got != "wallet" {
			t.Errorf("read on a restarted process after deleting the row = %q, %v; want %q", got, err, "wallet")
		}
	})
}
