package apptest

import (
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	sssrec "github.com/openfort-xyz/shield/internal/adapters/encryption/sss_reconstruction_strategy"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/cypher"
	"github.com/openfort-xyz/shield/pkg/random"
	"golang.org/x/crypto/bcrypt"
)

func (a Apps) register(t *testing.T, p dbtest.Project, part, secret, reference string) {
	t.Helper()
	shr := &share.Share{Secret: secret, Entropy: share.EntropyProject}
	if reference != "" {
		shr.Reference = &reference
	}
	if err := a.Share.RegisterShare(UserContext(p), shr, shareapp.WithEncryptionPart(part)); err != nil {
		t.Fatalf("RegisterShare(%q): %v", reference, err)
	}
}

func (a Apps) rotate(t *testing.T, p dbtest.Project, part string) string {
	t.Helper()
	newPart, err := a.Project.PrepareEncryptionKeyRotation(ProjectContext(p), part)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	committed, err := a.Project.CommitEncryptionKeyRotation(ProjectContext(p), part, newPart)
	if err != nil || !committed {
		t.Fatalf("commit: committed=%v err=%v", committed, err)
	}
	return newPart
}

func (a Apps) secret(t *testing.T, p dbtest.Project, part string) (string, error) {
	t.Helper()
	shr, err := a.Share.GetShare(UserContext(p), shareapp.WithEncryptionPart(part))
	if err != nil {
		return "", err
	}
	return shr.Secret, nil
}

func (a Apps) pendingCleared(t *testing.T, p dbtest.Project) bool {
	t.Helper()
	var n int64
	a.DB.Raw(`SELECT count(*) FROM shld_encryption_parts WHERE project_id = ? AND (pending_part IS NOT NULL OR pending_expires_at IS NOT NULL)`, p.ID).Scan(&n)
	return n == 0
}

func TestEncryptionKeyRotation_EveryShareDecryptsUnderNewPart(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	apps.register(t, p, part, "first", "")
	apps.register(t, p, part, "second", "second")
	others := map[string]string{
		dbtest.InsertShare(t, apps.DB, p, "user", "user-ciphertext"):   "user-ciphertext",
		dbtest.InsertShare(t, apps.DB, p, "none", "plain-share"):       "plain-share",
		dbtest.InsertShare(t, apps.DB, p, "passkey", "passkey-cipher"): "passkey-cipher",
	}

	newPart := apps.rotate(t, p, part)

	if got, err := apps.secret(t, p, newPart); err != nil || got != "first" {
		t.Errorf("default share under the new part = %q, %v; want %q", got, err, "first")
	}
	shrs, err := apps.Share.GetKeychainShares(UserContext(p), nil, shareapp.WithEncryptionPart(newPart))
	if err != nil {
		t.Fatalf("GetKeychainShares with the new part: %v", err)
	}
	decrypted := map[string]bool{}
	for _, shr := range shrs {
		if shr.Entropy == share.EntropyProject {
			decrypted[shr.Secret] = true
		}
	}
	if !decrypted["first"] || !decrypted["second"] {
		t.Errorf("project shares decrypted under the new part = %v, want first and second", decrypted)
	}
	for id, data := range others {
		if got := dbtest.ShareData(t, apps.DB, id); got != data {
			t.Errorf("share %s (not a live project share) changed: %q, want %q", id, got, data)
		}
	}
	if _, err := apps.secret(t, p, part); !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
		t.Errorf("old part after rotation: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
	}
	if !apps.pendingCleared(t, p) {
		t.Error("pending rotation not cleared after commit")
	}
}

