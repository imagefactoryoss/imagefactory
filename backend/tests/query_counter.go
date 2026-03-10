package tests

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

// QueryCounter tracks SQL query execution count
type QueryCounter struct {
	db    *sqlx.DB
	count int
	mu    sync.Mutex
}

// NewQueryCounter creates a new counter
func NewQueryCounter(db *sqlx.DB) *QueryCounter {
	return &QueryCounter{db: db, count: 0}
}

// Count returns current query count
func (qc *QueryCounter) Count() int {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	return qc.count
}

// Reset clears the counter
func (qc *QueryCounter) Reset() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.count = 0
}

// IncrementCount increments manually
func (qc *QueryCounter) IncrementCount() {
	qc.mu.Lock()
	qc.count++
	qc.mu.Unlock()
}

// incrementCount checks query type before counting
func (qc *QueryCounter) incrementCount(query string) {
	trimmed := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(trimmed, "SELECT") &&
		!strings.HasPrefix(trimmed, "INSERT") &&
		!strings.HasPrefix(trimmed, "UPDATE") &&
		!strings.HasPrefix(trimmed, "DELETE") {
		return
	}
	qc.IncrementCount()
}

// queryCountingDB wraps sqlx.DB for counting
type queryCountingDB struct {
	db      *sqlx.DB
	counter *QueryCounter
}

// WrappedDB returns wrapped database
func (qc *QueryCounter) WrappedDB() *queryCountingDB {
	return &queryCountingDB{db: qc.db, counter: qc}
}

// QueryContext counts SELECT/INSERT/UPDATE/DELETE
func (qcdb *queryCountingDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	qcdb.counter.incrementCount(query)
	return qcdb.db.QueryContext(ctx, query, args...)
}

// QueryRowContext counts single row queries
func (qcdb *queryCountingDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	qcdb.counter.incrementCount(query)
	return qcdb.db.QueryRowContext(ctx, query, args...)
}

// ExecContext counts executions
func (qcdb *queryCountingDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	qcdb.counter.incrementCount(query)
	return qcdb.db.ExecContext(ctx, query, args...)
}

// SelectContext counts SELECT operations
func (qcdb *queryCountingDB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	qcdb.counter.incrementCount(query)
	return qcdb.db.SelectContext(ctx, dest, query, args...)
}

// GetContext counts single row operations
func (qcdb *queryCountingDB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	qcdb.counter.incrementCount(query)
	return qcdb.db.GetContext(ctx, dest, query, args...)
}
