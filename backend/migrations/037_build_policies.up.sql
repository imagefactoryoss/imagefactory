-- +migrate Up
-- Build Policies Table
-- Stores configurable policies for build resource limits, scheduling rules, and approval workflows

CREATE TABLE IF NOT EXISTS build_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    policy_type VARCHAR(50) NOT NULL CHECK (policy_type IN ('resource_limit', 'scheduling_rule', 'approval_workflow')),
    policy_key VARCHAR(100) NOT NULL,
    policy_value JSONB NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_build_policies_tenant_id ON build_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_build_policies_type ON build_policies(policy_type);
CREATE INDEX IF NOT EXISTS idx_build_policies_key ON build_policies(policy_key);
CREATE INDEX IF NOT EXISTS idx_build_policies_tenant_type ON build_policies(tenant_id, policy_type);
CREATE INDEX IF NOT EXISTS idx_build_policies_active ON build_policies(is_active);

-- Unique constraint to prevent duplicate policies per tenant
CREATE UNIQUE INDEX idx_build_policies_tenant_key ON build_policies(tenant_id, policy_key);

-- Trigger for updated_at timestamp
CREATE TRIGGER update_build_policies_updated_at
    BEFORE UPDATE ON build_policies
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();