func TestEncryptionKeyRotation_RetryConverges(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	apps.register(t, p, part, "wallet", "")
	ctx := ProjectContext(p)

	// Lost prepare response: the first new part never reached the client.
	lost, err := apps.Project.PrepareEncryptionKeyRotation(ctx, part)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	newPart, err := apps.Project.PrepareEncryptionKeyRotation(ctx, part)
	if err != nil {
		t.Fatalf("prepare again: %v", err)
	}
	if _, err := apps.Project.CommitEncryptionKeyRotation(ctx, part, lost); !errors.Is(err, projectapp.ErrInvalidEncryptionPart) {
		t.Errorf("commit with a replaced pending part: err = %v, want %v", err, projectapp.ErrInvalidEncryptionPart)
	}
	committed, err := apps.Project.CommitEncryptionKeyRotation(ctx, part, newPart)
	if err != nil || !committed {
		t.Fatalf("commit: committed=%v err=%v", committed, err)
	}

	// Lost commit response: the same commit again is a no-op success.
	committed, err = apps.Project.CommitEncryptionKeyRotation(ctx, part, newPart)
	if err != nil || committed {
		t.Errorf("repeated commit: committed=%v err=%v, want false, nil", committed, err)
	}
	if got, err := apps.secret(t, p, newPart); err != nil || got != "wallet" {
		t.Errorf("share after repeated commit = %q, %v; want %q", got, err, "wallet")
	}

	// The old part can no longer start a rotation; the new one can.
	if _, err := apps.Project.PrepareEncryptionKeyRotation(ctx, part); !errors.Is(err, projectapp.ErrInvalidEncryptionPart) {
		t.Errorf("prepare with the old part: err = %v, want %v", err, projectapp.ErrInvalidEncryptionPart)
	}
	if _, err := apps.Project.PrepareEncryptionKeyRotation(ctx, newPart); err != nil {
		t.Errorf("prepare with the new part: %v", err)
	}
}

func TestEncryptionKeyRotation_WrongCurrentPartChangesNothing(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	apps.register(t, p, part, "wallet", "")
	_, otherPart := apps.SeedEncryptedProject(t)
	ctx := ProjectContext(p)

	if _, err := apps.Project.PrepareEncryptionKeyRotation(ctx, otherPart); !errors.Is(err, projectapp.ErrInvalidEncryptionPart) {
		t.Errorf("prepare with a wrong part: err = %v, want %v", err, projectapp.ErrInvalidEncryptionPart)
	}
	newPart, err := apps.Project.PrepareEncryptionKeyRotation(ctx, part)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := apps.Project.CommitEncryptionKeyRotation(ctx, otherPart, newPart); !errors.Is(err, projectapp.ErrInvalidEncryptionPart) {
		t.Errorf("commit with a wrong current part: err = %v, want %v", err, projectapp.ErrInvalidEncryptionPart)
	}
	if got, err := apps.secret(t, p, part); err != nil || got != "wallet" {
		t.Errorf("share after rejected commit = %q, %v; want it unchanged under the old part", got, err)
	}
}

// Keys created before key checks existed are verified by decrypting the
// project's shares; prepare then stores the check so a new part that leaves
// the server before commit cannot be misused (formal/rotation MutNoBackfill).
func TestEncryptionKeyRotation_LegacyKeyWithoutCheck(t *testing.T) {
	apps := New(t)
	key, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	storedPart, projectPart, err := sssrec.NewSSSReconstructionStrategy().Split(key)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	p := dbtest.SeedProject(t, apps.DB, string(storedPart), true)
	part := string(projectPart)
	apps.register(t, p, part, "wallet", "")
	_, otherPart := apps.SeedEncryptedProject(t)

	if _, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), otherPart); !errors.Is(err, projectapp.ErrInvalidEncryptionPart) {
		t.Errorf("legacy prepare with a wrong part: err = %v, want %v", err, projectapp.ErrInvalidEncryptionPart)
	}
	newPart, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), part)
	if err != nil {
		t.Fatalf("legacy prepare: %v", err)
	}

	// Before commit, the pending part must already be rejected for writes.
	err = apps.Share.RegisterShare(UserContext(p), &share.Share{Secret: "early", Entropy: share.EntropyProject, Reference: ptr("early")},
		shareapp.WithEncryptionPart(newPart))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
		t.Errorf("write with the pending part before commit: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
	}

	if committed, err := apps.Project.CommitEncryptionKeyRotation(ProjectContext(p), part, newPart); err != nil || !committed {
		t.Fatalf("legacy commit: committed=%v err=%v", committed, err)
	}
	if got, err := apps.secret(t, p, newPart); err != nil || got != "wallet" {
		t.Errorf("legacy share after rotation = %q, %v; want %q", got, err, "wallet")
	}
}

