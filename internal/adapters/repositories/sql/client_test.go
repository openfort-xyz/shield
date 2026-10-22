package sql

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Slow and failing statements are logged; their bound values (share
// ciphertext, encryption parts, API keys) must not be.
func TestGormLogger_OmitsBoundParameters(t *testing.T) {
	if os.Getenv("DB_NAME") == "" {
		t.Skip("DB_NAME not set; skipping database-backed test")
	}
	cfg, err := GetConfigFromEnv()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	dsn, err := cfg.postgresDSN()
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	var buf bytes.Buffer
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{Logger: newGormLogger(&buf)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	const secret = "SECRET-ENCRYPTION-PART"
	db.Exec("SELECT pg_sleep(0.25), ?::text", secret)
	db.Exec("SELECT * FROM no_such_table WHERE part = ?", secret)

	out := buf.String()
	if !strings.Contains(out, "SLOW SQL") || !strings.Contains(out, "no_such_table") {
		t.Fatalf("expected a slow-query and an error line, got %q", out)
	}
	if strings.Contains(out, secret) {
		t.Errorf("a bound parameter was logged: %q", out)
	}
}
