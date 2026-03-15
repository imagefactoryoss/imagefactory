package messaging

import "fmt"

type ValidationConfig struct {
	SchemaVersion  string
	ValidateEvents bool
}

var requiredFieldsByType = map[string][]string{
	EventTypeBuildCreated:   {"build_id", "build_name", "build_type"},
	EventTypeBuildStarted:   {"build_id"},
	EventTypeBuildCompleted: {"build_id", "image_id", "image_size", "duration"},
	EventTypeBuildFailed:    {"build_id", "status", "message"},

	EventTypeBuildExecutionStatusUpdate: {"build_id", "status"},
	EventTypeBuildExecutionCompleted:    {"build_id", "status"},
	EventTypeBuildExecutionFailed:       {"build_id", "status", "message"},

	EventTypeTenantCreated:   {"tenant_id", "tenant_name"},
	EventTypeTenantActivated: {"tenant_id"},

	EventTypeInfraProviderCreated: {"provider_id", "provider_type", "name", "created_by"},
	EventTypeInfraProviderUpdated: {"provider_id", "updated_by"},
	EventTypeInfraProviderDeleted: {"provider_id", "deleted_by"},

	EventTypeProjectCreated: {"project_id", "project_name"},
	EventTypeProjectUpdated: {"project_id", "project_name"},
	EventTypeProjectDeleted: {"project_id"},

	EventTypeExternalImageImportApprovalRequested: {"external_image_import_id", "requested_by_user_id", "status"},
	EventTypeExternalImageImportApproved:          {"external_image_import_id", "requested_by_user_id", "status"},
	EventTypeExternalImageImportRejected:          {"external_image_import_id", "requested_by_user_id", "status", "message"},
	EventTypeExternalImageImportDispatchFailed:    {"external_image_import_id", "requested_by_user_id", "status", "message", "dispatch_attempt", "idempotency_key"},
	EventTypeExternalImageImportCompleted:         {"external_image_import_id", "requested_by_user_id", "status"},
	EventTypeExternalImageImportQuarantined:       {"external_image_import_id", "requested_by_user_id", "status"},
	EventTypeExternalImageImportFailed:            {"external_image_import_id", "requested_by_user_id", "status", "message"},

	EventTypeQuarantineReleaseRequested:      {"external_image_import_id", "tenant_id", "actor_id", "release_state", "source_image_digest", "idempotency_key"},
	EventTypeQuarantineReleased:              {"external_image_import_id", "tenant_id", "actor_id", "release_state", "source_image_digest", "idempotency_key"},
	EventTypeQuarantineReleaseFailed:         {"external_image_import_id", "tenant_id", "actor_id", "release_state", "source_image_digest", "message", "idempotency_key"},
	EventTypeQuarantineReleaseAlert:          {"tenant_id", "actor_id", "state", "failure_ratio", "failure_ratio_threshold", "consecutive_failures", "consecutive_failures_threshold", "minimum_samples", "idempotency_key"},
	EventTypeQuarantineReleaseRecovered:      {"tenant_id", "actor_id", "state", "failure_ratio", "failure_ratio_threshold", "consecutive_failures", "consecutive_failures_threshold", "minimum_samples", "idempotency_key"},
	EventTypeQuarantineReleaseDriftDetected:  {"external_image_import_id", "tenant_id", "release_state", "source_image_digest", "internal_image_ref", "released_at", "idempotency_key"},
	EventTypeQuarantineReleaseDriftRecovered: {"external_image_import_id", "tenant_id", "previous_release_state", "source_image_digest", "internal_image_ref", "released_at", "idempotency_key"},

	EventTypeSREFindingObserved:          {"incident_id", "correlation_key", "domain", "incident_type", "status", "severity", "finding_id", "signal_type", "signal_key"},
	EventTypeSREIncidentResolved:         {"incident_id", "correlation_key", "domain", "incident_type", "status", "severity", "resolved_at"},
	EventTypeSREEvidenceAdded:            {"incident_id", "correlation_key", "domain", "incident_type", "status", "evidence_id", "evidence_type"},
	EventTypeSREActionProposed:           {"incident_id", "correlation_key", "domain", "incident_type", "status", "action_attempt_id", "action_key", "action_class", "target_kind", "target_ref"},
	EventTypeSREDetectorFindingObserved:  {"correlation_key", "domain", "incident_type", "summary", "source", "severity", "confidence", "finding_title", "finding_message", "signal_type", "signal_key"},
	EventTypeSREDetectorFindingRecovered: {"correlation_key", "domain", "incident_type", "summary", "source", "resolved_at"},
}

func validateEvent(event Event) error {
	if event.SchemaVersion == "" {
		return fmt.Errorf("schema_version is required")
	}

	requiredFields, ok := requiredFieldsByType[event.Type]
	if !ok {
		return nil
	}

	for _, field := range requiredFields {
		if !payloadHasValue(event.Payload, field) {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}

func payloadHasValue(payload map[string]interface{}, field string) bool {
	if payload == nil {
		return false
	}
	value, ok := payload[field]
	if !ok || value == nil {
		return false
	}
	switch cast := value.(type) {
	case string:
		return cast != ""
	default:
		return true
	}
}
