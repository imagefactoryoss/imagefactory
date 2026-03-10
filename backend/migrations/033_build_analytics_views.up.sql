-- Migration: 033_build_analytics_views.up.sql
-- Purpose: Create tenant-aware views and optimize queries for admin build analytics
-- Category: Admin Features - Analytics & Monitoring

-- ============================================================================
-- 1. BUILD ANALYTICS VIEW
-- ============================================================================
CREATE OR REPLACE VIEW v_build_analytics AS
SELECT
    b.tenant_id,
    COUNT(*) AS total_builds,
    COALESCE(SUM(CASE WHEN b.status = 'in_progress' THEN 1 ELSE 0 END), 0) AS running_builds,
    COALESCE(SUM(CASE WHEN b.status = 'success' THEN 1 ELSE 0 END), 0) AS completed_builds,
    COALESCE(SUM(CASE WHEN b.status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_builds,
    COALESCE(ROUND(
        (SUM(CASE WHEN b.status = 'success' THEN 1 ELSE 0 END)::NUMERIC /
         NULLIF(COUNT(*), 0) * 100)::NUMERIC,
        2
    ), 0) AS success_rate,
    COALESCE(ROUND(
        AVG(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)))::NUMERIC,
        0
    )::INTEGER, 0) AS average_duration_seconds,
    COALESCE((
        SELECT COUNT(*)
        FROM builds bq
        WHERE bq.status = 'queued'
          AND bq.tenant_id = b.tenant_id
    ), 0) AS queue_depth,
    CURRENT_TIMESTAMP AS last_updated
FROM builds b
WHERE b.status IN ('queued', 'in_progress', 'success', 'failed', 'cancelled')
GROUP BY b.tenant_id;

-- ============================================================================
-- 2. BUILD PERFORMANCE TRENDS VIEW (7-day history)
-- ============================================================================
CREATE OR REPLACE VIEW v_build_performance_trends AS
SELECT
    b.tenant_id,
    DATE(b.created_at) AS trend_date,
    COUNT(*) AS build_count,
    ROUND(
        AVG(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)))::NUMERIC,
        0
    )::INTEGER AS average_duration_seconds,
    ROUND(
        (SUM(CASE WHEN b.status = 'success' THEN 1 ELSE 0 END)::NUMERIC /
         NULLIF(COUNT(*), 0) * 100)::NUMERIC,
        2
    ) AS success_rate,
    ROUND(
        AVG(EXTRACT(EPOCH FROM (b.started_at - b.created_at)))::NUMERIC,
        0
    )::INTEGER AS average_queue_time_seconds
FROM builds b
WHERE b.created_at >= CURRENT_DATE - INTERVAL '7 days'
  AND b.status IN ('success', 'failed', 'cancelled')
GROUP BY b.tenant_id, DATE(b.created_at)
ORDER BY trend_date DESC;

-- ============================================================================
-- 3. BUILD FAILURE ANALYSIS VIEW
-- ============================================================================
CREATE OR REPLACE VIEW v_build_failures AS
SELECT
    b.id,
    b.tenant_id,
    b.project_id,
    b.status,
    EXTRACT(EPOCH FROM (b.completed_at - b.started_at))::INTEGER AS duration_seconds,
    b.error_message,
    b.created_at,
    b.completed_at
FROM builds b
WHERE b.status IN ('failed', 'cancelled')
  AND b.completed_at IS NOT NULL
ORDER BY b.completed_at DESC;

-- ============================================================================
-- 4. BUILD SLOWEST BUILDS VIEW
-- ============================================================================
CREATE OR REPLACE VIEW v_build_slowest_builds AS
SELECT
    b.id,
    b.tenant_id,
    b.project_id,
    p.name AS project_name,
    EXTRACT(EPOCH FROM (b.completed_at - b.started_at))::INTEGER AS duration_seconds,
    b.status,
    b.created_at,
    b.completed_at
FROM builds b
LEFT JOIN projects p ON p.id = b.project_id AND p.tenant_id = b.tenant_id
WHERE b.status IN ('success', 'failed', 'cancelled')
  AND b.completed_at IS NOT NULL
ORDER BY duration_seconds DESC
LIMIT 100;

-- ============================================================================
-- 5. BUILD FAILURE REASONS AGGREGATION VIEW
-- ============================================================================
CREATE OR REPLACE VIEW v_build_failure_reasons AS
SELECT
    b.tenant_id,
    COALESCE(SUBSTRING(b.error_message FROM 1 FOR 50), 'Unknown Error') AS failure_reason,
    COUNT(*) AS failure_count,
    ROUND(
        (COUNT(*)::NUMERIC /
         NULLIF((
             SELECT COUNT(*)
             FROM builds b2
             WHERE b2.tenant_id = b.tenant_id
               AND b2.status = 'failed'
               AND b2.completed_at >= CURRENT_DATE - INTERVAL '30 days'
         )::NUMERIC, 0) * 100),
        2
    ) AS percentage
FROM builds b
WHERE b.status = 'failed'
  AND b.completed_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY b.tenant_id, SUBSTRING(b.error_message FROM 1 FOR 50)
ORDER BY failure_count DESC;

-- ============================================================================
-- 6. BUILD FAILURE RATE BY PROJECT VIEW
-- ============================================================================
CREATE OR REPLACE VIEW v_build_failure_rate_by_project AS
SELECT
    p.tenant_id,
    p.id,
    p.name AS project_name,
    COUNT(*) AS total_builds,
    SUM(CASE WHEN b.status = 'failed' THEN 1 ELSE 0 END) AS failed_builds,
    ROUND(
        (SUM(CASE WHEN b.status = 'failed' THEN 1 ELSE 0 END)::NUMERIC /
         NULLIF(COUNT(*), 0) * 100)::NUMERIC,
        2
    ) AS failure_rate
FROM projects p
LEFT JOIN builds b ON p.id = b.project_id AND p.tenant_id = b.tenant_id
WHERE b.created_at >= CURRENT_DATE - INTERVAL '30 days'
  AND b.status IN ('success', 'failed', 'cancelled')
GROUP BY p.tenant_id, p.id, p.name
ORDER BY failure_rate DESC;

-- ============================================================================
-- 7. OPTIMIZE EXISTING INDEXES
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_builds_status_created_at
    ON builds(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_builds_project_status_created
    ON builds(project_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_builds_tenant_status_created
    ON builds(tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_builds_created_at_completed_at
    ON builds(created_at DESC, completed_at);

CREATE INDEX IF NOT EXISTS idx_builds_status_completed_at
    ON builds(status, completed_at DESC)
    WHERE status IN ('failed', 'cancelled');
