-- Migration: 007_governance_and_compliance.down.sql
-- Rollback: Governance & Compliance tables

DROP TRIGGER IF EXISTS update_incidents_updated_at ON incidents;
DROP TRIGGER IF EXISTS update_change_requests_updated_at ON change_requests;
DROP TRIGGER IF EXISTS update_compliance_evidence_updated_at ON compliance_evidence;
DROP TRIGGER IF EXISTS update_compliance_assessments_updated_at ON compliance_assessments;
DROP TRIGGER IF EXISTS update_compliance_controls_updated_at ON compliance_controls;
DROP TRIGGER IF EXISTS update_compliance_frameworks_updated_at ON compliance_frameworks;

DROP TABLE IF EXISTS incident_timelines CASCADE;
DROP TABLE IF EXISTS incidents CASCADE;
DROP TABLE IF EXISTS change_requests CASCADE;
DROP TABLE IF EXISTS compliance_evidence CASCADE;
DROP TABLE IF EXISTS compliance_assessments CASCADE;
DROP TABLE IF EXISTS compliance_controls CASCADE;
DROP TABLE IF EXISTS compliance_frameworks CASCADE;
