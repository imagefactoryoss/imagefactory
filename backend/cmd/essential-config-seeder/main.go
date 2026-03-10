package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/subosito/gotenv"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// EssentialConfig represents a configuration to be seeded
type EssentialConfig struct {
	ConfigType  systemconfig.ConfigType
	ConfigKey   string
	Description string
	Data        interface{}
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
		if err := ensureSystemBootstrap(db); err != nil {
			log.Fatalf("Failed to ensure system bootstrap data: %v", err)
		}
		if err := seedEssentialConfigs(db); err != nil {
			log.Fatalf("Failed to seed essential configs: %v", err)
		}
		if err := enforceDefaultAuthProviderActivation(db); err != nil {
			log.Fatalf("Failed to enforce default auth provider activation: %v", err)
		}
		if err := materializeToolAvailabilityForAllTenants(db); err != nil {
			log.Fatalf("Failed to materialize tenant tool availability defaults: %v", err)
		}
		log.Println("Essential configs seeded successfully")
	case "validate":
		if err := validateEssentialConfigs(db); err != nil {
			log.Fatalf("Failed to validate essential configs: %v", err)
		}
		log.Println("Essential configs validation completed")
	case "stats":
		if err := showEssentialConfigsStats(db); err != nil {
			log.Fatalf("Failed to show essential configs stats: %v", err)
		}
	case "reset":
		if err := resetEssentialConfigs(db); err != nil {
			log.Fatalf("Failed to reset essential configs: %v", err)
		}
		log.Println("Essential configs reset successfully")
	default:
		log.Fatalf("Unknown action: %s. Use seed, validate, stats, or reset", *action)
	}
}

func ensureSystemBootstrap(db *sql.DB) error {
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO tenants (tenant_code, name, slug, description, status)
		VALUES ('sysadmin', 'System Administrators', 'system-admin', 'System administrator tenant', 'active')
		ON CONFLICT (tenant_code) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed sysadmin tenant: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO tenant_groups (tenant_id, name, slug, description, role_type, is_system_group, status)
		SELECT t.id, 'System Administrators', 'system-admin', 'System administration group', 'system_administrator', true, 'active'
		FROM tenants t
		WHERE t.tenant_code = 'sysadmin'
		ON CONFLICT (tenant_id, slug) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed sysadmin tenant group: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO tenant_groups (tenant_id, name, slug, description, role_type, is_system_group, status)
		SELECT t.id, 'System Administrator Viewers', 'system-admin-viewer', 'System administration read-only group', 'system_administrator_viewer', true, 'active'
		FROM tenants t
		WHERE t.tenant_code = 'sysadmin'
		ON CONFLICT (tenant_id, slug) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed sysadmin viewer tenant group: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO tenant_groups (tenant_id, name, slug, description, role_type, is_system_group, status)
		SELECT t.id, 'Security Reviewers', 'security-reviewers', 'Central security/governance quarantine approver group', 'security_reviewer', true, 'active'
		FROM tenants t
		WHERE t.tenant_code = 'sysadmin'
		ON CONFLICT (tenant_id, slug) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed security reviewers tenant group: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO group_members (group_id, user_id, is_group_admin, added_at)
		SELECT tg.id, u.id, true, CURRENT_TIMESTAMP
		FROM tenant_groups tg
		JOIN tenants t ON t.id = tg.tenant_id
		JOIN users u ON u.email = 'admin@imagefactory.local'
		WHERE t.tenant_code = 'sysadmin'
		  AND tg.role_type = 'system_administrator'
		ON CONFLICT (group_id, user_id) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed sysadmin group member: %w", err)
	}

	log.Println("System bootstrap data ensured (sysadmin tenant, admin/viewer/security-reviewer groups, admin membership)")
	return nil
}

