package imagecatalog

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// BuildEvidence aggregates normalized evidence derived from a successful build execution.
type BuildEvidence struct {
	BuildID              uuid.UUID
	ImageID              uuid.UUID
	BuildCompletedAt     *time.Time
	BuildDurationSeconds int
	ImageDigest          string
	ImageSizeBytes       *int64
	ScanTool             string

	Artifacts         []BuildArtifactEvidence
	Layers            []LayerEvidence
	SBOM              *SBOMEvidence
	VulnerabilityScan *VulnerabilityScanEvidence
}

type BuildArtifactEvidence struct {
	Type        string
	Name        string
	Version     string
	Location    string
	MimeType    string
	SizeBytes   *int64
	SHA256      string
	IsAvailable bool
	ImageID     *uuid.UUID
}

type LayerEvidence struct {
	LayerNumber      int
	Digest           string
	SizeBytes        *int64
	MediaType        string
	HistoryCreatedBy string
	SourceCommand    string
	DiffID           string
}

type SBOMEvidence struct {
	Format       string
	Version      string
	Content      string
	GeneratedBy  string
	ToolVersion  string
	Checksum     string
	Status       string
	Packages     []SBOMPackageEvidence
	DurationSecs *int
}

type SBOMPackageEvidence struct {
	Name           string
	Version        string
	Type           string
	PackageURL     string
	HomepageURL    string
	LicenseName    string
	LicenseSPDXID  string
	PackagePath    string
	LayerDigest    string
	SourceCommand  string
	KnownVulnCount int
	CriticalCount  int
}

type VulnerabilityScanEvidence struct {
	Tool         string
	ToolVersion  string
	Status       string
	Critical     int
	High         int
	Medium       int
	Low          int
	Negligible   int
	Unknown      int
	PassFail     string
	ComplianceOK *bool
	ReportJSON   string
	ErrorMessage string
}

type BuildEvidenceRepository interface {
	PersistBuildEvidence(ctx context.Context, evidence *BuildEvidence) error
}
