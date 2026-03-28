-- Migration: 079_sre_remediation_pack_runs.up.sql
-- Purpose: Persist remediation pack dry-run/execute records for incident auditability

CREATE TABLE IF NOT EXISTS sre_remediation_pack_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NULL,
    incident_id UUID NOT NULL REFERENCES sre_incidents(id) ON DELETE CASCADE,
    pack_key VARCHAR(150) NOT NULL,
    pack_version VARCHAR(32) NOT NULL,
    run_kind VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    requested_by VARCHAR(255) NULL,
    request_id VARCHAR(255) NULL,
    approval_id UUID NULL REFERENCES sre_approvals(id) ON DELETE SET NULL,
    action_attempt_id UUID NULL REFERENCES sre_action_attempts(id) ON DELETE SET NULL,
    summary TEXT NOT NULL DEFAULT '',
    result_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sre_remediation_pack_runs_incident
    ON sre_remediation_pack_runs(incident_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sre_remediation_pack_runs_tenant_status
    ON sre_remediation_pack_runs(tenant_id, status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sre_remediation_pack_runs_request_id
    ON sre_remediation_pack_runs(request_id)
    WHERE request_id IS NOT NULL;
