-- Migration: 007_governance_and_compliance.up.sql
-- Category: Governance & Compliance
-- Purpose: Compliance frameworks, controls, assessments, incidents, and change management

-- ============================================================================
-- 1. COMPLIANCE FRAMEWORKS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance_frameworks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Framework information
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    
    -- Framework type
    framework_type VARCHAR(50) NOT NULL, -- ISO27001, SOC2, HIPAA, PCI-DSS, GDPR, custom
    
    -- Version and documentation
    version VARCHAR(20),
    documentation_url VARCHAR(500),
    
    -- Status and dates
    status VARCHAR(50) DEFAULT 'active', -- active, archived, deprecated
    adopted_at DATE,
    certification_date DATE,
    next_audit_date DATE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(company_id, name)
);

CREATE INDEX IF NOT EXISTS idx_compliance_frameworks_company_id ON compliance_frameworks(company_id);
CREATE INDEX IF NOT EXISTS idx_compliance_frameworks_type ON compliance_frameworks(framework_type);

-- ============================================================================
-- 2. COMPLIANCE CONTROLS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance_controls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    compliance_framework_id UUID NOT NULL REFERENCES compliance_frameworks(id) ON DELETE CASCADE,
    
    -- Control information
    control_code VARCHAR(50) NOT NULL,
    control_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Control details
    objective TEXT,
    scope VARCHAR(255),
    
    -- Responsibility
    responsible_org_unit_id UUID REFERENCES org_units(id) ON DELETE SET NULL,
    responsible_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Evidence and documentation
    evidence_location VARCHAR(500),
    documentation_url VARCHAR(500),
    
    -- Assessment
    status VARCHAR(50) DEFAULT 'pending', -- pending, implemented, partially_implemented, not_applicable
    last_assessed_at TIMESTAMP WITH TIME ZONE,
    assessment_result VARCHAR(50), -- compliant, non_compliant, partially_compliant
    
    -- Remediation tracking
    requires_remediation BOOLEAN DEFAULT false,
    remediation_deadline DATE,
    remediation_notes TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(compliance_framework_id, control_code)
);

CREATE INDEX IF NOT EXISTS idx_compliance_controls_framework_id ON compliance_controls(compliance_framework_id);
CREATE INDEX IF NOT EXISTS idx_compliance_controls_status ON compliance_controls(status);
CREATE INDEX IF NOT EXISTS idx_compliance_controls_org_unit_id ON compliance_controls(responsible_org_unit_id);

-- ============================================================================
-- 3. COMPLIANCE ASSESSMENTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance_assessments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    compliance_framework_id UUID NOT NULL REFERENCES compliance_frameworks(id) ON DELETE CASCADE,
    
    -- Assessment information
    assessment_name VARCHAR(255) NOT NULL,
    assessment_type VARCHAR(50), -- internal, external, audit, certification
    
    -- Scope
    scope_org_unit_id UUID REFERENCES org_units(id) ON DELETE SET NULL,
    
    -- Timeline
    scheduled_date DATE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Assessor information
    conducted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    external_auditor_name VARCHAR(255),
    external_auditor_company VARCHAR(255),
    
    -- Results
    overall_status VARCHAR(50), -- passed, failed, passed_with_conditions, in_progress
    findings_critical_count INTEGER DEFAULT 0,
    findings_major_count INTEGER DEFAULT 0,
    findings_minor_count INTEGER DEFAULT 0,
    
    -- Report
    report_location VARCHAR(500),
    report_content TEXT,
    
    -- Follow-up
    remediation_deadline DATE,
    follow_up_assessment_required BOOLEAN DEFAULT false,
    follow_up_completed_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compliance_assessments_company_id ON compliance_assessments(company_id);
CREATE INDEX IF NOT EXISTS idx_compliance_assessments_framework_id ON compliance_assessments(compliance_framework_id);
CREATE INDEX IF NOT EXISTS idx_compliance_assessments_status ON compliance_assessments(overall_status);

