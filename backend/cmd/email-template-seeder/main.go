package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/subosito/gotenv"
)

// EmailTemplate represents a single email template
type EmailTemplate struct {
	TemplateType string
	Category     string
	Subject      string
	HTMLTemplate string
	TextTemplate string
	Description  string
}

// Template categories
const (
	CategoryUserManagement    = "user_management"
	CategoryProjectManagement = "project_management"
	CategoryCompliance        = "compliance"
	CategoryNotifications     = "notifications"
	CategoryInvitations       = "invitations"
)

// Template types
const (
	TemplateTenantOnboarding  = "tenant_onboarding"
	TemplateUserInvitation    = "user_invitation"
	TemplateUserAddedToTenant = "user_added_to_tenant"
	TemplateUserSuspended     = "user_suspended"
	TemplateUserActivated     = "user_activated"
)

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

	// Execute action
	switch *action {
	case "seed":
		// Create database connection for seed action
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=%s",
			host, port, user, password, dbName, sslMode, dbSchema)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}

		fmt.Println("✅ Connected to database successfully")
		seedTemplates(db)

	case "validate":
		validateTemplates()

	case "stats":
		printStats()

	case "reset":
		// Create database connection for reset action
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=%s",
			host, port, user, password, dbName, sslMode, dbSchema)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}

		fmt.Println("✅ Connected to database successfully")
		resetTemplates(db)

	default:
		fmt.Printf("Unknown action: %s\n", *action)
		fmt.Println("Available actions: seed, validate, stats, reset")
		os.Exit(1)
	}
}

// getAllTemplates returns all email templates from all categories
func getAllTemplates() []EmailTemplate {
	var allTemplates []EmailTemplate

	allTemplates = append(allTemplates, getUserManagementTemplates()...)
	allTemplates = append(allTemplates, getInvitationTemplates()...)
	allTemplates = append(allTemplates, getNotificationTemplates()...)

	return allTemplates
}

// seedTemplates inserts or updates email templates in the database
func seedTemplates(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	allTemplates := getAllTemplates()

	if len(allTemplates) == 0 {
		fmt.Println("⚠️  No templates to seed")
		return
	}

	fmt.Printf("🌱 Seeding %d email templates...\n", len(allTemplates))

	for _, tmpl := range allTemplates {
		query := `
			INSERT INTO notification_templates (template_type, name, description, subject_template, body_template, html_template, is_default, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, true, true)
			ON CONFLICT (template_type) DO UPDATE SET
				subject_template = $4,
				html_template = $6,
				body_template = $5,
				updated_at = NOW()
		`

		result, err := db.ExecContext(ctx, query,
			tmpl.TemplateType,
			tmpl.TemplateType, // name defaults to template_type
			tmpl.Description,
			tmpl.Subject,
			tmpl.TextTemplate,
			tmpl.HTMLTemplate,
		)
		if err != nil {
			log.Fatalf("Failed to seed template %s: %v", tmpl.TemplateType, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Fatalf("Failed to get rows affected for %s: %v", tmpl.TemplateType, err)
		}

		if rowsAffected > 0 {
			fmt.Printf("  ✓ Seeded template: %s (Category: %s)\n", tmpl.TemplateType, tmpl.Category)
		}
	}

	fmt.Println("\n✅ Seeding completed successfully!")
}

// validateTemplates checks if all templates are valid
func validateTemplates() {
	allTemplates := getAllTemplates()

	fmt.Printf("🔍 Validating %d templates...\n\n", len(allTemplates))

	var errors []string
	for i, tmpl := range allTemplates {
		fmt.Printf("[%d] %s\n", i+1, tmpl.TemplateType)
		fmt.Printf("    Category: %s\n", tmpl.Category)
		if len(tmpl.Subject) > 50 {
			fmt.Printf("    Subject: %s...\n", tmpl.Subject[:50])
		} else {
			fmt.Printf("    Subject: %s\n", tmpl.Subject)
		}

		// Validate required fields
		if tmpl.TemplateType == "" {
			errors = append(errors, fmt.Sprintf("Template %d: missing template_type", i+1))
		}
		if tmpl.Category == "" {
			errors = append(errors, fmt.Sprintf("Template %d: missing category", i+1))
		}
		if tmpl.Subject == "" {
			errors = append(errors, fmt.Sprintf("Template %d: missing subject", i+1))
		}
		if tmpl.HTMLTemplate == "" {
			errors = append(errors, fmt.Sprintf("Template %d: missing html_template", i+1))
		}
		if tmpl.TextTemplate == "" {
			errors = append(errors, fmt.Sprintf("Template %d: missing text_template", i+1))
		}

		fmt.Println()
	}

	if len(errors) > 0 {
		fmt.Println("❌ Validation failed with errors:")
		for _, err := range errors {
			fmt.Printf("   - %s\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("✅ All templates are valid!")
}

// printStats prints statistics about available templates
func printStats() {
	allTemplates := getAllTemplates()

	fmt.Println("📊 Template Statistics")
	fmt.Println("======================")
	fmt.Printf("Total Templates: %d\n\n", len(allTemplates))

	// Group by category
	categories := make(map[string]int)
	for _, t := range allTemplates {
		categories[t.Category]++
	}

	fmt.Println("By Category:")
	for category, count := range categories {
		fmt.Printf("  - %s: %d\n", category, count)
	}

	fmt.Println("\nTemplate Types:")
	for i, t := range allTemplates {
		fmt.Printf("  %d. %s (%s)\n", i+1, t.TemplateType, t.Category)
	}
}

// resetTemplates clears all seeded templates from the database
func resetTemplates(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	allTemplates := getAllTemplates()

	if len(allTemplates) == 0 {
		fmt.Println("⚠️  No templates to reset")
		return
	}

	fmt.Printf("🔄 Resetting %d email templates...\n", len(allTemplates))

	// Reset templates to basic versions
	basicSubject := "Notification"
	basicHTML := "<html><body><p>This is a notification email.</p></body></html>"
	basicText := "This is a notification email."

	for _, tmpl := range allTemplates {
		query := `
			UPDATE notification_templates
			SET subject_template = $1,
				html_template = $2,
				body_template = $3,
				updated_at = NOW()
			WHERE template_type = $4
		`

		result, err := db.ExecContext(ctx, query,
			basicSubject,
			basicHTML,
			basicText,
			tmpl.TemplateType,
		)
		if err != nil {
			log.Fatalf("Failed to reset template %s: %v", tmpl.TemplateType, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Fatalf("Failed to get rows affected for %s: %v", tmpl.TemplateType, err)
		}

		if rowsAffected > 0 {
			fmt.Printf("  ✓ Reset template: %s\n", tmpl.TemplateType)
		}
	}

	fmt.Println("\n✅ Reset completed successfully!")
}

// getEnv retrieves an environment variable with a default fallback value
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
