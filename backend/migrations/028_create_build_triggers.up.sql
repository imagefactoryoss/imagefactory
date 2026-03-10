-- Migration: 031_create_build_triggers.up.sql
-- Purpose: Create build_triggers table for webhook, scheduled, and git-event triggers
-- Date: February 1, 2026

BEGIN;

-- Create build_triggers table
CREATE TABLE IF NOT EXISTS build_triggers (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Foreign keys
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    
    -- Trigger type classification
    trigger_type VARCHAR(50) NOT NULL CHECK (trigger_type IN ('webhook', 'schedule', 'git_event')),
    
    -- Basic metadata
    trigger_name VARCHAR(255) NOT NULL,
    trigger_description TEXT,
    
    -- Webhook-specific configuration
    webhook_url VARCHAR(512),
    webhook_secret VARCHAR(255),
    webhook_events TEXT[],  -- Array of event types: 'push', 'pull_request', 'release', etc.
    
    -- Schedule-specific configuration (cron-based triggers)
    cron_expression VARCHAR(100),  -- Cron format: "0 0 * * *" (every midnight)
    timezone VARCHAR(50) DEFAULT 'UTC',
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    next_trigger_at TIMESTAMP WITH TIME ZONE,
    
    -- Git event specific configuration
    git_provider VARCHAR(50) CHECK (git_provider IN ('github', 'gitlab', 'gitea', 'bitbucket')),
    git_repository_url VARCHAR(512),
    git_branch_pattern VARCHAR(255),  -- Can use wildcards: main, feature/*, release-*
    
    -- Status and control
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    UNIQUE(build_id, trigger_type, webhook_url)  -- No duplicate webhooks for same build
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_build_triggers_tenant_id 
    ON build_triggers(tenant_id);

CREATE INDEX IF NOT EXISTS idx_build_triggers_project_id 
    ON build_triggers(project_id);

CREATE INDEX IF NOT EXISTS idx_build_triggers_build_id 
    ON build_triggers(build_id);

-- Index for finding active scheduled triggers (used by scheduler service)
CREATE INDEX IF NOT EXISTS idx_build_triggers_scheduled_active 
    ON build_triggers(is_active, next_trigger_at) 
    WHERE trigger_type = 'schedule' AND is_active = true;

-- Index for finding webhook triggers by URL
CREATE INDEX IF NOT EXISTS idx_build_triggers_webhook_url 
    ON build_triggers(webhook_url) 
    WHERE trigger_type = 'webhook';

-- Index for finding git event triggers
CREATE INDEX IF NOT EXISTS idx_build_triggers_git_event 
    ON build_triggers(git_repository_url) 
    WHERE trigger_type = 'git_event';

-- Create trigger function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_build_triggers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update updated_at on row update
CREATE TRIGGER update_build_triggers_timestamp
    BEFORE UPDATE ON build_triggers
    FOR EACH ROW
    EXECUTE FUNCTION update_build_triggers_updated_at();

COMMIT;

BEGIN;

-- Create webhook_receipts table for inbound audit + dedupe diagnostics
CREATE TABLE IF NOT EXISTS webhook_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    delivery_id VARCHAR(255),
    event_type VARCHAR(100) NOT NULL,
    event_ref VARCHAR(255),
    event_branch VARCHAR(255),
    event_commit_sha VARCHAR(64),
    repo_url VARCHAR(512),
    event_sha VARCHAR(128),
    signature_valid BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(50) NOT NULL,
    reason TEXT,
    matched_trigger_count INTEGER NOT NULL DEFAULT 0,
    triggered_build_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_receipts_project_provider_delivery
    ON webhook_receipts(project_id, provider, delivery_id)
    WHERE delivery_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_webhook_receipts_project_received_at
    ON webhook_receipts(project_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_receipts_provider_status
    ON webhook_receipts(provider, status);

COMMIT;
