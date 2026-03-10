package build

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ConfigVersionRepository defines operations for config version persistence
type ConfigVersionRepository interface {
	Create(ctx context.Context, version *ConfigVersion) error
	GetByID(ctx context.Context, id uuid.UUID) (*ConfigVersion, error)
	GetByBuildIDAndVersion(ctx context.Context, buildID uuid.UUID, versionNumber int) (*ConfigVersion, error)
	ListByBuildID(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]*ConfigVersion, error)
	CountByBuildID(ctx context.Context, buildID uuid.UUID) (int, error)
	GetLatestVersion(ctx context.Context, buildID uuid.UUID) (*ConfigVersion, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CreateDiff(ctx context.Context, diff *ConfigVersionDiff) error
	GetDiff(ctx context.Context, fromVersionID, toVersionID uuid.UUID) (*ConfigVersionDiff, error)
}

// ConfigVersionServiceImpl handles business logic for configuration versioning
type ConfigVersionServiceImpl struct {
	repo ConfigVersionRepository
}

// NewConfigVersionServiceImpl creates a new config version service
func NewConfigVersionServiceImpl(repo ConfigVersionRepository) *ConfigVersionServiceImpl {
	return &ConfigVersionServiceImpl{repo: repo}
}

// CreateVersion creates a new version snapshot of a configuration
func (s *ConfigVersionServiceImpl) CreateVersion(ctx context.Context, req *CreateConfigVersionRequest, userID *uuid.UUID) (*ConfigVersion, error) {
	if req.BuildID == uuid.Nil {
		return nil, ErrInvalidBuildID
	}
	if req.ConfigID == uuid.Nil {
		return nil, ErrInvalidConfigID
	}
	if req.Method == "" {
		return nil, ErrInvalidBuildMethod
	}

	// Create version record
	version := &ConfigVersion{
		ID:              uuid.New(),
		BuildID:         req.BuildID,
		ConfigID:        req.ConfigID,
		Method:          req.Method,
		Description:     req.Description,
		CreatedByUserID: userID,
	}

	// Store the snapshot
	version.ConfigSnapshot.Data = req.Snapshot

	// Save to repository (version_number auto-incremented by trigger)
	if err := s.repo.Create(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create config version: %w", err)
	}

	return version, nil
}

// GetVersion retrieves a specific version
func (s *ConfigVersionServiceImpl) GetVersion(ctx context.Context, versionID uuid.UUID) (*ConfigVersion, error) {
	return s.repo.GetByID(ctx, versionID)
}

// ListVersions retrieves all versions for a build with pagination
func (s *ConfigVersionServiceImpl) ListVersions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]*ConfigVersion, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	versions, err := s.repo.ListByBuildID(ctx, buildID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list config versions: %w", err)
	}

	total, err := s.repo.CountByBuildID(ctx, buildID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count config versions: %w", err)
	}

	return versions, total, nil
}

// GetLatestVersion retrieves the most recent version for a build
func (s *ConfigVersionServiceImpl) GetLatestVersion(ctx context.Context, buildID uuid.UUID) (*ConfigVersion, error) {
	return s.repo.GetLatestVersion(ctx, buildID)
}

// RestoreVersion creates a new version by restoring from a previous version
func (s *ConfigVersionServiceImpl) RestoreVersion(ctx context.Context, buildID uuid.UUID, fromVersionID uuid.UUID, userID *uuid.UUID) (*ConfigVersion, error) {
	// Get the version to restore from
	sourceVersion, err := s.repo.GetByID(ctx, fromVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source version: %w", err)
	}

	// Verify it belongs to the same build
	if sourceVersion.BuildID != buildID {
		return nil, ErrConfigVersionNotFound
	}

	// Create new version with restored snapshot
	newVersion := &ConfigVersion{
		ID:              uuid.New(),
		BuildID:         buildID,
		ConfigID:        sourceVersion.ConfigID,
		Method:          sourceVersion.Method,
		Description:     stringPtr(fmt.Sprintf("Restored from v%d", sourceVersion.VersionNumber)),
		CreatedByUserID: userID,
	}

	// Copy snapshot data
	newVersion.ConfigSnapshot.Data = sourceVersion.ConfigSnapshot.Data

	// Save new version
	if err := s.repo.Create(ctx, newVersion); err != nil {
		return nil, fmt.Errorf("failed to restore config version: %w", err)
	}

	return newVersion, nil
}

// CompareVersions generates a diff between two versions
func (s *ConfigVersionServiceImpl) CompareVersions(ctx context.Context, fromVersionID, toVersionID uuid.UUID) (*ConfigVersionDiff, error) {
	// Check if diff already exists in cache
	existingDiff, err := s.repo.GetDiff(ctx, fromVersionID, toVersionID)
	if err == nil {
		return existingDiff, nil
	}

	// Get both versions
	fromVersion, err := s.repo.GetByID(ctx, fromVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get 'from' version: %w", err)
	}

	toVersion, err := s.repo.GetByID(ctx, toVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get 'to' version: %w", err)
	}

	// Verify they're from the same build
	if fromVersion.BuildID != toVersion.BuildID {
		return nil, ErrVersionsFromDifferentBuilds
	}

	// Calculate diff
	diff := calculateDiff(fromVersion.ConfigSnapshot.Data, toVersion.ConfigSnapshot.Data)

	// Create diff record
	diffRecord := &ConfigVersionDiff{
		ID:            uuid.New(),
		VersionIDFrom: fromVersionID,
		VersionIDTo:   toVersionID,
		DiffSummary:   diff,
		ChangesCount:  countChanges(diff),
	}

	// Save diff record for caching
	if err := s.repo.CreateDiff(ctx, diffRecord); err != nil {
		// Log error but don't fail - diff calculation is secondary
		fmt.Printf("warning: failed to cache diff: %v\n", err)
	}

	return diffRecord, nil
}

// DeleteVersion removes a configuration version
func (s *ConfigVersionServiceImpl) DeleteVersion(ctx context.Context, versionID uuid.UUID) error {
	return s.repo.Delete(ctx, versionID)
}

// Helper functions

// calculateDiff compares two config snapshots and returns the differences
func calculateDiff(oldConfig, newConfig map[string]interface{}) VersionDiffSummary {
	diff := VersionDiffSummary{
		Added:    make(map[string]interface{}),
		Removed:  make(map[string]interface{}),
		Modified: make(map[string]interface{}),
	}

	// Check for removed and modified fields
	for key, oldVal := range oldConfig {
		if newVal, exists := newConfig[key]; exists {
			// Field exists in both - check if modified
			if !deepEqual(oldVal, newVal) {
				diff.Modified[key] = map[string]interface{}{
					"old": oldVal,
					"new": newVal,
				}
			}
		} else {
			// Field removed
			diff.Removed[key] = oldVal
		}
	}

	// Check for added fields
	for key, newVal := range newConfig {
		if _, exists := oldConfig[key]; !exists {
			diff.Added[key] = newVal
		}
	}

	return diff
}

// deepEqual performs deep equality comparison
func deepEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// For complex types, use JSON marshaling for comparison
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// countChanges counts total changes in a diff
func countChanges(diff VersionDiffSummary) int {
	return len(diff.Added) + len(diff.Removed) + len(diff.Modified)
}

// stringPtr is a helper to create string pointer
func stringPtr(s string) *string {
	return &s
}
