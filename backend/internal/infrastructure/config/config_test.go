package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate_Success(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			Environment: "test",
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Name:            "test_db",
			User:            "postgres",
			Password:        "password",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 0,
		},
		Auth: AuthConfig{
			JWTSecret: "test-secret-key",
		},
		Build: BuildConfig{
			MaxConcurrentJobs: 10,
			WorkerPoolSize:    5,
		},
		Redis: RedisConfig{
			Port: 6379,
		},
		Dispatcher: DispatcherConfig{
			MaxDispatchPerTick: 1,
		},
		Workflow: WorkflowConfig{
			MaxStepsPerTick: 1,
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_MissingJWTSecret(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			Environment: "test",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Name:         "test_db",
			User:         "postgres",
			Password:     "password",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Auth: AuthConfig{
			JWTSecret: "", // Missing
		},
		Build: BuildConfig{
			MaxConcurrentJobs: 10,
			WorkerPoolSize:    5,
		},
		Redis: RedisConfig{
			Port: 6379,
		},
		Dispatcher: DispatcherConfig{
			MaxDispatchPerTick: 1,
		},
		Workflow: WorkflowConfig{
			MaxStepsPerTick: 1,
		},
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT secret is required")
}

func TestConfig_Validate_MissingDatabaseHost(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			Environment: "test",
		},
		Database: DatabaseConfig{
			Host:         "", // Missing
			Port:         5432,
			Name:         "test_db",
			User:         "postgres",
			Password:     "password",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Auth: AuthConfig{
			JWTSecret: "test-secret",
		},
		Build: BuildConfig{
			MaxConcurrentJobs: 10,
			WorkerPoolSize:    5,
		},
		Redis: RedisConfig{
			Port: 6379,
		},
		Dispatcher: DispatcherConfig{
			MaxDispatchPerTick: 1,
		},
		Workflow: WorkflowConfig{
			MaxStepsPerTick: 1,
		},
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database host is required")
}

func TestConfig_Validate_InvalidServerPort(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:        0, // Invalid
			Host:        "0.0.0.0",
			Environment: "test",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Name:         "test_db",
			User:         "postgres",
			Password:     "password",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Auth: AuthConfig{
			JWTSecret: "test-secret",
		},
		Build: BuildConfig{
			MaxConcurrentJobs: 10,
			WorkerPoolSize:    5,
		},
		Redis: RedisConfig{
			Port: 6379,
		},
		Dispatcher: DispatcherConfig{
			MaxDispatchPerTick: 1,
		},
		Workflow: WorkflowConfig{
			MaxStepsPerTick: 1,
		},
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server port must be between 1 and 65535")
}

func TestConfig_Validate_InvalidMaxOpenConns(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			Environment: "test",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Name:         "test_db",
			User:         "postgres",
			Password:     "password",
			MaxOpenConns: 0, // Invalid - must be positive
			MaxIdleConns: 5,
		},
		Auth: AuthConfig{
			JWTSecret: "test-secret",
		},
		Build: BuildConfig{
			MaxConcurrentJobs: 10,
			WorkerPoolSize:    5,
		},
		Redis: RedisConfig{
			Port: 6379,
		},
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_open_conns must be positive")
}

func TestConfig_Validate_InvalidBuildConcurrency(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			Environment: "test",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Name:         "test_db",
			User:         "postgres",
			Password:     "password",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Auth: AuthConfig{
			JWTSecret: "test-secret",
		},
		Build: BuildConfig{
			MaxConcurrentJobs: 0, // Invalid
			WorkerPoolSize:    5,
		},
		Redis: RedisConfig{
			Port: 6379,
		},
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_concurrent_jobs must be positive")
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Save current env vars
	oldJWTSecret := os.Getenv("IF_AUTH_JWT_SECRET")
	oldPort := os.Getenv("IF_SERVER_PORT")
	oldDBHost := os.Getenv("IF_DATABASE_HOST")
	oldDBName := os.Getenv("IF_DATABASE_NAME")
	oldDBUser := os.Getenv("IF_DATABASE_USER")

	defer func() {
		// Restore old env vars
		if oldJWTSecret != "" {
			os.Setenv("IF_AUTH_JWT_SECRET", oldJWTSecret)
		}
		if oldPort != "" {
			os.Setenv("IF_SERVER_PORT", oldPort)
		}
		if oldDBHost != "" {
			os.Setenv("IF_DATABASE_HOST", oldDBHost)
		}
		if oldDBName != "" {
			os.Setenv("IF_DATABASE_NAME", oldDBName)
		}
		if oldDBUser != "" {
			os.Setenv("IF_DATABASE_USER", oldDBUser)
		}
	}()

	// Set test env vars
	os.Setenv("IF_AUTH_JWT_SECRET", "test-jwt-secret")
	os.Setenv("IF_SERVER_PORT", "9090")
	os.Setenv("IF_DATABASE_HOST", "test-host")
	os.Setenv("IF_DATABASE_NAME", "test_database")
	os.Setenv("IF_DATABASE_USER", "test_user")

	config, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "test-jwt-secret", config.Auth.JWTSecret)
	assert.Equal(t, 9090, config.Server.Port)
	assert.Equal(t, "test-host", config.Database.Host)
	assert.Equal(t, "test_database", config.Database.Name)
	assert.Equal(t, "test_user", config.Database.User)
}