-- ============================================================================
-- 4. COMPLIANCE EVIDENCE TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance_evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    compliance_control_id UUID NOT NULL REFERENCES compliance_controls(id) ON DELETE CASCADE,
    
    -- Evidence information
    evidence_type VARCHAR(50) NOT NULL, -- document, log, screenshot, test_result, report, certification
    evidence_title VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Evidence location
    evidence_url VARCHAR(500),
    evidence_file_path VARCHAR(500),
    
    -- Integrity
    sha256_digest VARCHAR(64),
    
    -- Dates
    evidence_date DATE,
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, archived, expired
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compliance_evidence_control_id ON compliance_evidence(compliance_control_id);
CREATE INDEX IF NOT EXISTS idx_compliance_evidence_type ON compliance_evidence(evidence_type);
CREATE INDEX IF NOT EXISTS idx_compliance_evidence_status ON compliance_evidence(status);

-- ============================================================================
-- 5. CHANGE REQUESTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS change_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Change information
    title VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Change type
    change_type VARCHAR(50) NOT NULL, -- infrastructure, application, security, process, configuration
    impact_level VARCHAR(50) NOT NULL, -- low, medium, high, critical
    
    -- Scope
    affected_org_unit_id UUID REFERENCES org_units(id) ON DELETE SET NULL,
    affected_systems TEXT, -- JSON array of affected system names
    
    -- Requester
    requested_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Implementation plan
    implementation_plan TEXT,
    rollback_plan TEXT,
    estimated_duration_minutes INTEGER,
    
    -- Scheduling
    scheduled_start_time TIMESTAMP WITH TIME ZONE,
    scheduled_end_time TIMESTAMP WITH TIME ZONE,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- draft, submitted, pending_approval, approved, rejected, in_progress, completed, cancelled
    
    -- Approval tracking
    approved_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    
    -- Execution
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    actual_duration_minutes INTEGER,
    
    -- Compliance
    requires_change_control BOOLEAN DEFAULT false,
    affected_controls TEXT, -- JSON array of compliance control IDs
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_change_requests_company_id ON change_requests(company_id);
CREATE INDEX IF NOT EXISTS idx_change_requests_status ON change_requests(status);
CREATE INDEX IF NOT EXISTS idx_change_requests_impact_level ON change_requests(impact_level);

-- ============================================================================
-- 6. INCIDENTS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Incident information
    title VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Severity
    severity VARCHAR(50) NOT NULL DEFAULT 'medium', -- critical, high, medium, low
    incident_type VARCHAR(50), -- security, availability, performance, data_loss, compliance
    
    -- Scope
    affected_systems TEXT, -- JSON array of affected system names
    affected_users_count INTEGER,
    
    -- Timeline
    discovered_at TIMESTAMP WITH TIME ZONE,
    reported_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reported_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Response
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Investigation
    investigating_team VARCHAR(255),
    root_cause TEXT,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- open, investigating, mitigation_in_progress, mitigated, resolved, closed
    
    -- Resolution
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    lessons_learned TEXT,
    
    -- Impact assessment
    system_downtime_minutes INTEGER,
    data_affected BOOLEAN DEFAULT false,
    users_impacted INTEGER,
    
    -- Compliance
    requires_disclosure BOOLEAN DEFAULT false,
    disclosure_date DATE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_incidents_company_id ON incidents(company_id);
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_incidents_reported_at ON incidents(reported_at DESC);

-- ============================================================================
-- 7. INCIDENT TIMELINES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS incident_timelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    
    -- Timeline entry
    event_type VARCHAR(50) NOT NULL, -- detection, escalation, mitigation_started, mitigation_completed, incident_resolved, post_mortem
    event_title VARCHAR(255),
    event_description TEXT,
    
    -- Actor
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_name VARCHAR(255),
    
    -- Timestamp
    event_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Impact
    impact_assessment TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(incident_id, event_timestamp)
);

CREATE INDEX IF NOT EXISTS idx_incident_timelines_incident_id ON incident_timelines(incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_timelines_event_type ON incident_timelines(event_type);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE TRIGGER update_compliance_frameworks_updated_at
    BEFORE UPDATE ON compliance_frameworks
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_compliance_controls_updated_at
    BEFORE UPDATE ON compliance_controls
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_compliance_assessments_updated_at
    BEFORE UPDATE ON compliance_assessments
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_compliance_evidence_updated_at
    BEFORE UPDATE ON compliance_evidence
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_change_requests_updated_at
    BEFORE UPDATE ON change_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_incidents_updated_at
    BEFORE UPDATE ON incidents
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
