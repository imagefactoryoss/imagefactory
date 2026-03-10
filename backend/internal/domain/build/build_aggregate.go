package build

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// BuildResult represents the result of a completed build
type BuildResult struct {
	ImageID     string                 `json:"image_id"`
	ImageDigest string                 `json:"image_digest"`
	Size        int64                  `json:"size"`
	Duration    time.Duration          `json:"duration"`
	Logs        []string               `json:"logs"`
	Artifacts   []string               `json:"artifacts"`
	SBOM        map[string]interface{} `json:"sbom"`
	ScanResults map[string]interface{} `json:"scan_results"`
}

// Build represents the build aggregate root
type Build struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	projectID   uuid.UUID
	createdBy   *uuid.UUID
	manifest    BuildManifest
	config      *BuildConfigData // Method-specific configuration
	status      BuildStatus
	result      *BuildResult
	errorMsg    string
	createdAt   time.Time
	startedAt   *time.Time
	completedAt *time.Time
	updatedAt   time.Time
	version     int

	// Infrastructure selection fields for Phase 2
	infrastructureType       string     // "kubernetes" or "build_node"
	infrastructureReason     string     // Reason for infrastructure selection
	selectedAt               *time.Time // When infrastructure was selected
	infrastructureProviderID *uuid.UUID

	// Dispatcher retry tracking
	dispatchAttempts  int
	dispatchNextRunAt *time.Time
}

// NewBuild creates a new build aggregate
func NewBuild(tenantID, projectID uuid.UUID, manifest BuildManifest, createdBy *uuid.UUID) (*Build, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	if projectID == uuid.Nil {
		return nil, errors.New("project ID is required")
	}

	if err := validateManifest(manifest); err != nil {
		return nil, err
	}

	b := &Build{
		id:        uuid.New(),
		tenantID:  tenantID,
		projectID: projectID,
		createdBy: createdBy,
		manifest:  manifest,
		status:    BuildStatusPending,
		createdAt: time.Now().UTC(),
		updatedAt: time.Now().UTC(),
		version:   1,
	}

	if manifest.InfrastructureType != "" && manifest.InfrastructureType != "auto" {
		b.SetInfrastructureSelectionWithProvider(manifest.InfrastructureType, "user_selected", manifest.InfrastructureProviderID)
	}

	return b, nil
}

// NewBuildFromDB creates a build aggregate from database data without manifest validation.
// Use this for listing/build retrieval where the DB doesn't store full manifests.
func NewBuildFromDB(id, tenantID, projectID uuid.UUID, manifest BuildManifest, status BuildStatus, createdAt, updatedAt time.Time, createdBy *uuid.UUID) *Build {
	return &Build{
		id:        id,
		tenantID:  tenantID,
		projectID: projectID,
		createdBy: createdBy,
		manifest:  manifest,
		status:    status,
		createdAt: createdAt,
		updatedAt: updatedAt,
		version:   1,
	}
}
