package main

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/testutil"
)

func openEssentialSeederTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("IF_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set IF_TEST_POSTGRES_DSN to run essential config seeder integration tests")
	}
	testutil.RequireSafeTestDSN(t, dsn, "IF_TEST_POSTGRES_DSN")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect postgres: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createSeederTempTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ddl := `
	CREATE TEMP TABLE tenants (
		id TEXT PRIMARY KEY DEFAULT md5(random()::text || clock_timestamp()::text),
		tenant_code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		slug TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL
	);
	CREATE TEMP TABLE tenant_groups (
		id TEXT PRIMARY KEY DEFAULT md5(random()::text || clock_timestamp()::text),
		tenant_id TEXT NOT NULL,
		name TEXT NOT NULL,
		slug TEXT NOT NULL,
		description TEXT,
		role_type TEXT NOT NULL,
		is_system_group BOOLEAN NOT NULL DEFAULT false,
		status TEXT NOT NULL,
		UNIQUE (tenant_id, slug)
	);
	CREATE TEMP TABLE users (
		id TEXT PRIMARY KEY DEFAULT md5(random()::text || clock_timestamp()::text),
		email TEXT NOT NULL UNIQUE
	);
	CREATE TEMP TABLE group_members (
		group_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		is_group_admin BOOLEAN NOT NULL DEFAULT false,
		added_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (group_id, user_id)
	);`

	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("failed creating seeder temp tables: %v", err)
	}
}

func TestEnsureSystemBootstrap_SeedsSecurityReviewersGroupIdempotently(t *testing.T) {
	db := openEssentialSeederTestDB(t)
	createSeederTempTables(t, db)

	if _, err := db.Exec(`INSERT INTO users (email) VALUES ('admin@imagefactory.local')`); err != nil {
		t.Fatalf("failed seeding admin user: %v", err)
	}

	if err := ensureSystemBootstrap(db); err != nil {
		t.Fatalf("ensureSystemBootstrap first call failed: %v", err)
	}
	if err := ensureSystemBootstrap(db); err != nil {
		t.Fatalf("ensureSystemBootstrap second call failed: %v", err)
	}

	var tenantCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE tenant_code='sysadmin'`).Scan(&tenantCount); err != nil {
		t.Fatalf("failed counting sysadmin tenants: %v", err)
	}
	if tenantCount != 1 {
		t.Fatalf("expected 1 sysadmin tenant, got %d", tenantCount)
	}

	var groupCount int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM tenant_groups g
		JOIN tenants t ON t.id = g.tenant_id
		WHERE t.tenant_code='sysadmin'
	`).Scan(&groupCount); err != nil {
		t.Fatalf("failed counting sysadmin groups: %v", err)
	}
	if groupCount != 3 {
		t.Fatalf("expected 3 sysadmin groups (admin/viewer/security-reviewers), got %d", groupCount)
	}

	var roleType string
	var isSystem bool
	if err := db.QueryRow(`
		SELECT g.role_type, g.is_system_group
		FROM tenant_groups g
		JOIN tenants t ON t.id = g.tenant_id
		WHERE t.tenant_code='sysadmin' AND g.slug='security-reviewers'
	`).Scan(&roleType, &isSystem); err != nil {
		t.Fatalf("failed loading security-reviewers group: %v", err)
	}
	if roleType != "security_reviewer" {
		t.Fatalf("expected security-reviewers role_type=security_reviewer, got %q", roleType)
	}
	if !isSystem {
		t.Fatalf("expected security-reviewers to be a system group")
	}

	var adminMemberships int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM group_members gm
		JOIN tenant_groups g ON g.id = gm.group_id
		JOIN tenants t ON t.id = g.tenant_id
		JOIN users u ON u.id = gm.user_id
		WHERE t.tenant_code='sysadmin'
		  AND g.role_type='system_administrator'
		  AND u.email='admin@imagefactory.local'
	`).Scan(&adminMemberships); err != nil {
		t.Fatalf("failed counting sysadmin memberships: %v", err)
	}
	if adminMemberships != 1 {
		t.Fatalf("expected one sysadmin membership for admin user, got %d", adminMemberships)
	}
}
