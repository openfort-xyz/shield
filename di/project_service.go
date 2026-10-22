package di

import (
	"sync"
	"time"

	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/internal/core/services/projectsvc"
)

const projectCacheTTL = 60 * time.Second

// ProvideProjectService returns a process-wide singleton project service. The
// service owns the API-key cache, so the auth middleware and the project
// application must share one instance for a key rotation's cache eviction to
// reach the middleware.
//
// It is a plain provider (not a wire injector) so that Wire generates calls to
// this memoized function, as with ProvideSQL.
func ProvideProjectService() (services.ProjectService, error) {
	projectServiceOnce.Do(func() {
		repo, err := ProvideSQLProjectRepository()
		if err != nil {
			projectServiceErr = err
			return
		}
		projectService = projectsvc.New(repo, projectCacheTTL)
	})
	return projectService, projectServiceErr
}

var (
	projectServiceOnce sync.Once
	projectService     services.ProjectService
	projectServiceErr  error
)
