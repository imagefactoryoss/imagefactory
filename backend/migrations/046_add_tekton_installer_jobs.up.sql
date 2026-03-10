-- Migration: 061_add_tekton_installer_jobs.up.sql
-- Category: Kubernetes Integration
-- Purpose: Track Tekton installer jobs with provider-level active-job locking and audit trail events

CREATE TABLE IF NOT EXISTS tekton_installer_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES infrastructure_providers(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id),
    install_mode VARCHAR(64) NOT NULL CHECK (install_mode IN ('gitops', 'image_factory_installer')),
    asset_version VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled')),
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tekton_installer_jobs_provider_active
    ON tekton_installer_jobs(provider_id)
    WHERE status IN ('pending', 'running');

CREATE INDEX IF NOT EXISTS idx_tekton_installer_jobs_provider_created
    ON tekton_installer_jobs(provider_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tekton_installer_jobs_tenant_created
    ON tekton_installer_jobs(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS tekton_installer_job_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES tekton_installer_jobs(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES infrastructure_providers(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    message TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tekton_installer_job_events_job_created
    ON tekton_installer_job_events(job_id, created_at ASC);

CREATE TABLE IF NOT EXISTS provider_prepare_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES infrastructure_providers(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id),
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled')),
    requested_actions JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_provider_prepare_runs_provider_active
    ON provider_prepare_runs(provider_id)
    WHERE status IN ('pending', 'running');

CREATE INDEX IF NOT EXISTS idx_provider_prepare_runs_provider_created
    ON provider_prepare_runs(provider_id, created_at DESC);

CREATE TABLE IF NOT EXISTS provider_prepare_run_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES provider_prepare_runs(id) ON DELETE CASCADE,
    check_key VARCHAR(128) NOT NULL,
    category VARCHAR(64) NOT NULL,
    severity VARCHAR(16) NOT NULL CHECK (severity IN ('info', 'warn', 'error')),
    ok BOOLEAN NOT NULL,
    message TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_prepare_run_checks_run_created
    ON provider_prepare_run_checks(run_id, created_at ASC);

-- Per-tenant namespace provisioning runs for managed clusters.
-- This is separate from provider-level prepare runs because tasks/pipelines and RBAC are namespace-scoped.
CREATE TABLE IF NOT EXISTS provider_tenant_namespace_prepares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES infrastructure_providers(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    namespace TEXT NOT NULL,
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled')),
    result_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    desired_asset_version VARCHAR(128),
    installed_asset_version VARCHAR(128),
    asset_drift_status VARCHAR(16) NOT NULL DEFAULT 'unknown'
        CHECK (asset_drift_status IN ('current', 'stale', 'unknown')),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_provider_tenant_namespace_prepares_provider_tenant
    ON provider_tenant_namespace_prepares(provider_id, tenant_id);

CREATE INDEX IF NOT EXISTS idx_provider_tenant_namespace_prepares_provider_updated
    ON provider_tenant_namespace_prepares(provider_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_provider_tenant_namespace_prepares_drift_status
    ON provider_tenant_namespace_prepares(provider_id, asset_drift_status, updated_at DESC);
