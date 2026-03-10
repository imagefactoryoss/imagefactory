package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
)

func main() {
	var down = flag.Int("down", 0, "Number of migrations to rollback (0 = migrate up)")
	flag.Parse()

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Connecting to database: %s\n", cfg.Database.URL)

	// Connect to database
	db, err := sqlx.Connect("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Database connection successful")

	// Create migration driver
	schema := normalizeSchema(cfg.Database.Schema)
	if err := ensureSchemaExists(db, schema); err != nil {
		log.Fatalf("Failed to ensure schema: %v", err)
	}

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{SchemaName: schema})
	if err != nil {
		log.Fatalf("Failed to create migration driver: %v", err)
	}

	// Get migrations directory
	migrationsDir, err := filepath.Abs("migrations")
	if err != nil {
		log.Fatalf("Failed to get migrations directory: %v", err)
	}

	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		log.Fatalf("Migrations directory does not exist: %s", migrationsDir)
	}

	fmt.Printf("Using migrations from: %s\n", migrationsDir)

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsDir),
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if *down > 0 {
		// Rollback migrations
		fmt.Printf("Rolling back %d migration(s)...\n", *down)
		if err := m.Steps(-*down); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		fmt.Println("Rollback completed successfully")
	} else {
		// Run migrations up
		fmt.Println("Running migrations...")
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		if err == migrate.ErrNoChange {
			fmt.Println("No new migrations to apply")
		} else {
			fmt.Println("Migrations completed successfully")
		}
	}

	// Get current version
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		log.Printf("Warning: Could not get migration version: %v", err)
	} else if err == migrate.ErrNilVersion {
		fmt.Println("Database schema version: No migrations applied")
	} else {
		fmt.Printf("Database schema version: %d (dirty: %t)\n", version, dirty)
	}
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
	_, err := db.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema))
	return err
}