func TestLoad_WithMissingRequiredEnv(t *testing.T) {
	// Save current env vars
	oldJWTSecret := os.Getenv("IF_AUTH_JWT_SECRET")
	oldDBHost := os.Getenv("IF_DATABASE_HOST")
	oldDBName := os.Getenv("IF_DATABASE_NAME")
	oldDBUser := os.Getenv("IF_DATABASE_USER")

	defer func() {
		// Restore old env vars
		if oldJWTSecret != "" {
			os.Setenv("IF_AUTH_JWT_SECRET", oldJWTSecret)
		}
		if oldDBHost != "" {
			os.Setenv("IF_DATABASE_HOST", oldDBHost)
		}
		if oldDBName != "" {
			os.Setenv("IF_DATABASE_NAME", oldDBName)
		}
		if oldDBUser != "" {
			os.Setenv("IF_DATABASE_USER", oldDBUser)
		}
	}()

	// Unset critical env var
	os.Unsetenv("IF_AUTH_JWT_SECRET")
	os.Setenv("IF_DATABASE_HOST", "localhost")
	os.Setenv("IF_DATABASE_NAME", "test_db")
	os.Setenv("IF_DATABASE_USER", "postgres")

	config, err := Load()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "JWT secret is required")
}

func TestLoad_SetDefaults(t *testing.T) {
	// Save and clear relevant env vars - need to unset all test vars
	oldJWTSecret := os.Getenv("IF_AUTH_JWT_SECRET")
	oldPort := os.Getenv("IF_SERVER_PORT")
	oldDBHost := os.Getenv("IF_DATABASE_HOST")
	oldDBName := os.Getenv("IF_DATABASE_NAME")
	oldDBUser := os.Getenv("IF_DATABASE_USER")

	defer func() {
		// Restore old env vars
		if oldJWTSecret != "" {
			os.Setenv("IF_AUTH_JWT_SECRET", oldJWTSecret)
		} else {
			os.Unsetenv("IF_AUTH_JWT_SECRET")
		}
		if oldPort != "" {
			os.Setenv("IF_SERVER_PORT", oldPort)
		} else {
			os.Unsetenv("IF_SERVER_PORT")
		}
		if oldDBHost != "" {
			os.Setenv("IF_DATABASE_HOST", oldDBHost)
		} else {
			os.Unsetenv("IF_DATABASE_HOST")
		}
		if oldDBName != "" {
			os.Setenv("IF_DATABASE_NAME", oldDBName)
		} else {
			os.Unsetenv("IF_DATABASE_NAME")
		}
		if oldDBUser != "" {
			os.Setenv("IF_DATABASE_USER", oldDBUser)
		} else {
			os.Unsetenv("IF_DATABASE_USER")
		}
	}()

	// Set only JWT secret, leave others unset to test defaults
	os.Setenv("IF_AUTH_JWT_SECRET", "test-secret")
	os.Unsetenv("IF_SERVER_PORT")
	os.Unsetenv("IF_DATABASE_HOST")
	os.Unsetenv("IF_DATABASE_NAME")
	os.Unsetenv("IF_DATABASE_USER")

	config, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify defaults are set
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "image_factory", config.Database.Name)
	assert.Equal(t, 25, config.Database.MaxOpenConns)
	assert.Equal(t, 5, config.Database.MaxIdleConns)
	assert.Equal(t, "info", config.Logger.Level)
	assert.Equal(t, "json", config.Logger.Format)
	assert.Equal(t, 6379, config.Redis.Port)
	assert.Equal(t, "localhost:5000", config.Build.RegistryURL)
}

func TestLoad_DatabaseSchemaDefaultsToPublic(t *testing.T) {
	oldJWTSecret := os.Getenv("IF_AUTH_JWT_SECRET")
	oldDBSchema := os.Getenv("IF_DATABASE_SCHEMA")
	defer func() {
		if oldJWTSecret != "" {
			os.Setenv("IF_AUTH_JWT_SECRET", oldJWTSecret)
		} else {
			os.Unsetenv("IF_AUTH_JWT_SECRET")
		}
		if oldDBSchema != "" {
			os.Setenv("IF_DATABASE_SCHEMA", oldDBSchema)
		} else {
			os.Unsetenv("IF_DATABASE_SCHEMA")
		}
	}()

	os.Setenv("IF_AUTH_JWT_SECRET", "test-secret")
	os.Unsetenv("IF_DATABASE_SCHEMA")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "public", cfg.Database.Schema)
}

func TestLoad_AppendsSearchPathToDatabaseURL(t *testing.T) {
	oldJWTSecret := os.Getenv("IF_AUTH_JWT_SECRET")
	oldDBURL := os.Getenv("IF_DATABASE_URL")
	oldDBSchema := os.Getenv("IF_DATABASE_SCHEMA")
	defer func() {
		if oldJWTSecret != "" {
			os.Setenv("IF_AUTH_JWT_SECRET", oldJWTSecret)
		} else {
			os.Unsetenv("IF_AUTH_JWT_SECRET")
		}
		if oldDBURL != "" {
			os.Setenv("IF_DATABASE_URL", oldDBURL)
		} else {
			os.Unsetenv("IF_DATABASE_URL")
		}
		if oldDBSchema != "" {
			os.Setenv("IF_DATABASE_SCHEMA", oldDBSchema)
		} else {
			os.Unsetenv("IF_DATABASE_SCHEMA")
		}
	}()

	os.Setenv("IF_AUTH_JWT_SECRET", "test-secret")
	os.Setenv("IF_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/image_factory_dev?sslmode=disable")
	os.Setenv("IF_DATABASE_SCHEMA", "image_factory")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Contains(t, cfg.Database.URL, "search_path=image_factory")
}
