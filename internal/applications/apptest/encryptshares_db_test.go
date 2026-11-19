package apptest

import (
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
)

// EncryptProjectShares reads the project's entropy-none shares, encrypts
// them, then writes them back. A share updated in between keeps the update
// instead of being overwritten with the stale secret.
func TestEncryptProjectShares_ConcurrentUpdateNotOverwritten(t *testing.T) {
	tests := []struct {
		name        string
		update      *share.Share
		wantEntropy string
		wantData    string
	}{
		{"secret changed", &share.Share{Secret: "new-secret"}, "none", "new-secret"},
		{"moved to user entropy, data unchanged", &share.Share{Entropy: share.EntropyUser,
			EncryptionParameters: &share.EncryptionParameters{Salt: "salt", Iterations: 1000, Length: 256, Digest: "SHA-256"}},
			"user", "plain-secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parking := NewParkingShareRepo()
			apps := NewWithShareRepo(t, parking.Wrap)
			p, part := apps.SeedEncryptedProject(t)
			id := dbtest.InsertShare(t, apps.DB, p, "none", "plain-secret")

			parking.Armed.Store(true)
			errc := make(chan error, 1)
			go func() { errc <- apps.Project.EncryptProjectShares(ProjectContext(p), part) }()
			<-parking.Parked
			parking.Armed.Store(false)
			if _, err := apps.Share.UpdateShare(UserContext(p), tt.update, shareReference(t, apps, id)); err != nil {
				t.Fatalf("concurrent UpdateShare: %v", err)
			}
			close(parking.Release)

			if err := <-errc; err != nil {
				t.Errorf("EncryptProjectShares: %v", err)
			}
			var got struct{ Entropy, Data string }
			apps.DB.Raw(`SELECT entropy, data FROM shld_shares WHERE id = ?`, id).Scan(&got)
			if got.Entropy != tt.wantEntropy || got.Data != tt.wantData {
				t.Errorf("share after the racing update = (%s, %q), want (%s, %q)", got.Entropy, got.Data, tt.wantEntropy, tt.wantData)
			}
		})
	}
}
