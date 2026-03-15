CREATE TABLE IF NOT EXISTS sre_detector_rule_suggestions (
    id UUID PRIMARY KEY,
    tenant_id UUID NULL REFERENCES tenants(id) ON DELETE CASCADE,
    incident_id UUID NULL REFERENCES sre_incidents(id) ON DELETE SET NULL,
    fingerprint TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    query TEXT NOT NULL,
    threshold INT NOT NULL DEFAULT 1,
    domain TEXT NOT NULL,
    incident_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    confidence TEXT NOT NULL DEFAULT 'medium',
    signal_key TEXT,
    source TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    auto_created BOOLEAN NOT NULL DEFAULT FALSE,
    reason TEXT,
    evidence_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    proposed_by TEXT,
    reviewed_by TEXT,
    reviewed_at TIMESTAMP NULL,
    activated_rule_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT sre_detector_rule_suggestions_status_check CHECK (status IN ('pending', 'accepted', 'rejected')),
    CONSTRAINT sre_detector_rule_suggestions_severity_check CHECK (severity IN ('info', 'warning', 'critical')),
    CONSTRAINT sre_detector_rule_suggestions_confidence_check CHECK (confidence IN ('low', 'medium', 'high'))
);

CREATE INDEX IF NOT EXISTS idx_sre_detector_rule_suggestions_tenant ON sre_detector_rule_suggestions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sre_detector_rule_suggestions_status ON sre_detector_rule_suggestions(status);
CREATE INDEX IF NOT EXISTS idx_sre_detector_rule_suggestions_created_at ON sre_detector_rule_suggestions(created_at DESC);

CREATE OR REPLACE FUNCTION update_sre_detector_rule_suggestions_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS sre_detector_rule_suggestions_timestamp_trigger ON sre_detector_rule_suggestions;
CREATE TRIGGER sre_detector_rule_suggestions_timestamp_trigger
BEFORE UPDATE ON sre_detector_rule_suggestions
FOR EACH ROW
EXECUTE FUNCTION update_sre_detector_rule_suggestions_timestamp();
