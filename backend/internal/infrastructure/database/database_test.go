package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestResolveMigrationsPathFallback(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	got := resolveMigrationsPath()
	if got != "file://./migrations" {
		t.Fatalf("expected default fallback path, got %q", got)
	}
}

func TestResolveMigrationsPathFindsCandidate(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	tmp := t.TempDir()
	migrationDir := filepath.Join(tmp, "migrations")
	if err := os.MkdirAll(migrationDir, 0o755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	got := resolveMigrationsPath()
	if !strings.HasPrefix(got, "file://") || !strings.Contains(got, "migrations") {
		t.Fatalf("expected migrations file path, got %q", got)
	}
}

func TestWithTenantContextReturnsSameDB(t *testing.T) {
	db := &sqlx.DB{}
	out := WithTenantContext(db, "tenant-1")
	if out != db {
		t.Fatal("expected same DB pointer returned")
	}
}
