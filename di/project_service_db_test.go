package di

import (
	"testing"

	"github.com/openfort-xyz/shield/internal/adapters/repositories/sql/dbtest"
)

// The project service owns the API-key cache; the auth middleware and the
// project application must share one instance so an eviction reaches both.
func TestProvideProjectService_SingleInstance(t *testing.T) {
	dbtest.RequireDB(t)

	first, err := ProvideProjectService()
	if err != nil {
		t.Fatalf("ProvideProjectService: %v", err)
	}
	second, err := ProvideProjectService()
	if err != nil {
		t.Fatalf("ProvideProjectService: %v", err)
	}

	if first != second {
		t.Fatalf("ProvideProjectService returned two instances (%p, %p); cache evictions would not reach the auth middleware", first, second)
	}
}