func seedEssentialConfigs(db *sql.DB) error {
	ldapAllowedDomains := ldapAllowedDomainsFromEnv()
	tektonManifestURLs := tektonCoreManifestURLsFromEnv()
	sorRuntimeMode := defaultSORRuntimeErrorMode()

	configs := []EssentialConfig{
		{
			ConfigType:  systemconfig.ConfigTypeLDAP,
			ConfigKey:   "ldap_active_directory",
			Description: "Default LDAP configuration for user authentication",
			Data: systemconfig.LDAPConfig{
				ProviderName:    getEnvAny("Active Directory", "IF_AUTH_LDAP_PROVIDER_NAME"),
				ProviderType:    getEnvAny("active_directory", "IF_AUTH_LDAP_PROVIDER_TYPE"),
				Host:            getEnvAny("localhost", "IF_AUTH_LDAP_SERVER", "LDAP_HOST"),
				Port:            getEnvAsIntAny(389, "IF_AUTH_LDAP_PORT", "LDAP_PORT"),
				BaseDN:          getEnvAny("dc=example,dc=com", "IF_AUTH_LDAP_BASE_DN", "LDAP_BASE_DN"),
				UserSearchBase:  getEnvAny("", "IF_AUTH_LDAP_USER_SEARCH_BASE", "LDAP_USER_SEARCH_BASE"),
				GroupSearchBase: getEnvAny("", "IF_AUTH_LDAP_GROUP_SEARCH_BASE", "LDAP_GROUP_SEARCH_BASE"),
				BindDN:          getEnvAny("cn=admin,dc=example,dc=com", "IF_AUTH_LDAP_BIND_DN", "LDAP_BIND_DN"),
				BindPassword:    getEnvAny("", "IF_AUTH_LDAP_BIND_PASSWORD", "LDAP_BIND_PASSWORD"),
				UserFilter:      getEnvAny("(uid=%s)", "LDAP_USER_FILTER"),
				GroupFilter:     getEnvAny("(member=%s)", "LDAP_GROUP_FILTER"),
				StartTLS:        getEnvAsBoolAny(false, "IF_AUTH_LDAP_USE_TLS", "LDAP_START_TLS"),
				SSL:             getEnvAsBoolAny(false, "IF_AUTH_LDAP_USE_SSL", "LDAP_SSL"),
				AllowedDomains:  ldapAllowedDomains,
				Enabled:         getEnvAsBoolAny(true, "IF_AUTH_LDAP_ENABLED", "LDAP_ENABLED"),
			},
		},
		{
			ConfigType:  systemconfig.ConfigTypeSMTP,
			ConfigKey:   "smtp",
			Description: "Default SMTP configuration for outbound notifications",
			Data: systemconfig.SMTPConfig{
				Host:     getEnvOrDefault("IF_SMTP_HOST", "localhost"),
				Port:     getEnvAsInt("IF_SMTP_PORT", 1025),
				Username: getEnvOrDefault("IF_SMTP_USERNAME", ""),
				Password: getEnvOrDefault("IF_SMTP_PASSWORD", ""),
				From:     getEnvOrDefault("IF_SMTP_FROM_EMAIL", "noreply@image-factory.com"),
				StartTLS: getEnvAsBool("IF_SMTP_USE_TLS", false),
				SSL:      getEnvAsBool("IF_SMTP_SSL", false),
				Enabled:  getEnvAsBool("IF_SMTP_ENABLED", true),
			},
		},
		{
			ConfigType:  systemconfig.ConfigTypeExternalServices,
			ConfigKey:   "external_service_tenant_service",
			Description: "Configuration for external tenant management service",
			Data: systemconfig.ExternalServiceConfig{
				Name:        "tenant-service",
				Description: "External tenant management service",
				URL:         getEnvOrDefault("EXTERNAL_TENANT_SERVICE_URL", "http://localhost:8082"),
				APIKey:      getEnvOrDefault("EXTERNAL_TENANT_API_KEY", "dev-tenant-api-key-12345"),
				Headers:     map[string]string{},
				Enabled:     true,
			},
		},
		{
			ConfigType:  systemconfig.ConfigTypeExternalServices,
			ConfigKey:   "external_service_audit_service",
			Description: "Configuration for external audit logging service",
			Data: systemconfig.ExternalServiceConfig{
				Name:        "audit-service",
				Description: "External audit logging service",
				URL:         getEnvOrDefault("EXTERNAL_AUDIT_SERVICE_URL", "http://localhost:8083"),
				APIKey:      getEnvOrDefault("EXTERNAL_AUDIT_API_KEY", ""),
				Headers:     map[string]string{},
				Enabled:     false,
			},
		},
		{
			ConfigType:  systemconfig.ConfigTypeExternalServices,
			ConfigKey:   "external_service_sor_service",
			Description: "Configuration for external EPR registration validation service",
			Data: systemconfig.ExternalServiceConfig{
				Name:        "epr-service",
				Description: "External EPR registration validation service",
				URL:         getEnvOrDefault("IF_SOR_SERVICE_URL", "http://localhost:8084"),
				APIKey:      getEnvOrDefault("IF_SOR_API_KEY", ""),
				Headers:     map[string]string{},
				Enabled:     getEnvAsBool("IF_SOR_SERVICE_ENABLED", false),
			},
		},
		{
			ConfigType:  systemconfig.ConfigTypeToolSettings,
			ConfigKey:   "epr_registration",
			Description: "EPR registration prerequisite policy for quarantine admission",
			Data: systemconfig.SORRegistrationConfig{
				Enforce:          getEnvAsBool("IF_SOR_ENFORCE", true),
				RuntimeErrorMode: sorRuntimeMode,
			},
		},
		{
			ConfigType:  systemconfig.ConfigTypeMessaging,
			ConfigKey:   "messaging",
			Description: "Messaging configuration settings",
			Data: systemconfig.MessagingConfig{
				EnableNATS: true,
			},
		},
		{
			ConfigType:  systemconfig.ConfigTypeTekton,
			ConfigKey:   "tekton_core",
			Description: "Tekton core install/config defaults for provider preparation (air-gapped friendly)",
			Data: systemconfig.TektonCoreConfig{
				InstallSource:   getEnvOrDefault("IF_TEKTON_CORE_INSTALL_SOURCE", "manifest"),
				ManifestURLs:    tektonManifestURLs,
				HelmRepoURL:     getEnvOrDefault("IF_TEKTON_HELM_REPO_URL", ""),
				HelmChart:       getEnvOrDefault("IF_TEKTON_HELM_CHART", "tekton-pipeline"),
				HelmReleaseName: getEnvOrDefault("IF_TEKTON_HELM_RELEASE_NAME", "tekton-pipelines"),
				HelmNamespace:   getEnvOrDefault("IF_TEKTON_HELM_NAMESPACE", "tekton-pipelines"),
				AssetsDir:       getEnvOrDefault("IF_TEKTON_ASSETS_DIR", ""),
			},
		},
	}

	ctx := context.Background()

	for _, config := range configs {
		if err := seedEssentialConfig(ctx, db, config); err != nil {
			return fmt.Errorf("failed to seed config %s/%s: %w", config.ConfigType, config.ConfigKey, err)
		}
	}

	return nil
}

