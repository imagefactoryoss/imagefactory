package queue

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestNewEmailQueueBackend(t *testing.T) {
	if _, err := NewEmailQueueBackend(QueueConfig{Backend: QueueBackendPostgres}); err == nil {
		t.Fatal("expected postgres backend error without DB")
	}

	if _, err := NewEmailQueueBackend(QueueConfig{Backend: QueueBackendRedis}); err == nil {
		t.Fatal("expected redis not implemented error")
	}

	if _, err := NewEmailQueueBackend(QueueConfig{Backend: QueueBackendNATS}); err == nil {
		t.Fatal("expected nats not implemented error")
	}

	if _, err := NewEmailQueueBackend(QueueConfig{Backend: QueueBackendType("unknown")}); err == nil {
		t.Fatal("expected unknown backend error")
	}

	backend, err := NewEmailQueueBackend(QueueConfig{
		Backend: QueueBackendPostgres,
		DB:      &sqlx.DB{},
	})
	if err != nil {
		t.Fatalf("expected postgres backend success with DB, got %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestMustNewEmailQueueBackend(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for invalid queue config")
		}
	}()
	_ = MustNewEmailQueueBackend(QueueConfig{Backend: QueueBackendPostgres})
}
