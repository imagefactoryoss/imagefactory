package database

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
)

// NewConnection creates a new database connection
func NewConnection(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Migrate runs database migrations
func Migrate(databaseURL, schema string) error {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database for migration: %w", err)
	}
	defer db.Close()
	schema = normalizeSchema(schema)
	if err := ensureSchemaExists(db, schema); err != nil {
		return err
	}

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{SchemaName: schema})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrationsPath := resolveMigrationsPath()
	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// MigrateDown rolls back database migrations
func MigrateDown(databaseURL string, steps int, schema string) error {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database for rollback: %w", err)
	}
	defer db.Close()
	schema = normalizeSchema(schema)
	if err := ensureSchemaExists(db, schema); err != nil {
		return err
	}

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{SchemaName: schema})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrationsPath := resolveMigrationsPath()
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Steps(-steps); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	return nil
}

// HealthCheck checks if the database is healthy
func HealthCheck(db *sqlx.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

func resolveMigrationsPath() string {
	candidates := []string{
		"./migrations",
		"./backend/migrations",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return "file://" + path
		}
	}
	// Default to original path to preserve prior behavior.
	return "file://./migrations"
}

// TenantContext represents tenant-specific database context
type TenantContext struct {
	TenantID string
	Schema   string
}

// SchemaReadiness captures key schema signals that help detect search_path/schema drift.
type SchemaReadiness struct {
	CurrentSchema        string
	SchemaMigrations     bool
	TenantsDomainColumn  bool
	TenantsDomainNotNull bool
}

// CheckSchemaReadiness returns schema diagnostics for the current database session/schema.
func CheckSchemaReadiness(ctx context.Context, db *sqlx.DB) (SchemaReadiness, error) {
	readiness := SchemaReadiness{}
	if db == nil {
		return readiness, fmt.Errorf("database is nil")
	}

	if err := db.GetContext(ctx, &readiness.CurrentSchema, `SELECT current_schema()`); err != nil {
		return readiness, fmt.Errorf("resolve current schema: %w", err)
	}

	if err := db.GetContext(ctx, &readiness.SchemaMigrations, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'schema_migrations'
		)
	`); err != nil {
		return readiness, fmt.Errorf("check schema_migrations table: %w", err)
	}

	type tenantDomainState struct {
		HasDomainColumn bool `db:"has_domain_column"`
		DomainNotNull   bool `db:"domain_not_null"`
	}
	var state tenantDomainState
	if err := db.GetContext(ctx, &state, `
		SELECT
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND table_name = 'tenants'
				  AND column_name = 'domain'
			) AS has_domain_column,
			EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND table_name = 'tenants'
				  AND column_name = 'domain'
				  AND is_nullable = 'NO'
			) AS domain_not_null
	`); err != nil {
		return readiness, fmt.Errorf("check tenants.domain column: %w", err)
	}
	readiness.TenantsDomainColumn = state.HasDomainColumn
	readiness.TenantsDomainNotNull = state.DomainNotNull
	return readiness, nil
}

// WithTenantContext adds tenant context to the database connection
func WithTenantContext(db *sqlx.DB, tenantID string) *sqlx.DB {
	// In a real implementation, you might set the search_path for tenant isolation
	// For now, we'll just return the database connection
	// You can implement row-level security or schema-based isolation here
	return db
}

// Transaction executes a function within a database transaction
func Transaction(db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

var schemaIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeSchema(schema string) string {
	trimmed := strings.TrimSpace(schema)
	if trimmed == "" {
		return "public"
	}
	return trimmed
}

func ensureSchemaExists(db *sqlx.DB, schema string) error {
	if !schemaIdentifierPattern.MatchString(schema) {
		return fmt.Errorf("invalid database schema: %q", schema)
	}
	if _, err := db.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema)); err != nil {
		return fmt.Errorf("failed to ensure schema %s exists: %w", schema, err)
	}
	return nil
}