func defaultSORRuntimeErrorMode() string {
	override := strings.ToLower(strings.TrimSpace(getEnvOrDefault("IF_SOR_RUNTIME_ERROR_MODE", "")))
	if override != "" {
		return override
	}
	env := strings.ToLower(strings.TrimSpace(getEnvOrDefault("IF_SERVER_ENVIRONMENT", "development")))
	if env == "development" || env == "dev" || env == "local" {
		return "allow"
	}
	return "error"
}

func tektonCoreManifestURLsFromEnv() []string {
	raw := strings.TrimSpace(getEnvOrDefault("IF_TEKTON_CORE_MANIFEST_URLS", ""))
	if raw == "" {
		raw = strings.TrimSpace(getEnvOrDefault("IF_TEKTON_CORE_MANIFEST_URL", ""))
	}
	if raw == "" {
		// Default matches infrastructure service defaults; can be overridden later in UI/DB.
		return []string{"https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml"}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func materializeToolAvailabilityForAllTenants(db *sql.DB) error {
	ctx := context.Background()

	var actorID uuid.UUID
	err := db.QueryRowContext(ctx, `
		SELECT id FROM users
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&actorID)
	if err != nil {
		return fmt.Errorf("failed to resolve actor user for tenant config materialization: %w", err)
	}

	var globalConfigValue []byte
	var description string
	err = db.QueryRowContext(ctx, `
		SELECT config_value, COALESCE(description, '')
		FROM system_configs
		WHERE tenant_id IS NULL
		  AND config_type = 'tool_settings'
		  AND config_key = 'tool_availability'
		  AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&globalConfigValue, &description)
	if err != nil {
		if err == sql.ErrNoRows {
			defaultPayload, marshalErr := json.Marshal(defaultGlobalToolAvailabilityConfig())
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal default global tool_availability config: %w", marshalErr)
			}
			description = "Global tool availability configuration"
			if _, insertErr := db.ExecContext(ctx, `
				INSERT INTO system_configs (
					id, tenant_id, config_type, config_key, config_value,
					status, description, is_default, created_by, updated_by,
					created_at, updated_at, version
				) VALUES (
					$1, NULL, 'tool_settings', 'tool_availability', $2,
					'active', $3, true, $4, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1
				)
				ON CONFLICT (config_type, config_key) WHERE tenant_id IS NULL DO NOTHING
			`, uuid.New(), defaultPayload, description, actorID); insertErr != nil {
				return fmt.Errorf("failed to create global tool_availability config: %w", insertErr)
			}
			globalConfigValue = defaultPayload
			log.Println("Created missing global tool_availability config")
		} else {
			return fmt.Errorf("failed to load global tool_availability config: %w", err)
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT id FROM tenants`)
	if err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var createdCount int
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return fmt.Errorf("failed to scan tenant id: %w", err)
		}

		var exists bool
		err = db.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM system_configs
				WHERE tenant_id = $1
				  AND config_type = 'tool_settings'
				  AND config_key = 'tool_availability'
			)
		`, tenantID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check tenant tool_availability existence: %w", err)
		}
		if exists {
			continue
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO system_configs (
				id, tenant_id, config_type, config_key, config_value,
				status, description, is_default, created_by, updated_by,
				created_at, updated_at, version
			) VALUES (
				$1, $2, 'tool_settings', 'tool_availability', $3,
				'active', $4, false, $5, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1
			)
		`, uuid.New(), tenantID, globalConfigValue, description, actorID)
		if err != nil {
			return fmt.Errorf("failed to insert tenant tool_availability for tenant %s: %w", tenantID.String(), err)
		}
		createdCount++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed while iterating tenants: %w", err)
	}

	log.Printf("Materialized tenant tool availability defaults for %d tenant(s)", createdCount)
	return nil
}

func defaultGlobalToolAvailabilityConfig() map[string]interface{} {
	return map[string]interface{}{
		"build_methods": map[string]bool{
			"container": true,
			"packer":    true,
			"paketo":    true,
			"kaniko":    true,
			"buildx":    true,
			"nix":       true,
		},
		"sbom_tools": map[string]bool{
			"syft":  true,
			"grype": true,
			"trivy": true,
		},
		"scan_tools": map[string]bool{
			"trivy": true,
			"clair": false,
			"grype": true,
			"snyk":  false,
		},
		"registry_types": map[string]bool{
			"s3":          true,
			"harbor":      false,
			"quay":        false,
			"artifactory": false,
		},
		"secret_managers": map[string]bool{
			"vault":              false,
			"aws_secretsmanager": true,
			"azure_keyvault":     false,
			"gcp_secretmanager":  false,
		},
		"trivy_runtime": map[string]string{
			"cache_mode":         "shared",
			"db_repository":      "image-factory-registry:5000/security/trivy-db:2,mirror.gcr.io/aquasec/trivy-db:2",
			"java_db_repository": "image-factory-registry:5000/security/trivy-java-db:1,mirror.gcr.io/aquasec/trivy-java-db:1",
		},
	}
}

func enforceDefaultAuthProviderActivation(db *sql.DB) error {
	ctx := context.Background()

	// Ensure only Active Directory default LDAP provider is enabled globally.
	if _, err := db.ExecContext(ctx, `
		UPDATE system_configs
		SET config_value = jsonb_set(COALESCE(config_value::jsonb, '{}'::jsonb), '{enabled}', 'false'::jsonb, true),
		    updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id IS NULL
		  AND config_type = 'ldap'
		  AND config_key <> 'ldap_active_directory'
	`); err != nil {
		return fmt.Errorf("failed to disable non-default LDAP providers: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE system_configs
		SET config_value = jsonb_set(COALESCE(config_value::jsonb, '{}'::jsonb), '{enabled}', 'true'::jsonb, true),
		    updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id IS NULL
		  AND config_type = 'ldap'
		  AND config_key = 'ldap_active_directory'
	`); err != nil {
		return fmt.Errorf("failed to enable default LDAP provider: %w", err)
	}

	return nil
}

func seedEssentialConfig(ctx context.Context, db *sql.DB, config EssentialConfig) error {
	// Get a valid user ID for created_by/updated_by
	var userID uuid.UUID
	err := db.QueryRowContext(ctx, `
		SELECT id FROM users LIMIT 1
	`).Scan(&userID)

	if err != nil {
		// If no users exist, use a nil UUID (this might not work, but let's try)
		log.Printf("Warning: No users found in database, using nil UUID for audit fields")
		userID = uuid.Nil
	}

	// Marshal config data
	configValue, err := json.Marshal(config.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Check if global config already exists.
	var existingID uuid.UUID
	findErr := db.QueryRowContext(ctx, `
		SELECT id
		FROM system_configs
		WHERE tenant_id IS NULL
		  AND config_type = $1
		  AND config_key = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, config.ConfigType, config.ConfigKey).Scan(&existingID)

	overwriteExisting := getEnvAsBoolAny(false, "IF_SEED_ESSENTIAL_OVERWRITE", "SEED_ESSENTIAL_OVERWRITE")
	if findErr == nil {
		if !overwriteExisting {
			log.Printf("Skipping existing essential config (overwrite disabled): %s/%s", config.ConfigType, config.ConfigKey)
			return nil
		}
		_, err = db.ExecContext(ctx, `
			UPDATE system_configs
			SET config_value = $2,
			    description = $3,
			    status = 'active',
			    is_default = true,
			    updated_by = $4,
			    updated_at = $5,
			    version = version + 1
			WHERE id = $1
		`, existingID, configValue, config.Description, userID, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
		log.Printf("Updated essential config: %s/%s", config.ConfigType, config.ConfigKey)
		return nil
	}
	if findErr != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing config: %w", findErr)
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
	`, uuid.New(), nil, config.ConfigType, config.ConfigKey, configValue,
		"active", config.Description, true,
		userID, userID, time.Now().UTC(), time.Now().UTC(), 1)

	if err != nil {
		return fmt.Errorf("failed to insert config: %w", err)
	}

	log.Printf("Seeded essential config: %s/%s", config.ConfigType, config.ConfigKey)
	return nil
}

func getEnvAny(defaultValue string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return defaultValue
}

func getEnvAsIntAny(defaultValue int, keys ...string) int {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			var parsed int
			if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
				return parsed
			}
		}
	}
	return defaultValue
}

