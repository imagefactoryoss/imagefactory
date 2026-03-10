package imagecatalog

import (
	"testing"
)

func TestParseSBOMEvidence_PrefersPackageBearingEntryOverSummary(t *testing.T) {
	raw := []byte(`[
		{"name":"sbom-summary","type":"pipeline-result","value":"{\"format\":\"spdx\",\"generator\":\"syft\",\"package_count\":19}"},
		{"name":"sbom.sbom-evidence","type":"taskrun-result","value":"{\"format\":\"spdx\",\"generator\":\"syft\",\"package_count\":2,\"packages\":[{\"name\":\"openssl\",\"version\":\"3.0.0\"},{\"name\":\"curl\",\"version\":\"8.0.0\"}]}"}
	]`)

	sbom := parseSBOMEvidence(raw)
	if sbom == nil {
		t.Fatal("expected SBOM evidence")
	}
	if len(sbom.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(sbom.Packages))
	}
	if sbom.GeneratedBy != "syft" {
		t.Fatalf("expected generator syft, got %q", sbom.GeneratedBy)
	}
}

func TestParseSBOMEvidence_ReturnsSummaryWhenOnlySummaryExists(t *testing.T) {
	raw := []byte(`[
		{"name":"sbom-summary","type":"pipeline-result","value":"{\"format\":\"spdx\",\"generator\":\"syft\",\"package_count\":19}"}
	]`)

	sbom := parseSBOMEvidence(raw)
	if sbom == nil {
		t.Fatal("expected SBOM evidence")
	}
	if len(sbom.Packages) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(sbom.Packages))
	}
	if sbom.GeneratedBy != "syft" {
		t.Fatalf("expected generator syft, got %q", sbom.GeneratedBy)
	}
}
