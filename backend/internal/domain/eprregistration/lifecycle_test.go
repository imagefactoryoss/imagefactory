package eprregistration

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewRequest_DefaultLifecycleStatusActive(t *testing.T) {
	req, err := NewRequest(uuid.New(), uuid.New(), "EPR-001", "Product", "Tech", "reason")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if req.LifecycleStatus != LifecycleStatusActive {
		t.Fatalf("expected lifecycle_status=%q, got %q", LifecycleStatusActive, req.LifecycleStatus)
	}
}

func TestApprove_SetsApprovedAndLastReviewedTimestamps(t *testing.T) {
	req, err := NewRequest(uuid.New(), uuid.New(), "EPR-001", "Product", "Tech", "reason")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if err := req.Approve(uuid.New(), "ok"); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if req.ApprovedAt == nil {
		t.Fatal("expected approved_at to be set")
	}
	if req.LastReviewedAt == nil {
		t.Fatal("expected last_reviewed_at to be set")
	}
	if req.LifecycleStatus != LifecycleStatusActive {
		t.Fatalf("expected lifecycle_status=%q, got %q", LifecycleStatusActive, req.LifecycleStatus)
	}
}

func TestParseLifecycleStatus(t *testing.T) {
	cases := []string{"active", "expiring", "expired", "suspended"}
	for _, c := range cases {
		got, err := ParseLifecycleStatus(c)
		if err != nil {
			t.Fatalf("ParseLifecycleStatus(%q) error = %v", c, err)
		}
		if string(got) != c {
			t.Fatalf("ParseLifecycleStatus(%q) = %q", c, got)
		}
	}

	if _, err := ParseLifecycleStatus("unknown"); err == nil {
		t.Fatal("expected invalid lifecycle status to return error")
	}
}

func TestLifecycleActions_RequireApprovedRequest(t *testing.T) {
	req, err := NewRequest(uuid.New(), uuid.New(), "EPR-010", "Product", "Tech", "reason")
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if err := req.Suspend(uuid.New(), "hold"); err == nil {
		t.Fatal("expected suspend to fail for pending request")
	}
	if err := req.Approve(uuid.New(), "ok"); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if err := req.Suspend(uuid.New(), "hold"); err != nil {
		t.Fatalf("Suspend() error = %v", err)
	}
	if req.LifecycleStatus != LifecycleStatusSuspended {
		t.Fatalf("expected suspended lifecycle, got %q", req.LifecycleStatus)
	}
	if err := req.Reactivate(uuid.New(), "clear"); err != nil {
		t.Fatalf("Reactivate() error = %v", err)
	}
	if req.LifecycleStatus != LifecycleStatusActive {
		t.Fatalf("expected active lifecycle, got %q", req.LifecycleStatus)
	}
}