func getEnvAsBoolAny(defaultValue bool, keys ...string) bool {
	for _, key := range keys {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if value == "" {
			continue
		}
		switch value {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		}
	}
	return defaultValue
}

func getEnvCSVAny(keys ...string) []string {
	for _, key := range keys {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}
		parts := strings.Split(raw, ",")
		values := make([]string, 0, len(parts))
		for _, p := range parts {
			v := strings.TrimSpace(strings.ToLower(strings.TrimPrefix(p, "@")))
			if v == "" {
				continue
			}
			values = append(values, v)
		}
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func domainFromEmail(email string) string {
	if !strings.Contains(email, "@") {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(parts[1]))
}

func ldapAllowedDomainsFromEnv() []string {
	if values := getEnvCSVAny("IF_AUTH_LDAP_ALLOWED_DOMAINS", "LDAP_ALLOWED_DOMAINS"); len(values) > 0 {
		return values
	}
	if domain := domainFromEmail(getEnvOrDefault("IF_SMTP_FROM_EMAIL", "")); domain != "" {
		return []string{domain}
	}
	return []string{}
}

func validateEssentialConfigs(db *sql.DB) error {
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT config_type, config_key, config_value FROM system_configs
		WHERE is_default = true AND status = 'active'
	`)

	if err != nil {
		return fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()

	var validCount, invalidCount int

	for rows.Next() {
		var configType string
		var configKey string
		var configValue []byte

		if err := rows.Scan(&configType, &configKey, &configValue); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Basic validation - check if JSON is valid
		var jsonData interface{}
		if err := json.Unmarshal(configValue, &jsonData); err != nil {
			log.Printf("Invalid JSON for config %s/%s: %v", configType, configKey, err)
			invalidCount++
			continue
		}

		validCount++
		log.Printf("Config %s/%s is valid", configType, configKey)
	}

	log.Printf("Validation complete: %d valid, %d invalid", validCount, invalidCount)
	return nil
}

func showEssentialConfigsStats(db *sql.DB) error {
	ctx := context.Background()

	var totalCount int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM system_configs
		WHERE is_default = true
	`).Scan(&totalCount)

	if err != nil {
		return fmt.Errorf("failed to count configs: %w", err)
	}

	var activeCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM system_configs
		WHERE is_default = true AND status = 'active'
	`).Scan(&activeCount)

	if err != nil {
		return fmt.Errorf("failed to count active configs: %w", err)
	}

	log.Printf("Essential Configs Stats:")
	log.Printf("Total configs: %d", totalCount)
	log.Printf("Active configs: %d", activeCount)

	return nil
}

func resetEssentialConfigs(db *sql.DB) error {
	ctx := context.Background()

	result, err := db.ExecContext(ctx, `
		DELETE FROM system_configs
		WHERE is_default = true
	`)

	if err != nil {
		return fmt.Errorf("failed to delete configs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("Deleted %d essential config configurations", rowsAffected)
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

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var parsed int
		if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}
