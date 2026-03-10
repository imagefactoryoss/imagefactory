package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/subosito/gotenv"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// ExternalServiceSeed represents a service to be seeded
type ExternalServiceSeed struct {
	Name        string
	Description string
	URL         string
	APIKey      string
	Enabled     bool
}

func main() {
	// Load .env file
	envFile := flag.String("env", ".env.development", "Path to .env file")
	action := flag.String("action", "seed", "Action to perform: seed, validate, stats, reset")
	flag.Parse()

	// Load environment variables
	if err := gotenv.Load(*envFile); err != nil {
		log.Printf("Warning: Could not load %s file: %v", *envFile, err)
	}

	// Get database credentials from environment variables with defaults
	host := getEnv("IF_DATABASE_HOST", "localhost")
	port := getEnv("IF_DATABASE_PORT", "5432")
	user := getEnv("IF_DATABASE_USER", "postgres")
	password := getEnv("IF_DATABASE_PASSWORD", "postgres")
	dbName := getEnv("IF_DATABASE_NAME", "image_factory_dev")
	sslMode := getEnv("IF_DATABASE_SSL_MODE", "disable")
	dbSchema := getEnv("IF_DATABASE_SCHEMA", "public")

	// Build database connection string
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s", user, password, host, port, dbName, sslMode, dbSchema)

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	switch *action {
	case "seed":
		if err := seedExternalServices(db); err != nil {
			log.Fatalf("Failed to seed external services: %v", err)
		}
		log.Println("External services seeded successfully")
	case "validate":
		if err := validateExternalServices(db); err != nil {
			log.Fatalf("Failed to validate external services: %v", err)
		}
		log.Println("External services validation completed")
	case "stats":
		if err := showExternalServicesStats(db); err != nil {
			log.Fatalf("Failed to show external services stats: %v", err)
		}
	case "reset":
		if err := resetExternalServices(db); err != nil {
			log.Fatalf("Failed to reset external services: %v", err)
		}
		log.Println("External services reset successfully")
	default:
		log.Fatalf("Unknown action: %s. Use seed, validate, stats, or reset", *action)
	}
}

func seedExternalServices(db *sql.DB) error {
	// Note: Essential external services (like tenant-service) are now seeded by essential-config-seeder
	// This seeder is for additional external services beyond the essential ones
	services := []ExternalServiceSeed{
		// Add additional services here as needed
	}

	ctx := context.Background()

	for _, service := range services {
		if err := seedExternalService(ctx, db, service); err != nil {
			return fmt.Errorf("failed to seed service %s: %w", service.Name, err)
		}
	}

	log.Println("No additional external services to seed (essential services are handled by essential-config-seeder)")
	return nil
}

func seedExternalService(ctx context.Context, db *sql.DB, service ExternalServiceSeed) error {
	// Check if service already exists
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM system_configs
			WHERE config_type = $1 AND config_key = $2
		)
	`, systemconfig.ConfigTypeExternalServices, fmt.Sprintf("external_service_%s", service.Name)).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if service exists: %w", err)
	}

	if exists {
		log.Printf("Service %s already exists, skipping", service.Name)
		return nil
	}

	// Get a valid user ID for created_by/updated_by
	var userID uuid.UUID
	err = db.QueryRowContext(ctx, `
		SELECT id FROM users LIMIT 1
	`).Scan(&userID)

	if err != nil {
		// If no users exist, use a nil UUID (this might not work, but let's try)
		log.Printf("Warning: No users found in database, using nil UUID for audit fields")
		userID = uuid.Nil
	}

	// Create service config
	config := &systemconfig.ExternalServiceConfig{
		Name:        service.Name,
		Description: service.Description,
		URL:         service.URL,
		APIKey:      service.APIKey,
		Enabled:     service.Enabled,
	}

	configValue, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Insert into database
	_, err = db.ExecContext(ctx, `
		INSERT INTO system_configs (
			id, tenant_id, config_type, config_key, config_value,
			status, description, is_default, created_by, updated_by,
			created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`, uuid.New(), nil, systemconfig.ConfigTypeExternalServices,
		fmt.Sprintf("external_service_%s", service.Name), configValue,
		"active", fmt.Sprintf("Configuration for %s", service.Name), false,
		userID, userID, time.Now().UTC(), time.Now().UTC(), 1)

	if err != nil {
		return fmt.Errorf("failed to insert service config: %w", err)
	}

	log.Printf("Seeded external service: %s", service.Name)
	return nil
}

func validateExternalServices(db *sql.DB) error {
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT config_key, config_value FROM system_configs
		WHERE config_type = $1 AND status = 'active'
	`, systemconfig.ConfigTypeExternalServices)

	if err != nil {
		return fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var validCount, invalidCount int

	for rows.Next() {
		var configKey string
		var configValue []byte

		if err := rows.Scan(&configKey, &configValue); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		var service systemconfig.ExternalServiceConfig
		if err := json.Unmarshal(configValue, &service); err != nil {
			log.Printf("Invalid JSON for service %s: %v", configKey, err)
			invalidCount++
			continue
		}

		// Validate required fields
		if service.Name == "" {
			log.Printf("Service %s missing name", configKey)
			invalidCount++
			continue
		}

		if service.URL == "" {
			log.Printf("Service %s missing URL", configKey)
			invalidCount++
			continue
		}

		if service.APIKey == "" {
			log.Printf("Service %s missing API key", configKey)
			invalidCount++
			continue
		}

		validCount++
		log.Printf("Service %s is valid", service.Name)
	}

	log.Printf("Validation complete: %d valid, %d invalid", validCount, invalidCount)
	return nil
}

func showExternalServicesStats(db *sql.DB) error {
	ctx := context.Background()

	var totalCount int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM system_configs
		WHERE config_type = $1
	`, systemconfig.ConfigTypeExternalServices).Scan(&totalCount)

	if err != nil {
		return fmt.Errorf("failed to count services: %w", err)
	}

	var activeCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM system_configs
		WHERE config_type = $1 AND status = 'active'
	`, systemconfig.ConfigTypeExternalServices).Scan(&activeCount)

	if err != nil {
		return fmt.Errorf("failed to count active services: %w", err)
	}

	log.Printf("External Services Stats:")
	log.Printf("Total services: %d", totalCount)
	log.Printf("Active services: %d", activeCount)

	return nil
}

func resetExternalServices(db *sql.DB) error {
	ctx := context.Background()

	result, err := db.ExecContext(ctx, `
		DELETE FROM system_configs
		WHERE config_type = $1
	`, systemconfig.ConfigTypeExternalServices)

	if err != nil {
		return fmt.Errorf("failed to delete services: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("Deleted %d external service configurations", rowsAffected)
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
