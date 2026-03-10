package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/subosito/gotenv"
)

// ExternalTenant represents a tenant from the external system
type ExternalTenant struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	Industry     string `json:"industry"`
	Country      string `json:"country"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// APIKeyMiddleware validates API keys for external tenant service
type APIKeyMiddleware struct {
	expectedAPIKey string
}

// NewAPIKeyMiddleware creates a new API key middleware
func NewAPIKeyMiddleware(expectedAPIKey string) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		expectedAPIKey: expectedAPIKey,
	}
}

// Middleware validates the API key
func (m *APIKeyMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for API key in X-API-Key header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Also check Authorization header (Bearer token)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					apiKey = parts[1]
				}
			}
		}

		if apiKey == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "missing_api_key", Message: "API key required"})
			return
		}

		if apiKey != m.expectedAPIKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid_api_key", Message: "Invalid API key"})
			return
		}

		// API key is valid, proceed
		next(w, r)
	}
}

var db *sql.DB

func init() {
	// Get database connection details from environment variables
	// Supports both IF_DATABASE_* (Image Factory config) and DB_* (fallback) prefixes
	host := os.Getenv("IF_DATABASE_HOST")
	if host == "" {
		host = os.Getenv("DB_HOST")
	}
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("IF_DATABASE_PORT")
	if port == "" {
		port = os.Getenv("DB_PORT")
	}
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("IF_DATABASE_USER")
	if user == "" {
		user = os.Getenv("DB_USER")
	}
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("IF_DATABASE_PASSWORD")
	if password == "" {
		password = os.Getenv("DB_PASSWORD")
	}
	if password == "" {
		password = "postgres"
	}

	dbname := os.Getenv("IF_DATABASE_NAME")
	if dbname == "" {
		dbname = os.Getenv("DB_NAME")
	}
	if dbname == "" {
		dbname = "image_factory_dev"
	}

	schema := os.Getenv("IF_DATABASE_SCHEMA")
	if schema == "" {
		schema = os.Getenv("DB_SCHEMA")
	}
	if schema == "" {
		schema = "public"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		host, port, user, password, dbname, schema)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	var currentSchema string
	if err := db.QueryRow(`SELECT current_schema()`).Scan(&currentSchema); err != nil {
		log.Printf("⚠️  Database schema readiness check failed: unable to resolve current schema: %v", err)
	} else {
		if currentSchema != schema {
			log.Printf("⚠️  Database schema mismatch detected: configured=%s current=%s", schema, currentSchema)
		}
		var schemaMigrations bool
		if err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = current_schema()
				  AND table_name = 'schema_migrations'
			)
		`).Scan(&schemaMigrations); err != nil {
			log.Printf("⚠️  Database schema readiness check failed: schema_migrations lookup failed: %v", err)
		}
		log.Printf("✅ Database schema readiness check completed (current_schema=%s, schema_migrations_present=%t)", currentSchema, schemaMigrations)
	}

	log.Println("✅ External Tenant Service connected to database")
}

// getTenants retrieves all external tenants, optionally filtered
func getTenants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Optional query parameters for filtering
	industryParam := r.URL.Query().Get("industry")
	countryParam := r.URL.Query().Get("country")

	query := "SELECT id, tenant_id, name, slug, description, contact_email, industry, country, created_at, updated_at FROM external_tenants WHERE 1=1"
	args := []interface{}{}

	if industryParam != "" {
		query += " AND industry = $" + strconv.Itoa(len(args)+1)
		args = append(args, industryParam)
	}

	if countryParam != "" {
		query += " AND country = $" + strconv.Itoa(len(args)+1)
		args = append(args, countryParam)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query_error", Message: "Failed to query tenants"})
		log.Printf("❌ Database query error: %v", err)
		return
	}
	defer rows.Close()

	tenants := []ExternalTenant{}
	for rows.Next() {
		var tenant ExternalTenant
		if err := rows.Scan(&tenant.ID, &tenant.TenantID, &tenant.Name, &tenant.Slug,
			&tenant.Description, &tenant.ContactEmail, &tenant.Industry, &tenant.Country,
			&tenant.CreatedAt, &tenant.UpdatedAt); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "scan_error", Message: "Failed to scan tenant data"})
			log.Printf("❌ Row scan error: %v", err)
			return
		}
		tenants = append(tenants, tenant)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    tenants,
		"count":   len(tenants),
	})
}

