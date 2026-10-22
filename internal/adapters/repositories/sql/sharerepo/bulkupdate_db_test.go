package sharerepo

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

func TestBulkUpdate_RollsBackAllOnError(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	p := dbtest.SeedProject(t, client, "", false)

	first := dbtest.InsertShare(t, client, p, "project", "old-1")
	second := dbtest.InsertShare(t, client, p, "project", "old-2")

	err := repo.BulkUpdate(context.Background(), []*share.Share{
		{ID: first, Secret: "new-1", UserID: p.UserID, Entropy: share.EntropyProject},
		// No such user: violates fk_share_user.
		{ID: second, Secret: "new-2", UserID: uuid.NewString(), Entropy: share.EntropyProject},
	})
	if err == nil {
		t.Fatal("BulkUpdate succeeded although the second share violates a foreign key")
	}

	if got := dbtest.ShareData(t, client, first); got != "old-1" {
		t.Errorf("first share was written despite the failed batch: data = %q, want %q", got, "old-1")
	}
	if got := dbtest.ShareData(t, client, second); got != "old-2" {
		t.Errorf("second share changed: data = %q, want %q", got, "old-2")
	}
}
