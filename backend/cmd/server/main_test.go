package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/subosito/gotenv"
)

func TestEnvironmentFileLoading(t *testing.T) {
	// Create a temporary env file
	tempFile, err := os.CreateTemp("", "test-env-*.env")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test environment variables
	envContent := `TEST_VAR1=value1
TEST_VAR2=value2
IF_SERVER_PORT=9999
IF_DATABASE_NAME=test_db
`
	_, err = tempFile.WriteString(envContent)
	require.NoError(t, err)
	tempFile.Close()

	// Clear any existing environment variables
	os.Unsetenv("TEST_VAR1")
	os.Unsetenv("TEST_VAR2")
	os.Unsetenv("IF_SERVER_PORT")
	os.Unsetenv("IF_DATABASE_NAME")

	// Load the environment file
	err = gotenv.Load(tempFile.Name())
	require.NoError(t, err)

	// Verify environment variables are loaded
	assert.Equal(t, "value1", os.Getenv("TEST_VAR1"))
	assert.Equal(t, "value2", os.Getenv("TEST_VAR2"))
	assert.Equal(t, "9999", os.Getenv("IF_SERVER_PORT"))
	assert.Equal(t, "test_db", os.Getenv("IF_DATABASE_NAME"))

	// Clean up
	os.Unsetenv("TEST_VAR1")
	os.Unsetenv("TEST_VAR2")
	os.Unsetenv("IF_SERVER_PORT")
	os.Unsetenv("IF_DATABASE_NAME")
}

func TestEnvironmentFileNotFound(t *testing.T) {
	// Try to load a non-existent file
	err := gotenv.Load("/path/that/does/not/exist.env")
	assert.Error(t, err)
}
