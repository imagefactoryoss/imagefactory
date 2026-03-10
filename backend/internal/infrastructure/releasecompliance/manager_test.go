package releasecompliance

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestManagerEvaluate_DetectsAndRecoversWithoutDuplicates(t *testing.T) {
	manager := NewManager()
	tenantID := uuid.New()
	importID := uuid.New()
	record := DriftRecord{
		ExternalImageImportID: importID,
		TenantID:              tenantID,
		ReleaseState:          "withdrawn",
		InternalImageRef:      "registry.local/q/app@sha256:abc",
		SourceImageDigest:     "sha256:abc",
		ReleasedAt:            time.Now().UTC().Add(-time.Hour),
	}

	detected, recovered := manager.Evaluate([]DriftRecord{record})
	if len(detected) != 1 || len(recovered) != 0 {
		t.Fatalf("expected first tick detect=1 recover=0, got detect=%d recover=%d", len(detected), len(recovered))
	}

	detected, recovered = manager.Evaluate([]DriftRecord{record})
	if len(detected) != 0 || len(recovered) != 0 {
		t.Fatalf("expected duplicate drift to be deduped, got detect=%d recover=%d", len(detected), len(recovered))
	}

	detected, recovered = manager.Evaluate(nil)
	if len(detected) != 0 || len(recovered) != 1 {
		t.Fatalf("expected recovery transition, got detect=%d recover=%d", len(detected), len(recovered))
	}
}
