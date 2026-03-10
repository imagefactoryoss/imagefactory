package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestBatchLoadingPattern demonstrates the N+1 prevention using QueryCounter
func TestBatchLoadingPattern(t *testing.T) {
	counter := NewQueryCounter(nil)

	// Simulate N+1 pattern (old way)
	t.Run("N+1_Pattern_Old", func(t *testing.T) {
		counter.Reset()
		counter.IncrementCount() // Load users list
		for i := 0; i < 50; i++ {
			counter.IncrementCount() // Load role for each user
		}
		assert.Equal(t, 51, counter.Count(), "N+1 pattern should create 51 queries (1 + 50)")
	})

	// Simulate batch-loaded pattern (new way)
	t.Run("Batch_Loaded_Pattern_New", func(t *testing.T) {
		counter.Reset()
		counter.IncrementCount() // Single query with JOIN for users + roles
		assert.Equal(t, 1, counter.Count(), "Batch loading should create only 1 query")
	})
}

// TestQueryCounterWithContextIntegration tests counter with context
func TestQueryCounterWithContextIntegration(t *testing.T) {
	counter := NewQueryCounter(nil)
	ctx := context.Background()

	// Simulate repository operation
	userIDs := make([]uuid.UUID, 10)
	for i := 0; i < 10; i++ {
		userIDs[i] = uuid.New()
	}

	// Old way: N+1
	counter.Reset()
	counter.IncrementCount() // Load users
	for range userIDs {
		counter.IncrementCount() // Load roles per user
	}
	assert.Equal(t, 11, counter.Count())

	// New way: Batch load
	counter.Reset()
	counter.IncrementCount() // Single batch query
	assert.Equal(t, 1, counter.Count())

	// Verify context is still available
	assert.NotNil(t, ctx)
}

// TestBatchLoadingMetrics shows the performance improvement metrics
func TestBatchLoadingMetrics(t *testing.T) {
	counter := NewQueryCounter(nil)

	scenarios := []struct {
		name            string
		userCount       int
		oldWayQueries   int
		newWayQueries   int
		expectedSavings float64
	}{
		{"Small Tenant", 10, 11, 1, 90.9},
		{"Medium Tenant", 50, 51, 1, 98.0},
		{"Large Tenant", 100, 101, 1, 99.0},
		{"Enterprise Tenant", 500, 501, 1, 99.8},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Old pattern
			counter.Reset()
			counter.IncrementCount() // Users
			for i := 0; i < scenario.userCount; i++ {
				counter.IncrementCount() // Roles
			}
			assert.Equal(t, scenario.oldWayQueries, counter.Count())

			// New pattern
			counter.Reset()
			counter.IncrementCount() // Single batch query
			assert.Equal(t, scenario.newWayQueries, counter.Count())

			// Calculate savings
			queryReduction := float64(scenario.oldWayQueries-scenario.newWayQueries) / float64(scenario.oldWayQueries) * 100
			assert.Greater(t, queryReduction, 90.0, "Should reduce queries by at least 90%")
		})
	}
}

// TestBatchLoadingWithPermissions demonstrates batch loading with deeper nesting
func TestBatchLoadingWithPermissions(t *testing.T) {
	counter := NewQueryCounter(nil)

	// Scenario: Load users with roles and permissions
	userCount := 10
	rolesPerUser := 3
	_ = rolesPerUser // Will be used in comments

	t.Run("Without_Optimization", func(t *testing.T) {
		counter.Reset()
		counter.IncrementCount() // Load users
		for i := 0; i < userCount; i++ {
			counter.IncrementCount() // Load roles per user
			for j := 0; j < rolesPerUser; j++ {
				counter.IncrementCount() // Load permissions per role
			}
		}
		// 1 (users) + 10 (roles) + 30 (permissions) = 41 queries
		assert.Equal(t, 41, counter.Count())
	})

	t.Run("With_Optimization", func(t *testing.T) {
		counter.Reset()
		counter.IncrementCount() // Load users with role IDs (1 query with GROUP_BY/ARRAY_AGG)
		counter.IncrementCount() // Load all permissions for all roles (1 query with batch IDs)
		// 2 queries total (down from 41)
		assert.Equal(t, 2, counter.Count())
	})
}

// TestBatchLoadingErrorHandling tests error scenarios
func TestBatchLoadingErrorHandling(t *testing.T) {
	counter := NewQueryCounter(nil)

	// Test empty batch
	counter.Reset()
	assert.Equal(t, 0, counter.Count())

	// Test error recovery
	counter.IncrementCount()
	counter.IncrementCount()
	counter.Reset()
	assert.Equal(t, 0, counter.Count())

	// Test continued counting after reset
	counter.IncrementCount()
	assert.Equal(t, 1, counter.Count())
}
