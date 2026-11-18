package apptest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/domain/share"
	"github.com/openfort-xyz/shield/pkg/contexter"
)

// Notifier records every email and SMS body handed to it, by destination.
// With Fail set it records the message and then returns Fail, like a
// provider that errors after accepting a message.
type Notifier struct {
	Fail error
	sent map[string][]string
}

func (n *Notifier) SendEmail(_ context.Context, to, _, body, _ string) (float32, error) {
	n.sent[to] = append(n.sent[to], body)
	return 0, n.Fail
}

func (n *Notifier) SendSMS(_ context.Context, to, message string) (float32, error) {
	n.sent[to] = append(n.sent[to], message)
	return 0, n.Fail
}

// Sent returns the messages handed over for the destination, oldest first.
func (n *Notifier) Sent(to string) []string {
	return n.sent[to]
}

// Seed2FAProject is SeedEncryptedProject with 2FA enabled and an OTP rate
// limit that allows several requests per test.
func (a Apps) Seed2FAProject(t *testing.T) (dbtest.Project, string) {
	t.Helper()
	p, part := a.SeedEncryptedProject(t)
	ctx := ProjectContext(p)
	if err := a.ProjectSvc.SaveProjectRateLimits(ctx, p.ID, 20, 20); err != nil {
		t.Fatalf("SaveProjectRateLimits: %v", err)
	}
	if err := a.Project.Enable2FA(ctx); err != nil {
		t.Fatalf("Enable2FA: %v", err)
	}
	return p, part
}

// User is a project user as the user-authenticated routes see it.
type User struct {
	ID         string
	ExternalID string
	Project    dbtest.Project
}

// NewUser inserts a user into the project under a fresh external ID and
// removes the external ID's contact binding when the test ends.
func (a Apps) NewUser(t *testing.T, p dbtest.Project) User {
	t.Helper()
	u := User{ID: uuid.NewString(), ExternalID: "ext-" + uuid.NewString(), Project: p}
	dbtest.Exec(t, a.DB, `INSERT INTO shld_users (id, project_id) VALUES (?, ?)`, u.ID, p.ID)
	t.Cleanup(func() {
		if err := a.DB.Exec(`DELETE FROM shld_user_contacts WHERE external_user_id = ?`, u.ExternalID).Error; err != nil {
			t.Errorf("cleanup user contacts: %v", err)
		}
	})
	return u
}

// Context returns a context authenticated as the user.
func (u User) Context() context.Context {
	return contexter.WithExternalUserID(contexter.WithUserID(ProjectContext(u.Project), u.ID), u.ExternalID)
}

// StoreShare registers a project-encrypted share holding secret for the user.
func (a Apps) StoreShare(t *testing.T, u User, part, secret string) {
	t.Helper()
	err := a.Share.RegisterShare(u.Context(), &share.Share{Secret: secret, Entropy: share.EntropyProject},
		shareapp.WithEncryptionPart(part))
	if err != nil {
		t.Fatalf("RegisterShare: %v", err)
	}
}

// RequestEmailOTP has the project send an OTP for externalID to email and
// returns the code delivered.
func (a Apps) RequestEmailOTP(t *testing.T, p dbtest.Project, externalID, email string) string {
	t.Helper()
	before := len(a.Notifier.Sent(email))
	if err := a.Project.GenerateOTP(ProjectContext(p), externalID, false, &email, nil); err != nil {
		t.Fatalf("GenerateOTP: %v", err)
	}
	sent := a.Notifier.Sent(email)
	if len(sent) != before+1 {
		t.Fatalf("GenerateOTP delivered %d messages to %s, want 1", len(sent)-before, email)
	}
	return sent[len(sent)-1]
}

// OTPSession mints an encryption session in project p for externalID,
// verified with an OTP delivered to email.
func (a Apps) OTPSession(t *testing.T, p dbtest.Project, part, externalID, email string) string {
	t.Helper()
	code := a.RequestEmailOTP(t, p, externalID, email)
	session, err := a.Project.RegisterEncryptionSession(ProjectContext(p), part, externalID, &code)
	if err != nil {
		t.Fatalf("RegisterEncryptionSession: %v", err)
	}
	return session
}