func TestEncryptionKeyRotation_PreconditionErrors(t *testing.T) {
	apps := New(t)

	t.Run("commit without prepare", func(t *testing.T) {
		p, part := apps.SeedEncryptedProject(t)
		_, otherPart := apps.SeedEncryptedProject(t)
		if _, err := apps.Project.CommitEncryptionKeyRotation(ProjectContext(p), part, otherPart); !errors.Is(err, projectapp.ErrNoPendingRotation) {
			t.Errorf("err = %v, want %v", err, projectapp.ErrNoPendingRotation)
		}
	})

	t.Run("expired pending rotation", func(t *testing.T) {
		p, part := apps.SeedEncryptedProject(t)
		newPart, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), part)
		if err != nil {
			t.Fatalf("prepare: %v", err)
		}
		dbtest.Exec(t, apps.DB, `UPDATE shld_encryption_parts SET pending_expires_at = now() - interval '1 minute' WHERE project_id = ?`, p.ID)
		if _, err := apps.Project.CommitEncryptionKeyRotation(ProjectContext(p), part, newPart); !errors.Is(err, projectapp.ErrRotationExpired) {
			t.Errorf("err = %v, want %v", err, projectapp.ErrRotationExpired)
		}
		if _, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), part); err != nil {
			t.Errorf("prepare after expiry: %v", err)
		}
	})

	t.Run("key not migrated", func(t *testing.T) {
		p := dbtest.SeedProject(t, apps.DB, "stored", false)
		if _, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), "part"); !errors.Is(err, projectapp.ErrProjectNotMigrated) {
			t.Errorf("err = %v, want %v", err, projectapp.ErrProjectNotMigrated)
		}
	})

	t.Run("no encryption key", func(t *testing.T) {
		p := dbtest.SeedProject(t, apps.DB, "", true)
		if _, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), "part"); !errors.Is(err, projectapp.ErrEncryptionNotConfigured) {
			t.Errorf("err = %v, want %v", err, projectapp.ErrEncryptionNotConfigured)
		}
	})
}

func TestEncryptionKeyRotation_ConcurrentCommitsSerialize(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	apps.register(t, p, part, "wallet", "")
	newPart, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), part)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	var wg sync.WaitGroup
	results := make([]bool, 2)
	errs := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = apps.Project.CommitEncryptionKeyRotation(ProjectContext(p), part, newPart)
		}()
	}
	wg.Wait()

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("concurrent commits: errs = %v", errs)
	}
	if results[0] == results[1] {
		t.Errorf("concurrent commits committed = %v; want exactly one", results)
	}
	if got, err := apps.secret(t, p, newPart); err != nil || got != "wallet" {
		t.Errorf("share after concurrent commits = %q, %v; want %q", got, err, "wallet")
	}
}

func TestEncryptionKeyRotation_StalePartsRejectedAfterCommit(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	apps.register(t, p, part, "wallet", "")
	session, err := apps.Project.RegisterEncryptionSession(ProjectContext(p), part, p.UserID, nil)
	if err != nil {
		t.Fatalf("RegisterEncryptionSession: %v", err)
	}

	newPart := apps.rotate(t, p, part)

	if _, err := apps.Share.GetShare(UserContext(p), shareapp.WithEncryptionSession(session)); !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
		t.Errorf("session opened with the old part: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
	}
	err = apps.Share.RegisterShare(UserContext(p), &share.Share{Secret: "late", Entropy: share.EntropyProject, Reference: ptr("late")},
		shareapp.WithEncryptionPart(part))
	if !errors.Is(err, shareapp.ErrInvalidEncryptionPart) {
		t.Errorf("write with the old part: err = %v, want %v", err, shareapp.ErrInvalidEncryptionPart)
	}
	apps.register(t, p, newPart, "fresh", "fresh")
}

func ptr(s string) *string { return &s }

// Import takes ciphertext as-is; project-entropy ciphertext must come from a
// project key Shield can vouch for, and export never produces it.
func TestImportShare_ProjectEntropyRejected(t *testing.T) {
	apps := New(t)
	p, _ := apps.SeedEncryptedProject(t)

	err := apps.Share.ImportShare(ProjectContext(p), &share.Share{UserID: p.UserID, Secret: "ciphertext", Entropy: share.EntropyProject, Reference: ptr("imported")})

	if !errors.Is(err, shareapp.ErrProjectEntropyImport) {
		t.Errorf("import of a project-entropy share: err = %v, want %v", err, shareapp.ErrProjectEntropyImport)
	}
	if n := apps.ShareCount(t, p); n != 0 {
		t.Errorf("imported project share stored: %d rows", n)
	}
}

// With no key check and no share to decrypt, nothing proves the caller's
// part; storing a check from it could lock out the real part.
func TestEncryptionKeyRotation_LegacyWithoutSharesRefused(t *testing.T) {
	apps := New(t)
	_, someonesPart := apps.SeedEncryptedProject(t)
	p := dbtest.SeedProject(t, apps.DB, "AQIDBA==", true)

	_, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), someonesPart)

	if !errors.Is(err, projectapp.ErrUnverifiableEncryptionKey) {
		t.Errorf("prepare on a legacy project without shares: err = %v, want %v", err, projectapp.ErrUnverifiableEncryptionKey)
	}
	var check *string
	apps.DB.Raw(`SELECT key_check FROM shld_encryption_parts WHERE project_id = ?`, p.ID).Scan(&check)
	if check != nil {
		t.Errorf("key check backfilled from an unverified part: %q", *check)
	}
}

