package apptest

import (
	"context"
	"errors"
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/random"
)

func runShamirJob(t *testing.T, apps Apps, p dbtest.Project, projectPart, key string) error {
	t.Helper()
	return apps.Job.Execute(context.Background(), p.ID, storedPart(t, apps, p), project.ProjectPart(projectPart), key)
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
