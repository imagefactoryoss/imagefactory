package postgres

import (
	"testing"

	"github.com/srikarm/image-factory/internal/application/imagecatalog"
)

func TestBuildEvidencePresenceHelpers(t *testing.T) {
	t.Run("hasLayerEvidence", func(t *testing.T) {
		if hasLayerEvidence(nil) {
			t.Fatal("expected false for nil evidence")
		}
		if hasLayerEvidence(&imagecatalog.BuildEvidence{}) {
			t.Fatal("expected false for empty layer list")
		}
		if hasLayerEvidence(&imagecatalog.BuildEvidence{
			Layers: []imagecatalog.LayerEvidence{{Digest: "   "}},
		}) {
			t.Fatal("expected false for blank layer digest")
		}
		if !hasLayerEvidence(&imagecatalog.BuildEvidence{
			Layers: []imagecatalog.LayerEvidence{{Digest: "sha256:abc"}},
		}) {
			t.Fatal("expected true for non-empty layer digest")
		}
	})

	t.Run("hasSBOMContent", func(t *testing.T) {
		if hasSBOMContent(nil) {
			t.Fatal("expected false for nil evidence")
		}
		if hasSBOMContent(&imagecatalog.BuildEvidence{
			SBOM: &imagecatalog.SBOMEvidence{Content: "   "},
		}) {
			t.Fatal("expected false for blank sbom content")
		}
		if !hasSBOMContent(&imagecatalog.BuildEvidence{
			SBOM: &imagecatalog.SBOMEvidence{Content: "{}"},
		}) {
			t.Fatal("expected true for populated sbom content")
		}
	})

	t.Run("hasSBOMPackages", func(t *testing.T) {
		if hasSBOMPackages(&imagecatalog.BuildEvidence{
			SBOM: &imagecatalog.SBOMEvidence{Content: "{}"},
		}) {
			t.Fatal("expected false when sbom has no packages")
		}
		if hasSBOMPackages(&imagecatalog.BuildEvidence{
			SBOM: &imagecatalog.SBOMEvidence{
				Content: "{}",
				Packages: []imagecatalog.SBOMPackageEvidence{
					{Name: "   "},
				},
			},
		}) {
			t.Fatal("expected false when package names are blank")
		}
		if !hasSBOMPackages(&imagecatalog.BuildEvidence{
			SBOM: &imagecatalog.SBOMEvidence{
				Content: "{}",
				Packages: []imagecatalog.SBOMPackageEvidence{
					{Name: "openssl"},
				},
			},
		}) {
			t.Fatal("expected true with non-empty package")
		}
	})

	t.Run("hasLayerPackageEvidence", func(t *testing.T) {
		if hasLayerPackageEvidence(&imagecatalog.BuildEvidence{
			SBOM: &imagecatalog.SBOMEvidence{
				Content: "{}",
				Packages: []imagecatalog.SBOMPackageEvidence{
					{Name: "openssl"},
				},
			},
		}) {
			t.Fatal("expected false without layer digest")
		}
		if !hasLayerPackageEvidence(&imagecatalog.BuildEvidence{
			SBOM: &imagecatalog.SBOMEvidence{
				Content: "{}",
				Packages: []imagecatalog.SBOMPackageEvidence{
					{Name: "openssl", LayerDigest: "sha256:abc"},
				},
			},
		}) {
			t.Fatal("expected true with package and layer digest")
		}
	})

	t.Run("hasVulnerabilityEvidence", func(t *testing.T) {
		if hasVulnerabilityEvidence(nil) {
			t.Fatal("expected false for nil evidence")
		}
		if hasVulnerabilityEvidence(&imagecatalog.BuildEvidence{}) {
			t.Fatal("expected false when vulnerability scan is absent")
		}
		if !hasVulnerabilityEvidence(&imagecatalog.BuildEvidence{
			VulnerabilityScan: &imagecatalog.VulnerabilityScanEvidence{Status: "completed"},
		}) {
			t.Fatal("expected true when vulnerability scan is present")
		}
	})
}

func TestDeriveEvidenceStatus(t *testing.T) {
	cases := []struct {
		name       string
		isFresh    bool
		dataExists bool
		want       string
	}{
		{name: "fresh", isFresh: true, dataExists: true, want: evidenceStatusFresh},
		{name: "stale", isFresh: false, dataExists: true, want: evidenceStatusStale},
		{name: "unavailable", isFresh: false, dataExists: false, want: evidenceStatusUnavailable},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := deriveEvidenceStatus(tc.isFresh, tc.dataExists)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestClampString(t *testing.T) {
	if got := clampString("  abc  ", 10); got != "abc" {
		t.Fatalf("expected trimmed string, got %q", got)
	}
	if got := clampString("abcdef", 3); got != "abc" {
		t.Fatalf("expected truncated string, got %q", got)
	}
	if got := clampString("", 3); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := clampString("  ", 3); got != "" {
		t.Fatalf("expected empty string after trim, got %q", got)
	}
	if got := clampString("abcdef", 0); got != "abcdef" {
		t.Fatalf("expected unchanged string when maxLen<=0, got %q", got)
	}
}