func TestEncryptionKeyRotation_CommitWithoutPrepareSamePartTwice(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	if _, err := apps.Project.CommitEncryptionKeyRotation(ProjectContext(p), part, part); !errors.Is(err, projectapp.ErrNoPendingRotation) {
		t.Errorf("commit with the current part as both parts: err = %v, want %v", err, projectapp.ErrNoPendingRotation)
	}
}

// A share that does not decrypt under a verified current key is corrupt data,
// not a wrong part: report it as an internal error and change nothing.
func TestEncryptionKeyRotation_CorruptShareIsInternal(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	apps.register(t, p, part, "wallet", "")
	dbtest.InsertShare(t, apps.DB, p, "project", "Y29ycnVwdC1jaXBoZXJ0ZXh0LXRoYXQtaXMtbG9uZw==")
	newPart, err := apps.Project.PrepareEncryptionKeyRotation(ProjectContext(p), part)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	_, err = apps.Project.CommitEncryptionKeyRotation(ProjectContext(p), part, newPart)

	if !errors.Is(err, projectapp.ErrInternal) {
		t.Errorf("commit with a corrupt share: err = %v, want %v", err, projectapp.ErrInternal)
	}
	if got, err := apps.secret(t, p, part); err != nil || got != "wallet" {
		t.Errorf("share after failed commit = %q, %v; want it unchanged under the old part", got, err)
	}
}

// Soft-deleted wallet shares stay restorable: they move to the new key too.
func TestEncryptionKeyRotation_ReencryptsSoftDeletedShares(t *testing.T) {
	apps := New(t)
	p, part := apps.SeedEncryptedProject(t)
	apps.register(t, p, part, "wallet", "")
	var id string
	apps.DB.Raw(`SELECT id FROM shld_shares WHERE user_id = ?`, p.UserID).Scan(&id)
	dbtest.Exec(t, apps.DB, `UPDATE shld_shares SET deleted_at = now() WHERE id = ?`, id)

	newPart := apps.rotate(t, p, part)

	var stored string
	apps.DB.Raw(`SELECT part FROM shld_encryption_parts WHERE project_id = ?`, p.ID).Scan(&stored)
	key, err := sssrec.NewSSSReconstructionStrategy().Reconstruct(project.StoredPart(stored), project.ProjectPart(newPart))
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}
	if plain, err := cypher.Decrypt(dbtest.ShareData(t, apps.DB, id), key); err != nil || plain != "wallet" {
		t.Errorf("soft-deleted share under the new key = %q, %v; want %q", plain, err, "wallet")
	}
}

// A prepare that authenticated with a key a concurrent commit then replaced
// must not mint a new pending key from the revoked one.
func TestAPIKeyRotation_PrepareWithRevokedKeyRefused(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", true)
	ctx := ProjectContext(p)
	exp := time.Now().Add(time.Minute)

	pending, err := apps.ProjectSvc.PrepareAPIKeyRotation(ctx, p.ID, p.APIKey, exp)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, committed, err := apps.ProjectSvc.CommitAPIKeyRotation(ctx, p.ID, pending); err != nil || !committed {
		t.Fatalf("commit: committed=%v err=%v", committed, err)
	}

	if _, err := apps.ProjectSvc.PrepareAPIKeyRotation(ctx, p.ID, p.APIKey, exp); !errors.Is(err, domainErrors.ErrAPIKeyRevoked) {
		t.Errorf("prepare with the revoked key: err = %v, want %v", err, domainErrors.ErrAPIKeyRevoked)
	}
	var stillPending *string
	apps.DB.Raw(`SELECT pending_api_key FROM shld_projects WHERE id = ?`, p.ID).Scan(&stillPending)
	if stillPending != nil {
		t.Errorf("revoked key minted a pending key %q", *stillPending)
	}
}