// getTenantByID retrieves a single external tenant by ID (either internal ID or external_id)
func getTenantByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	// Try to match by tenant_id first, then by internal ID
	query := "SELECT id, tenant_id, name, slug, description, contact_email, industry, country, created_at, updated_at FROM external_tenants WHERE tenant_id = $1 OR id = $1 LIMIT 1"

	var tenant ExternalTenant
	err := db.QueryRow(query, id).Scan(&tenant.ID, &tenant.TenantID, &tenant.Name, &tenant.Slug,
		&tenant.Description, &tenant.ContactEmail, &tenant.Industry, &tenant.Country,
		&tenant.CreatedAt, &tenant.UpdatedAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "not_found", Message: "Tenant not found"})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query_error", Message: "Failed to query tenant"})
		log.Printf("❌ Database query error: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    tenant,
	})
}

// getTenantBySlug retrieves a tenant by slug (URL-friendly identifier)
func getTenantBySlug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	slug := chi.URLParam(r, "slug")

	query := "SELECT id, tenant_id, name, slug, description, contact_email, industry, country, created_at, updated_at FROM external_tenants WHERE slug = $1"

	var tenant ExternalTenant
	err := db.QueryRow(query, slug).Scan(&tenant.ID, &tenant.TenantID, &tenant.Name, &tenant.Slug,
		&tenant.Description, &tenant.ContactEmail, &tenant.Industry, &tenant.Country,
		&tenant.CreatedAt, &tenant.UpdatedAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "not_found", Message: "Tenant not found"})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query_error", Message: "Failed to query tenant"})
		log.Printf("❌ Database query error: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    tenant,
	})
}

// createTenant creates a new external tenant
func createTenant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		TenantID     string `json:"tenant_id"`
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Description  string `json:"description"`
		ContactEmail string `json:"contact_email"`
		Industry     string `json:"industry"`
		Country      string `json:"country"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid_request", Message: "Invalid request body"})
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "validation_error", Message: "tenant_id, name, and slug are required"})
		return
	}

	// Validate tenant_id is 8 digits
	if len(req.TenantID) != 8 || !isNumeric(req.TenantID) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "validation_error", Message: "tenant_id must be 8 digits"})
		return
	}

	query := `INSERT INTO external_tenants (tenant_id, name, slug, description, contact_email, industry, country)
	           VALUES ($1, $2, $3, $4, $5, $6, $7)
	           RETURNING id, tenant_id, name, slug, description, contact_email, industry, country, created_at, updated_at`

	var tenant ExternalTenant
	err := db.QueryRow(query, req.TenantID, req.Name, req.Slug, req.Description, req.ContactEmail, req.Industry, req.Country).
		Scan(&tenant.ID, &tenant.TenantID, &tenant.Name, &tenant.Slug, &tenant.Description, &tenant.ContactEmail, &tenant.Industry, &tenant.Country, &tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "duplicate_error", Message: "tenant_id or slug already exists"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "database_error", Message: "Failed to create tenant"})
		log.Printf("❌ Database insert error: %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    tenant,
	})
}

// health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "external-tenant-service",
	})
}

// Helper function to check if string is numeric
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func main() {
	// Load environment variables from .env.development file
	envFile := "../.env.development"
	if _, err := os.Stat(envFile); err == nil {
		if err := gotenv.Load(envFile); err != nil {
			log.Printf("⚠️  Warning: Failed to load environment file %s: %v", envFile, err)
		} else {
			log.Printf("✅ Loaded environment from: %s", envFile)
		}
	} else {
		log.Printf("⚠️  Warning: Environment file not found: %s", envFile)
	}

	// Get API key from environment
	apiKey := os.Getenv("EXTERNAL_TENANT_API_KEY")
	if apiKey == "" {
		// Default for development
		apiKey = "dev-tenant-api-key-12345"
		log.Printf("⚠️  Using default API key for development")
	}
	log.Printf("🔑 External Tenant Service API Key: %s", apiKey)

	// Initialize API key middleware
	apiKeyMiddleware := NewAPIKeyMiddleware(apiKey)

	router := chi.NewRouter()

	// Health check (no auth required)
	router.Get("/health", healthCheck)

	// Tenant endpoints (require API key authentication)
	router.Get("/api/tenants", apiKeyMiddleware.Middleware(getTenants))
	router.Get("/api/tenants/{id}", apiKeyMiddleware.Middleware(getTenantByID))
	router.Get("/api/tenants/by-slug/{slug}", apiKeyMiddleware.Middleware(getTenantBySlug))
	router.Post("/api/tenants", apiKeyMiddleware.Middleware(createTenant))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("🔐 External Tenant Service with API key authentication listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
