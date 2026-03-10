package build

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SetDispatchState updates dispatcher retry tracking fields.
func (b *Build) SetDispatchState(attempts int, nextRunAt *time.Time) {
	b.dispatchAttempts = attempts
	b.dispatchNextRunAt = nextRunAt
}

// DispatchAttempts returns the number of dispatcher attempts.
func (b *Build) DispatchAttempts() int {
	return b.dispatchAttempts
}

// DispatchNextRunAt returns the next scheduled dispatch time.
func (b *Build) DispatchNextRunAt() *time.Time {
	return b.dispatchNextRunAt
}

// ID returns the build ID.
func (b *Build) ID() uuid.UUID { return b.id }

// TenantID returns the tenant ID.
func (b *Build) TenantID() uuid.UUID { return b.tenantID }

// ProjectID returns the project ID.
func (b *Build) ProjectID() uuid.UUID { return b.projectID }

// CreatedBy returns the user who created the build (if available).
func (b *Build) CreatedBy() *uuid.UUID { return b.createdBy }

// Manifest returns the build manifest.
func (b *Build) Manifest() BuildManifest { return b.manifest }

// Status returns the build status.
func (b *Build) Status() BuildStatus { return b.status }

// Result returns the build result.
func (b *Build) Result() *BuildResult { return b.result }

// ErrorMessage returns the error message if build failed.
func (b *Build) ErrorMessage() string { return b.errorMsg }

// CreatedAt returns the creation timestamp.
func (b *Build) CreatedAt() time.Time { return b.createdAt }

// StartedAt returns the start timestamp.
func (b *Build) StartedAt() *time.Time { return b.startedAt }

// CompletedAt returns the completion timestamp.
func (b *Build) CompletedAt() *time.Time { return b.completedAt }

// UpdatedAt returns the last update timestamp.
func (b *Build) UpdatedAt() time.Time { return b.updatedAt }

// Version returns the version for concurrency control.
func (b *Build) Version() int { return b.version }

// Config returns the build configuration.
func (b *Build) Config() *BuildConfigData { return b.config }

// SetConfig sets the build configuration.
func (b *Build) SetConfig(config *BuildConfigData) {
	if config != nil {
		config.BuildID = b.id
		b.config = config
		b.updatedAt = time.Now().UTC()
		b.version++
	}
}

// RestoreLifecycleState hydrates persisted execution timing/error state when rebuilding a Build aggregate.
func (b *Build) RestoreLifecycleState(startedAt, completedAt *time.Time, errorMsg string) {
	b.startedAt = startedAt
	b.completedAt = completedAt
	b.errorMsg = errorMsg
}

// Queue transitions the build to queued status.
func (b *Build) Queue() error {
	if !b.canTransitionTo(BuildStatusQueued) {
		return errors.New("can only queue pending builds")
	}

	b.status = BuildStatusQueued
	b.updatedAt = time.Now().UTC()
	b.version++
	return nil
}

// Start transitions the build to running status.
func (b *Build) Start() error {
	if !b.canTransitionTo(BuildStatusRunning) || b.status != BuildStatusQueued {
		return errors.New("can only start queued builds")
	}

	now := time.Now().UTC()
	b.status = BuildStatusRunning
	b.startedAt = &now
	b.updatedAt = now
	b.version++
	return nil
}

// Complete marks the build as completed with result.
func (b *Build) Complete(result BuildResult) error {
	if !b.canTransitionTo(BuildStatusCompleted) || b.status != BuildStatusRunning {
		return errors.New("can only complete running builds")
	}

	now := time.Now().UTC()
	b.status = BuildStatusCompleted
	b.result = &result
	b.completedAt = &now
	b.updatedAt = now
	b.version++
	return nil
}

// Fail marks the build as failed with error message.
func (b *Build) Fail(errorMsg string) error {
	if !b.canTransitionTo(BuildStatusFailed) {
		return errors.New("cannot fail completed or cancelled builds")
	}

	now := time.Now().UTC()
	b.status = BuildStatusFailed
	b.errorMsg = errorMsg
	b.completedAt = &now
	b.updatedAt = now
	b.version++
	return nil
}

// Cancel marks the build as cancelled.
func (b *Build) Cancel() error {
	if !b.canTransitionTo(BuildStatusCancelled) {
		return ErrCannotCancelBuild
	}

	now := time.Now().UTC()
	b.status = BuildStatusCancelled
	b.completedAt = &now
	b.updatedAt = now
	b.version++
	return nil
}

// RetryStart resets a terminal failed/cancelled build and starts it again as a new attempt.
func (b *Build) RetryStart() error {
	if !b.canTransitionTo(BuildStatusRunning) || (b.status != BuildStatusFailed && b.status != BuildStatusCancelled) {
		return errors.New("can only retry failed or cancelled builds")
	}

	now := time.Now().UTC()
	b.status = BuildStatusRunning
	b.result = nil
	b.errorMsg = ""
	b.startedAt = &now
	b.completedAt = nil
	b.updatedAt = now
	b.version++
	return nil
}

func (b *Build) canTransitionTo(next BuildStatus) bool {
	nextStatuses, ok := allowedBuildTransitions[b.status]
	if !ok {
		return false
	}
	_, allowed := nextStatuses[next]
	return allowed
}

// IsTerminal returns true if the build is in a terminal state.
func (b *Build) IsTerminal() bool {
	return b.status == BuildStatusCompleted || b.status == BuildStatusFailed || b.status == BuildStatusCancelled
}

// Duration returns the build duration if completed.
func (b *Build) Duration() time.Duration {
	if b.startedAt == nil || b.completedAt == nil {
		return 0
	}
	return b.completedAt.Sub(*b.startedAt)
}

// InfrastructureType returns the selected infrastructure type.
func (b *Build) InfrastructureType() string { return b.infrastructureType }

// InfrastructureReason returns the reason for infrastructure selection.
func (b *Build) InfrastructureReason() string { return b.infrastructureReason }

// SelectedAt returns when infrastructure was selected.
func (b *Build) SelectedAt() *time.Time { return b.selectedAt }

// SetInfrastructureSelection sets the infrastructure selection details.
func (b *Build) SetInfrastructureSelection(infrastructureType, reason string) {
	b.SetInfrastructureSelectionWithProvider(infrastructureType, reason, nil)
}

// SetInfrastructureSelectionWithProvider sets infrastructure selection with optional provider ID.
func (b *Build) SetInfrastructureSelectionWithProvider(infrastructureType, reason string, providerID *uuid.UUID) {
	now := time.Now().UTC()
	b.infrastructureType = infrastructureType
	b.infrastructureReason = reason
	b.infrastructureProviderID = providerID
	b.selectedAt = &now
	b.updatedAt = now
	b.version++
}

// SetInfrastructureProviderID sets the infrastructure provider selection.
func (b *Build) SetInfrastructureProviderID(providerID *uuid.UUID) {
	b.infrastructureProviderID = providerID
}

// InfrastructureProviderID returns the selected infrastructure provider ID, if any.
func (b *Build) InfrastructureProviderID() *uuid.UUID { return b.infrastructureProviderID }
