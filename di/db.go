package di

import (
	"sync"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql"
)

// ProvideSQL returns a process-wide singleton SQL client. It is memoized so the
// many repository/service/application injectors that depend on it all share a
// single connection pool instead of each opening its own via gorm.Open.
//
// It is a plain provider (not a wire injector) so that Wire generates calls to
// this memoized function rather than re-running sql.New for every dependent.
func ProvideSQL() (*sql.Client, error) {
	sqlOnce.Do(func() {
		cfg, err := sql.GetConfigFromEnv()
		if err != nil {
			sqlErr = err
			return
		}
		sqlClient, sqlErr = sql.New(cfg)
	})
	return sqlClient, sqlErr
}

var (
	sqlOnce   sync.Once
	sqlClient *sql.Client
	sqlErr    error
)
