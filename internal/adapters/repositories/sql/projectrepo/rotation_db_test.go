package projectrepo

import (
	"context"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

// A writer that takes a share out of project entropy (no guard: the new
// entropy needs no project key) must not be overwritten by a commit that
// listed the share before the write and re-encrypts it after.
func TestCommitEncryptionKeyRotation_UnguardedWriterNotOverwritten(t *testing.T) {
	client := dbtest.RequireDB(t)
	repo := New(client)
	p := dbtest.SeedProject(t, client, "P1", true)
	id := dbtest.InsertShare(t, client, p, "project", "C1")
	dbtest.Exec(t, client, `UPDATE shld_encryption_parts SET key_check = 'K1' WHERE project_id = ?`, p.ID)

	listed, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- repo.CommitEncryptionKeyRotation(context.Background(), p.ID,
			func(_ project.EncryptionKeyRotation, shares []project.EncryptedShare) (*project.CommittedRotation, error) {
				close(listed)
				<-release
				out := make([]project.EncryptedShare, 0, len(shares))
				for _, s := range shares {
					out = append(out, project.EncryptedShare{ID: s.ID, Secret: "C2"})
				}
				check := project.KeyCheck("K2")
				return &project.CommittedRotation{Key: project.EncryptionKey{Part: "P2", KeyCheck: &check}, Shares: out}, nil
			})
	}()
	<-listed

	written := make(chan error, 1)
	go func() {
		written <- client.Exec(`UPDATE shld_shares SET data = 'U', entropy = 'user' WHERE id = ?`, id).Error
	}()
	writerDone := false
	select {
	case err := <-written:
		writerDone = true
		t.Logf("unguarded write finished while the commit held its list (err %v)", err)
	case <-time.After(300 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("commit: %v", err)
	}
	if !writerDone {
		if err := <-written; err != nil {
			t.Fatalf("unguarded write after the commit: %v", err)
		}
	}

	var row struct{ Data, Entropy string }
	client.Raw(`SELECT data, entropy FROM shld_shares WHERE id = ?`, id).Scan(&row)
	if row.Data != "U" || row.Entropy != "user" {
		t.Errorf("share after racing write = (%q, %q), want (%q, %q): the commit overwrote a share that left project entropy", row.Data, row.Entropy, "U", "user")
	}
}
