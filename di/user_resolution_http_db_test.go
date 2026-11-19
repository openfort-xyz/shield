package di

import (
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

// linkedProjects lists the project of every user linked to the identity.
func linkedProjects(t *testing.T, providerID, externalUserID string) []string {
	t.Helper()
	var projects []string
	err := dbtest.RequireDB(t).Raw(`SELECT u.project_id FROM shld_external_users eu
		JOIN shld_users u ON u.id = eu.user_id
		WHERE eu.provider_id = ? AND eu.external_user_id = ?`, providerID, externalUserID).Scan(&projects).Error
	if err != nil {
		t.Fatalf("read linked users: %v", err)
	}
	return projects
}

func projectID(t *testing.T, apiKey string) string {
	t.Helper()
	var id string
	if err := dbtest.RequireDB(t).Raw(`SELECT id FROM shld_projects WHERE api_key = ?`, apiKey).Scan(&id).Error; err != nil {
		t.Fatalf("read project id: %v", err)
	}
	return id
}

func TestCreateUser_ProviderFromAnotherProject_HTTP(t *testing.T) {
	a := newUserProject(t)
	b := newUserProject(t)
	bUser := "b-user-" + uuid.NewString()
	createAsA := func() reply {
		return a.do(http.MethodPost, "/user", a.apiKey,
			map[string]any{"external_user_id": bUser, "provider_id": b.providerID})
	}

	if got := createAsA(); got.Status != http.StatusNotFound || got.str("code") != "PV_NOT_FOUND" {
		t.Errorf("A links an unseen identity to B's provider: got %d %v, want 404 PV_NOT_FOUND", got.Status, got.Body)
	}

	expect(t, "B's user registers a share", b.asUser(t, http.MethodPost, "/shares", bUser, map[string]any{
		"secret": "b-secret", "entropy": "project", "encryption_part": b.part,
	}), http.StatusCreated, "")
	bProject := projectID(t, b.apiKey)
	for _, p := range linkedProjects(t, b.providerID, bUser) {
		if p != bProject {
			t.Errorf("B's identity resolves to a user in project %s, want only B's project %s", p, bProject)
		}
	}

	if leaked := createAsA(); leaked.Status != http.StatusNotFound || leaked.str("user_id") != "" {
		t.Errorf("A looks up B's existing identity: got %d %v, want 404 without a user ID", leaked.Status, leaked.Body)
	}

	got := b.asUser(t, http.MethodGet, "/shares", bUser, nil, "X-Encryption-Part", b.part)
	expect(t, "B's user reads the share", got, http.StatusOK, "")
	if got.str("secret") != "b-secret" {
		t.Fatalf("secret = %q, want b-secret", got.str("secret"))
	}
}

func TestCreateUser_OwnProvider_HTTP(t *testing.T) {
	a := newUserProject(t)
	ext := "user-" + uuid.NewString()
	create := func() reply {
		return a.do(http.MethodPost, "/user", a.apiKey,
			map[string]any{"external_user_id": ext, "provider_id": a.providerID})
	}

	first := create()
	expect(t, "create user", first, http.StatusCreated, "")
	again := create()
	expect(t, "create the same user again", again, http.StatusCreated, "")
	if first.str("user_id") == "" || again.str("user_id") != first.str("user_id") {
		t.Fatalf("user IDs %q then %q, want the same non-empty ID", first.str("user_id"), again.str("user_id"))
	}

	expect(t, "the pre-created user registers a share", a.asUser(t, http.MethodPost, "/shares", ext, map[string]any{
		"secret": "a-secret", "entropy": "project", "encryption_part": a.part,
	}), http.StatusCreated, "")
	var owner string
	err := dbtest.RequireDB(t).Raw(`SELECT s.user_id FROM shld_shares s
		JOIN shld_external_users eu ON eu.user_id = s.user_id
		WHERE eu.provider_id = ? AND eu.external_user_id = ?`, a.providerID, ext).Scan(&owner).Error
	if err != nil {
		t.Fatalf("read share owner: %v", err)
	}
	if owner != first.str("user_id") {
		t.Fatalf("share owned by %q, want the pre-created user %q", owner, first.str("user_id"))
	}
}

func TestFirstLogin_ConcurrentRequests_HTTP(t *testing.T) {
	const rounds = 10
	p := newUserProject(t)
	for range rounds {
		firstLoginRace(t, p)
	}
}

// firstLoginRace sends concurrent first logins for one fresh identity. A single
// round can miss a broken lock, so the caller repeats it.
func firstLoginRace(t *testing.T, p userProject) {
	t.Helper()
	const requests = 32
	ext := "racer-" + uuid.NewString()
	headers := []string{"X-API-Key", p.apiKey, "X-Auth-Provider", "custom", "Authorization", "Bearer " + p.token(t, ext)}

	statuses := make([]int, requests)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			statuses[i] = p.do(http.MethodGet, "/shares", "", nil, headers...).Status
		}()
	}
	close(start)
	wg.Wait()

	for i, status := range statuses {
		if status >= http.StatusInternalServerError {
			t.Errorf("request %d: status %d", i, status)
		}
	}
	if linked := linkedProjects(t, p.providerID, ext); len(linked) != 1 {
		t.Fatalf("%d concurrent first logins created %d users, want 1", requests, len(linked))
	}
}

func TestLogin_IgnoresLinkFromAnotherProject_HTTP(t *testing.T) {
	a := newUserProject(t)
	b := newUserProject(t)
	bUser := "b-user-" + uuid.NewString()
	db := dbtest.RequireDB(t)
	aUser := uuid.NewString()
	dbtest.Exec(t, db, `INSERT INTO shld_users (id, project_id) VALUES (?, ?)`, aUser, projectID(t, a.apiKey))
	dbtest.Exec(t, db, `INSERT INTO shld_external_users (id, user_id, external_user_id, provider_id) VALUES (?, ?, ?, ?)`,
		uuid.NewString(), aUser, bUser, b.providerID)

	expect(t, "B's user registers a share", b.asUser(t, http.MethodPost, "/shares", bUser, map[string]any{
		"secret": "b-secret", "entropy": "project", "encryption_part": b.part,
	}), http.StatusCreated, "")
	var owners []string
	err := db.Raw(`SELECT u.project_id FROM shld_shares s JOIN shld_users u ON u.id = s.user_id
		WHERE u.id = ? OR u.project_id = ?`, aUser, projectID(t, b.apiKey)).Scan(&owners).Error
	if err != nil {
		t.Fatalf("read share owners: %v", err)
	}
	if len(owners) != 1 || owners[0] != projectID(t, b.apiKey) {
		t.Fatalf("B's share is owned by users in projects %v, want only B's project", owners)
	}
}