// withSecret gives a seeded project a real API secret (stored as its bcrypt
// hash, like Create) and returns the secret.
func withSecret(t *testing.T, apps Apps, p dbtest.Project) string {
	t.Helper()
	raw, err := random.GenerateRandomString(32)
	if err != nil {
		t.Fatal(err)
	}
	secret := hex.EncodeToString([]byte(raw))[:64]
	hash, err := bcrypt.GenerateFromPassword(project.APISecretBytes(secret), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	dbtest.Exec(t, apps.DB, `UPDATE shld_projects SET api_secret = ? WHERE id = ?`, string(hash), p.ID)
	return secret
}

func pendingSecretStored(apps Apps, p dbtest.Project) bool {
	var pending *string
	apps.DB.Raw(`SELECT pending_api_secret FROM shld_projects WHERE id = ?`, p.ID).Scan(&pending)
	return pending != nil
}

// A secret prepare that authenticated with a key a concurrent commit then
// replaced must not store a pending secret.
func TestAPISecretRotation_PrepareWithRevokedKeyRefused(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", true)
	secret := withSecret(t, apps, p)
	ctx := ProjectContext(p)
	exp := time.Now().Add(time.Minute)

	pending, err := apps.ProjectSvc.PrepareAPIKeyRotation(ctx, p.ID, p.APIKey, exp)
	if err != nil {
		t.Fatalf("prepare key: %v", err)
	}
	if _, committed, err := apps.ProjectSvc.CommitAPIKeyRotation(ctx, p.ID, pending); err != nil || !committed {
		t.Fatalf("commit key: committed=%v err=%v", committed, err)
	}

	if _, err := apps.ProjectSvc.PrepareAPISecretRotation(ctx, p.ID, p.APIKey, secret, exp); !errors.Is(err, domainErrors.ErrAPIKeyRevoked) {
		t.Errorf("secret prepare with the revoked key: err = %v, want %v", err, domainErrors.ErrAPIKeyRevoked)
	}
	if pendingSecretStored(apps, p) {
		t.Errorf("revoked key stored a pending secret")
	}
}

// A request that authenticated with the old secret just before a secret
// commit must not mint a new secret from it.
func TestAPISecretRotation_PrepareWithRevokedSecretRefused(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", true)
	oldSecret := withSecret(t, apps, p)
	ctx := ProjectContext(p)
	exp := time.Now().Add(time.Minute)

	newSecret, err := apps.ProjectSvc.PrepareAPISecretRotation(ctx, p.ID, p.APIKey, oldSecret, exp)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if committed, err := apps.ProjectSvc.CommitAPISecretRotation(ctx, p.ID, p.APIKey, newSecret); err != nil || !committed {
		t.Fatalf("commit: committed=%v err=%v", committed, err)
	}

	if _, err := apps.ProjectSvc.PrepareAPISecretRotation(ctx, p.ID, p.APIKey, oldSecret, exp); !errors.Is(err, domainErrors.ErrAPISecretRevoked) {
		t.Errorf("prepare with the revoked secret: err = %v, want %v", err, domainErrors.ErrAPISecretRevoked)
	}
	if pendingSecretStored(apps, p) {
		t.Errorf("revoked secret stored a pending secret")
	}
	if _, err := apps.ProjectSvc.PrepareAPISecretRotation(ctx, p.ID, p.APIKey, newSecret, exp); err != nil {
		t.Errorf("prepare with the current secret: %v", err)
	}
}

// Two commits of the same pending secret serialize on the project row: one
// swaps, the other sees the secret already current.
func TestAPISecretRotation_ConcurrentCommitsSerialize(t *testing.T) {
	apps := New(t)
	p := dbtest.SeedProject(t, apps.DB, "", true)
	secret := withSecret(t, apps, p)
	ctx := ProjectContext(p)

	pending, err := apps.ProjectSvc.PrepareAPISecretRotation(ctx, p.ID, p.APIKey, secret, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	var wg sync.WaitGroup
	results := make([]bool, 2)
	errs := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = apps.ProjectSvc.CommitAPISecretRotation(ctx, p.ID, p.APIKey, pending)
		}()
	}
	wg.Wait()

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("concurrent commits: errs = %v", errs)
	}
	if results[0] == results[1] {
		t.Errorf("concurrent commits committed = %v; want exactly one", results)
	}
	var row struct {
		APISecret        string
		PendingAPISecret *string
	}
	apps.DB.Raw(`SELECT api_secret, pending_api_secret FROM shld_projects WHERE id = ?`, p.ID).Scan(&row)
	if row.PendingAPISecret != nil || bcrypt.CompareHashAndPassword([]byte(row.APISecret), project.APISecretBytes(pending)) != nil {
		t.Errorf("row after concurrent commits: pending=%v, current matches pending secret = false", row.PendingAPISecret)
	}
}
