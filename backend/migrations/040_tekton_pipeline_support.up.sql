-- +migrate Up
-- Add Tekton pipeline support tables

-- Pipeline templates table
CREATE TABLE IF NOT EXISTS pipeline_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    build_method VARCHAR(50) NOT NULL,
    template_yaml TEXT NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_template_name_version UNIQUE (name, version)
);

-- Pipeline runs table
CREATE TABLE IF NOT EXISTS pipeline_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    build_execution_id UUID NOT NULL REFERENCES build_executions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    k8s_namespace VARCHAR(255) NOT NULL,
    k8s_name VARCHAR(255) NOT NULL,
    pipeline_template_id UUID REFERENCES pipeline_templates(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    start_time TIMESTAMP WITH TIME ZONE,
    completion_time TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_execution_pipeline_run UNIQUE (build_execution_id)
);

-- Pipeline run logs table
CREATE TABLE IF NOT EXISTS pipeline_run_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    task_name VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    log_level VARCHAR(20) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Pipeline run artifacts table
CREATE TABLE IF NOT EXISTS pipeline_run_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    artifact_type VARCHAR(50) NOT NULL,
    value TEXT,
    path VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_build_execution_id ON pipeline_runs(build_execution_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_tenant_id ON pipeline_runs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_templates_build_method ON pipeline_templates(build_method);
CREATE INDEX IF NOT EXISTS idx_pipeline_templates_active ON pipeline_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_logs_run_id ON pipeline_run_logs(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_logs_timestamp ON pipeline_run_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_artifacts_run_id ON pipeline_run_artifacts(pipeline_run_id);

-- +migrate Down
-- Remove Tekton pipeline support tables

DROP TABLE IF EXISTS pipeline_run_artifacts;
DROP TABLE IF EXISTS pipeline_run_logs;
DROP TABLE IF EXISTS pipeline_runs;
DROP TABLE IF EXISTS pipeline_templates;