package denialtelemetry

import (
	"testing"

	"github.com/google/uuid"
)

func TestMetrics_RecordDeniedAndSnapshot(t *testing.T) {
	metrics := NewMetrics()
	tenantID := uuid.New()

	metrics.RecordDenied(tenantID, "quarantine_request", "tenant_capability_not_entitled")
	metrics.RecordDenied(tenantID, "quarantine_request", "tenant_capability_not_entitled")
	metrics.RecordDenied(tenantID, "quarantine_request", "sor_registration_required")

	rows := metrics.Snapshot()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	var foundCapabilityDenied bool
	var foundSORDenied bool
	for _, row := range rows {
		if row.Reason == "tenant_capability_not_entitled" {
			foundCapabilityDenied = true
			if row.Count != 2 {
				t.Fatalf("expected count=2 for capability deny, got %d", row.Count)
			}
		}
		if row.Reason == "sor_registration_required" {
			foundSORDenied = true
			if row.Count != 1 {
				t.Fatalf("expected count=1 for SOR deny, got %d", row.Count)
			}
		}
	}
	if !foundCapabilityDenied || !foundSORDenied {
		t.Fatalf("expected both deny reasons in snapshot, got %+v", rows)
	}
}

func TestMetrics_RecordDeniedWithLabels(t *testing.T) {
	metrics := NewMetrics()
	tenantID := uuid.New()

	metrics.RecordDeniedWithLabels(tenantID, "quarantine_request", "sor_registration_required", map[string]string{
		"sor_runtime_mode": "allow",
		"sor_policy_scope": "tenant",
	})
	metrics.RecordDeniedWithLabels(tenantID, "quarantine_request", "sor_registration_required", map[string]string{
		"sor_runtime_mode": "allow",
		"sor_policy_scope": "tenant",
	})
	metrics.RecordDeniedWithLabels(tenantID, "quarantine_request", "sor_registration_required", map[string]string{
		"sor_runtime_mode": "error",
		"sor_policy_scope": "global",
	})

	rows := metrics.Snapshot()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	for _, row := range rows {
		if row.Labels["sor_runtime_mode"] == "allow" && row.Labels["sor_policy_scope"] == "tenant" {
			if row.Count != 2 {
				t.Fatalf("expected allow/tenant count=2, got %d", row.Count)
			}
		}
		if row.Labels["sor_runtime_mode"] == "error" && row.Labels["sor_policy_scope"] == "global" {
			if row.Count != 1 {
				t.Fatalf("expected error/global count=1, got %d", row.Count)
			}
		}
	}
}
