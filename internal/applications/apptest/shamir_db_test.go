package apptest

import (
	"context"
	"errors"
	"testing"

	depsssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/deprecated_sss_reconstruction_strategy"
	sssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/random"
)

func runShamirJob(t *testing.T, apps Apps, p dbtest.Project, projectPart, key string) error {
	t.Helper()
	return apps.Job.Execute(context.Background(), p.ID, storedPart(t, apps, p), project.ProjectPart(projectPart), key)
}

func storedPart(t *testing.T, apps Apps, p dbtest.Project) project.StoredPart {
	t.Helper()
	var part string
	if err := apps.DB.Raw(`SELECT part FROM shld_encryption_parts WHERE project_id = ? ORDER BY id LIMIT 1`, p.ID).Scan(&part).Error; err != nil {
		t.Fatalf("read stored part: %v", err)
	}
	return project.StoredPart(part)
}

func isMigrated(t *testing.T, apps Apps, p dbtest.Project) bool {
	t.Helper()
	var n int64
	if err := apps.DB.Raw(`SELECT count(*) FROM shld_shamir_migrations WHERE project_id = ? AND success`, p.ID).Scan(&n).Error; err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	return n > 0
}

// assertSharesDecrypt fails unless the project has want project-entropy
// shares, soft-deleted ones included, and each decrypts under the key that
// the project's recorded migration status makes of its parts: the key every
// later request reconstructs.
func assertSharesDecrypt(t *testing.T, apps Apps, p dbtest.Project, projectPart string, want int) {
	t.Helper()
	migrated := isMigrated(t, apps, p)
	reconstruct := depsssrec.NewSSSReconstructionStrategy().Reconstruct
	if migrated {
		reconstruct = sssrec.NewSSSReconstructionStrategy().Reconstruct
	}
	key, err := reconstruct(storedPart(t, apps, p), project.ProjectPart(projectPart))
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}

	var rows []struct{ ID, Data string }
	err = apps.DB.Raw(`SELECT s.id, s.data FROM shld_shares s JOIN shld_users u ON s.user_id = u.id
		WHERE u.project_id = ? AND s.entropy = 'project'`, p.ID).Scan(&rows).Error
	if err != nil {
		t.Fatalf("read shares: %v", err)
	}
	if len(rows) != want {
		t.Errorf("project shares = %d, want %d", len(rows), want)
	}
	for _, row := range rows {
		if _, err := cypher.Decrypt(row.Data, key); err != nil {
			t.Errorf("share %s does not decrypt under the project's key (migrated = %v): %v", row.ID, migrated, err)
		}
	}
}

// A migration that cannot decrypt the project's shares fails; the caller
// must see that failure, not a success.
func TestShamirJob_FailedMigrationReturnsItsError(t *testing.T) {
	apps := New(t)
	p, part, key := seedLegacyProject(t, apps, false)
	insertProjectShare(t, apps, p, key, "wallet")
	wrongKey, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	err = runShamirJob(t, apps, p, part, wrongKey)

	if !errors.Is(err, cypher.ErrAuthenticationFailed) {
		t.Errorf("migration with a key that does not decrypt the shares: err = %v, want %v", err, cypher.ErrAuthenticationFailed)
	}
	if isMigrated(t, apps, p) {
		t.Error("project recorded as migrated after a failed migration")
	}
	assertSharesDecrypt(t, apps, p, part, 1)
}
