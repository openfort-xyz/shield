// Package apptest wires the share and project applications to a real
// database for behaviour tests that cross the application boundary.
package apptest

import (
	"context"
	"testing"
	"time"

	"github.com/openfort-xyz/shield/internal/adapters/encryption"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/bunt"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/bunt/encryptionpartsrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/keychainrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/projectrepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/sharerepo"
	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/userrepo"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shamirjob"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/internal/core/services/projectsvc"
	"github.com/openfort-xyz/shield/internal/core/services/sharesvc"
	"github.com/openfort-xyz/shield/pkg/contexter"
)

// Apps holds real applications and the repositories behind them.
type Apps struct {
	DB          *sql.Client
	Share       *shareapp.ShareApplication
	Project     *projectapp.ProjectApplication
	ProjectSvc  services.ProjectService
	ProjectRepo repositories.ProjectRepository
	ShareRepo   repositories.ShareRepository
}

// New wires the applications to the database configured by DB_* variables,
// skipping the test when DB_NAME is unset.
func New(t *testing.T) Apps {
	t.Helper()
	return NewWithShareRepo(t, func(r repositories.ShareRepository) repositories.ShareRepository { return r })
}

// NewWithShareRepo is New with the share repository wrapped, e.g. to pause a
// write between key reconstruction and persistence.
func NewWithShareRepo(t *testing.T, wrap func(repositories.ShareRepository) repositories.ShareRepository) Apps {
	t.Helper()
	client := dbtest.RequireDB(t)

	buntClient, err := bunt.New()
	if err != nil {
		t.Fatalf("bunt: %v", err)
	}

	projectRepo := projectrepo.New(client)
	shareRepo := wrap(sharerepo.New(client))
	keychainRepo := keychainrepo.New(client)
	userRepo := userrepo.New(client)
	encryptionPartsRepo := encryptionpartsrepo.New(buntClient)
	factory := encryption.NewEncryptionFactory(encryptionPartsRepo, projectRepo)
	projectSvc := projectsvc.New(projectRepo, time.Minute)
	shareSvc := sharesvc.New(shareRepo, keychainRepo, factory)

	return Apps{
		DB:          client,
		Share:       shareapp.New(shareSvc, shareRepo, projectRepo, userRepo, keychainRepo, factory, shamirjob.New(projectRepo, shareRepo)),
		Project:     projectapp.New(projectSvc, projectRepo, nil, nil, shareRepo, nil, nil, factory, encryptionPartsRepo, nil, nil, nil),
		ProjectSvc:  projectSvc,
		ProjectRepo: projectRepo,
		ShareRepo:   shareRepo,
	}
}

// ProjectContext returns a context authenticated as the project.
func ProjectContext(p dbtest.Project) context.Context {
	return contexter.WithProjectID(context.Background(), p.ID)
}

// UserContext returns a context authenticated as the project's user.
func UserContext(p dbtest.Project) context.Context {
	return contexter.WithUserID(ProjectContext(p), p.UserID)
}

// SeedEncryptedProject seeds a project and registers its encryption key
// through the project application, returning the project's half of the key.
func (a Apps) SeedEncryptedProject(t *testing.T) (dbtest.Project, string) {
	t.Helper()
	p := dbtest.SeedProject(t, a.DB, "", false)
	part, err := a.Project.RegisterEncryptionKey(ProjectContext(p))
	if err != nil {
		t.Fatalf("RegisterEncryptionKey: %v", err)
	}
	return p, part
}

// ShareCount returns how many share rows the project's user has.
func (a Apps) ShareCount(t *testing.T, p dbtest.Project) int64 {
	t.Helper()
	var n int64
	if err := a.DB.Raw(`SELECT count(*) FROM shld_shares WHERE user_id = ?`, p.UserID).Scan(&n).Error; err != nil {
		t.Fatalf("count shares: %v", err)
	}
	return n
}
