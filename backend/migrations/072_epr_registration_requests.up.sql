CREATE TABLE IF NOT EXISTS epr_registration_requests (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    epr_record_id VARCHAR(255) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    technology_name VARCHAR(255) NOT NULL,
    business_justification TEXT,
    requested_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    decided_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    decision_reason TEXT,
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_epr_registration_requests_status CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX IF NOT EXISTS idx_epr_registration_requests_tenant_status
    ON epr_registration_requests (tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_epr_registration_requests_lookup
    ON epr_registration_requests (tenant_id, epr_record_id, status);

CREATE INDEX IF NOT EXISTS idx_epr_registration_requests_status_created
    ON epr_registration_requests (status, created_at DESC);
