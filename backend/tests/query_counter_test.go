package tests

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// TestNewQueryCounter tests counter initialization
func TestNewQueryCounter(t *testing.T) {
	db := &sqlx.DB{}
	counter := NewQueryCounter(db)

	assert.NotNil(t, counter)
	assert.Equal(t, 0, counter.Count())
	assert.NotNil(t, counter.db)
}

// TestQueryCounter_IncrementCount tests manual counter increment
func TestQueryCounter_IncrementCount(t *testing.T) {
	counter := NewQueryCounter(nil)
	assert.Equal(t, 0, counter.Count())

	counter.IncrementCount()
	assert.Equal(t, 1, counter.Count())

	counter.IncrementCount()
	counter.IncrementCount()
	assert.Equal(t, 3, counter.Count())
}

// TestQueryCounter_Reset tests counter reset functionality
func TestQueryCounter_Reset(t *testing.T) {
	counter := NewQueryCounter(nil)

	counter.IncrementCount()
	counter.IncrementCount()
	assert.Equal(t, 2, counter.Count())

	counter.Reset()
	assert.Equal(t, 0, counter.Count())
}

// TestQueryCounter_QueryTypeFiltering tests query type detection
func TestQueryCounter_QueryTypeFiltering(t *testing.T) {
	counter := NewQueryCounter(nil)

	// These should be counted
	counter.incrementCount("SELECT * FROM users")
	counter.incrementCount("INSERT INTO users VALUES (...)")
	counter.incrementCount("UPDATE users SET name = 'test'")
	counter.incrementCount("DELETE FROM users WHERE id = 1")

	assert.Equal(t, 4, counter.Count(), "All CRUD operations should be counted")

	counter.Reset()

	// These should NOT be counted (non-data queries)
	counter.incrementCount("SHOW TABLES")
	counter.incrementCount("EXPLAIN SELECT * FROM users")
	counter.incrementCount("BEGIN TRANSACTION")
	counter.incrementCount("COMMIT")

	assert.Equal(t, 0, counter.Count(), "Non-data queries should not be counted")
}

// TestQueryCounter_CaseSensitivity tests case-insensitive query detection
func TestQueryCounter_CaseSensitivity(t *testing.T) {
	counter := NewQueryCounter(nil)

	// Test lowercase
	counter.incrementCount("select * from users")
	assert.Equal(t, 1, counter.Count())

	counter.Reset()

	// Test mixed case
	counter.incrementCount("SeLeCt * FrOm users")
	assert.Equal(t, 1, counter.Count())

	counter.Reset()

	// Test with whitespace
	counter.incrementCount("  \n  SELECT * FROM users  \n  ")
	assert.Equal(t, 1, counter.Count())
}

// TestQueryCounter_WrappedDB tests wrapped DB creation
func TestQueryCounter_WrappedDB(t *testing.T) {
	counter := NewQueryCounter(&sqlx.DB{})
	wrappedDB := counter.WrappedDB()

	assert.NotNil(t, wrappedDB)
	assert.NotNil(t, wrappedDB.counter)
	assert.Equal(t, counter, wrappedDB.counter)
}

// TestQueryCounter_ThreadSafety tests concurrent access
func TestQueryCounter_ThreadSafety(t *testing.T) {
	counter := NewQueryCounter(nil)
	done := make(chan bool)

	// Launch 10 goroutines, each incrementing 5 times
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 5; j++ {
				counter.IncrementCount()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly 50 increments (10 goroutines * 5 increments)
	assert.Equal(t, 50, counter.Count())
}

// TestQueryCounter_UsagePattern demonstrates how to use the counter
func TestQueryCounter_UsagePattern(t *testing.T) {
	testDB := &sqlx.DB{}
	counter := NewQueryCounter(testDB)

	// Simulate single query scenario
	counter.IncrementCount() // FindByTenantID
	assert.Equal(t, 1, counter.Count())

	counter.Reset()

	// Simulate N+1 pattern
	counter.IncrementCount() // Load users (1 query)
	for i := 0; i < 5; i++ {
		counter.IncrementCount() // Load role for each user (5 queries)
	}
	assert.Equal(t, 6, counter.Count())

	counter.Reset()

	// Simulate optimized batch load
	counter.IncrementCount() // Load users with roles in 1 query
	assert.Equal(t, 1, counter.Count())
}
