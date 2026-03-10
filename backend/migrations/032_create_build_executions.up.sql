-- Create build_executions table for tracking build execution history and status
CREATE TABLE IF NOT EXISTS build_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES build_configs(id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled')),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    output TEXT,
    error_message TEXT,
    artifacts JSONB,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    monitor_owner VARCHAR(128),
    monitor_lease_expires_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create build_execution_logs table for streaming log entries
CREATE TABLE IF NOT EXISTS build_execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES build_executions(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(20) NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error')),
    message TEXT NOT NULL,
    metadata JSONB
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_build_executions_build_id ON build_executions(build_id);
CREATE INDEX IF NOT EXISTS idx_build_executions_config_id ON build_executions(config_id);
CREATE INDEX IF NOT EXISTS idx_build_executions_monitor_lease ON build_executions (monitor_lease_expires_at) WHERE status IN ('pending', 'running');
CREATE INDEX IF NOT EXISTS idx_build_executions_status ON build_executions(status);
CREATE INDEX IF NOT EXISTS idx_build_executions_created_at ON build_executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_build_execution_logs_execution_id ON build_execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_build_execution_logs_timestamp ON build_execution_logs(timestamp DESC);

-- Create auto-update timestamp trigger for build_executions
CREATE TRIGGER update_build_executions_timestamp
  BEFORE UPDATE ON build_executions
  FOR EACH ROW
  EXECUTE FUNCTION update_timestamp();
