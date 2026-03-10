package build

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ConfigVersion represents a snapshot of a build configuration at a point in time
type ConfigVersion struct {
	ID              uuid.UUID      `db:"id" json:"id"`
	BuildID         uuid.UUID      `db:"build_id" json:"build_id"`
	ConfigID        uuid.UUID      `db:"config_id" json:"config_id"`
	VersionNumber   int            `db:"version_number" json:"version_number"`
	Method          string         `db:"method" json:"method"`
	ConfigSnapshot  ConfigSnapshot `db:"config_snapshot" json:"config_snapshot"`
	Description     *string        `db:"description" json:"description,omitempty"`
	CreatedByUserID *uuid.UUID     `db:"created_by_user_id" json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at" json:"updated_at"`
}

// ConfigSnapshot holds the actual configuration data
type ConfigSnapshot struct {
	Data map[string]interface{} `json:"data"`
}

// Value implements the driver.Valuer interface for JSONB
func (cs ConfigSnapshot) Value() (driver.Value, error) {
	return json.Marshal(cs.Data)
}

// Scan implements the sql.Scanner interface for JSONB
func (cs *ConfigSnapshot) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return ErrInvalidConfigSnapshot
	}
	return json.Unmarshal(bytes, &cs.Data)
}

// ConfigVersionDiff represents the differences between two configuration versions
type ConfigVersionDiff struct {
	ID            uuid.UUID          `db:"id" json:"id"`
	VersionIDFrom uuid.UUID          `db:"version_id_from" json:"version_id_from"`
	VersionIDTo   uuid.UUID          `db:"version_id_to" json:"version_id_to"`
	DiffSummary   VersionDiffSummary `db:"diff_summary" json:"diff_summary"`
	ChangesCount  int                `db:"changes_count" json:"changes_count"`
	CreatedAt     time.Time          `db:"created_at" json:"created_at"`
}

// VersionDiffSummary contains the actual differences
type VersionDiffSummary struct {
	Added    map[string]interface{} `json:"added,omitempty"`
	Removed  map[string]interface{} `json:"removed,omitempty"`
	Modified map[string]interface{} `json:"modified,omitempty"`
}

// Value implements the driver.Valuer interface for JSONB
func (vds VersionDiffSummary) Value() (driver.Value, error) {
	return json.Marshal(vds)
}

// Scan implements the sql.Scanner interface for JSONB
func (vds *VersionDiffSummary) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return ErrInvalidVersionDiff
	}
	return json.Unmarshal(bytes, vds)
}

// CreateConfigVersionRequest is the API request to create a new config version
type CreateConfigVersionRequest struct {
	BuildID     uuid.UUID              `json:"build_id" binding:"required"`
	ConfigID    uuid.UUID              `json:"config_id" binding:"required"`
	Method      string                 `json:"method" binding:"required"`
	Description *string                `json:"description,omitempty"`
	Snapshot    map[string]interface{} `json:"snapshot" binding:"required"`
}

// RestoreConfigVersionRequest is the API request to restore a previous config version
type RestoreConfigVersionRequest struct {
	VersionID uuid.UUID `json:"version_id" binding:"required"`
	BuildID   uuid.UUID `json:"build_id" binding:"required"`
}

// ConfigVersionResponse is the API response for a configuration version
type ConfigVersionResponse struct {
	ID              uuid.UUID              `json:"id"`
	BuildID         uuid.UUID              `json:"build_id"`
	ConfigID        uuid.UUID              `json:"config_id"`
	VersionNumber   int                    `json:"version_number"`
	Method          string                 `json:"method"`
	ConfigSnapshot  map[string]interface{} `json:"config_snapshot"`
	Description     *string                `json:"description,omitempty"`
	CreatedByUserID *uuid.UUID             `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ListConfigVersionsResponse is the API response for listing versions
type ListConfigVersionsResponse struct {
	Count    int                     `json:"count"`
	Versions []ConfigVersionResponse `json:"versions"`
}

// ConfigVersionDiffResponse is the API response for version diffs
type ConfigVersionDiffResponse struct {
	ID            uuid.UUID          `json:"id"`
	VersionIDFrom uuid.UUID          `json:"version_id_from"`
	VersionIDTo   uuid.UUID          `json:"version_id_to"`
	DiffSummary   VersionDiffSummary `json:"diff_summary"`
	ChangesCount  int                `json:"changes_count"`
	CreatedAt     time.Time          `json:"created_at"`
}
