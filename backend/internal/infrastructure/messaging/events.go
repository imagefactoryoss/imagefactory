package messaging

const (
	EventTypeBuildCreated               = "build.created"
	EventTypeBuildStarted               = "build.started"
	EventTypeBuildCompleted             = "build.completed"
	EventTypeBuildFailed                = "build.failed"
	EventTypeBuildStatusUpdate          = "build.status.updated"
	EventTypeBuildExecutionCompleted    = "build.execution.completed"
	EventTypeBuildExecutionFailed       = "build.execution.failed"
	EventTypeBuildExecutionStatusUpdate = "build.execution.status.updated"

	EventTypeTenantCreated   = "tenant.created"
	EventTypeTenantActivated = "tenant.activated"

	EventTypeInfraProviderCreated = "infra.provider.created"
	EventTypeInfraProviderUpdated = "infra.provider.updated"
	EventTypeInfraProviderDeleted = "infra.provider.deleted"

	EventTypeProjectCreated = "project.created"
	EventTypeProjectUpdated = "project.updated"
	EventTypeProjectDeleted = "project.deleted"

	EventTypeExternalImageImportApprovalRequested = "external.image.import.approval.requested"
	EventTypeExternalImageImportApproved          = "external.image.import.approved"
	EventTypeExternalImageImportRejected          = "external.image.import.rejected"
	EventTypeExternalImageImportDispatchFailed    = "external.image.import.dispatch_failed"
	EventTypeExternalImageImportCompleted         = "external.image.import.completed"
	EventTypeExternalImageImportQuarantined       = "external.image.import.quarantined"
	EventTypeExternalImageImportFailed            = "external.image.import.failed"

	EventTypeQuarantineReleaseRequested      = "quarantine.release_requested"
	EventTypeQuarantineReleased              = "quarantine.released"
	EventTypeQuarantineReleaseFailed         = "quarantine.release_failed"
	EventTypeQuarantineReleaseConsumed       = "quarantine.release_consumed"
	EventTypeQuarantineReleaseAlert          = "quarantine.release_alert"
	EventTypeQuarantineReleaseRecovered      = "quarantine.release_recovered"
	EventTypeQuarantineReleaseDriftDetected  = "quarantine.release_drift_detected"
	EventTypeQuarantineReleaseDriftRecovered = "quarantine.release_drift_recovered"

	EventTypeEPRRegistrationRequested   = "epr.registration.requested"
	EventTypeEPRRegistrationApproved    = "epr.registration.approved"
	EventTypeEPRRegistrationRejected    = "epr.registration.rejected"
	EventTypeEPRRegistrationSuspended   = "epr.registration.suspended"
	EventTypeEPRRegistrationReactivated = "epr.registration.reactivated"
	EventTypeEPRRegistrationRevalidated = "epr.registration.revalidated"
	EventTypeEPRLifecycleExpiring       = "epr.lifecycle.expiring"
	EventTypeEPRLifecycleExpired        = "epr.lifecycle.expired"
)
