-- Migration: 076_sre_smart_bot_incident_ledger.up.sql
-- Purpose: Persist SRE Smart Bot incidents, findings, evidence, actions, and approvals

CREATE TABLE IF NOT EXISTS sre_incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NULL,
    correlation_key VARCHAR(255) NOT NULL,
    domain VARCHAR(100) NOT NULL,
    incident_type VARCHAR(150) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    severity VARCHAR(32) NOT NULL,
    confidence VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    source VARCHAR(100) NOT NULL,
    first_observed_at TIMESTAMPTZ NOT NULL,
    last_observed_at TIMESTAMPTZ NOT NULL,
    contained_at TIMESTAMPTZ NULL,
    resolved_at TIMESTAMPTZ NULL,
    suppressed_until TIMESTAMPTZ NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sre_incidents_correlation_key
    ON sre_incidents(correlation_key);
CREATE INDEX IF NOT EXISTS idx_sre_incidents_status
    ON sre_incidents(status, last_observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_sre_incidents_domain
    ON sre_incidents(domain, incident_type);
CREATE INDEX IF NOT EXISTS idx_sre_incidents_tenant
    ON sre_incidents(tenant_id);

CREATE TABLE IF NOT EXISTS sre_findings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NULL REFERENCES sre_incidents(id) ON DELETE SET NULL,
    source VARCHAR(100) NOT NULL,
    signal_type VARCHAR(100) NOT NULL,
    signal_key VARCHAR(255) NOT NULL,
    severity VARCHAR(32) NOT NULL,
    confidence VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sre_findings_incident
    ON sre_findings(incident_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_sre_findings_signal
    ON sre_findings(signal_type, signal_key, occurred_at DESC);

CREATE TABLE IF NOT EXISTS sre_incident_evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES sre_incidents(id) ON DELETE CASCADE,
    evidence_type VARCHAR(100) NOT NULL,
    summary TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    captured_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sre_incident_evidence_incident
    ON sre_incident_evidence(incident_id, captured_at DESC);

CREATE TABLE IF NOT EXISTS sre_action_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES sre_incidents(id) ON DELETE CASCADE,
    action_key VARCHAR(150) NOT NULL,
    action_class VARCHAR(32) NOT NULL,
    target_kind VARCHAR(100) NOT NULL,
    target_ref VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    actor_type VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255) NULL,
    approval_required BOOLEAN NOT NULL DEFAULT false,
    requested_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    error_message TEXT NULL,
    result_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sre_action_attempts_incident
    ON sre_action_attempts(incident_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_sre_action_attempts_status
    ON sre_action_attempts(status, requested_at DESC);

CREATE TABLE IF NOT EXISTS sre_approvals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES sre_incidents(id) ON DELETE CASCADE,
    action_attempt_id UUID NULL REFERENCES sre_action_attempts(id) ON DELETE SET NULL,
    channel_provider_id VARCHAR(150) NOT NULL,
    status VARCHAR(32) NOT NULL,
    request_message TEXT NOT NULL,
    requested_by VARCHAR(255) NULL,
    decided_by VARCHAR(255) NULL,
    decision_comment TEXT NULL,
    requested_at TIMESTAMPTZ NOT NULL,
    decided_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sre_approvals_incident
    ON sre_approvals(incident_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_sre_approvals_status
    ON sre_approvals(status, requested_at DESC);
