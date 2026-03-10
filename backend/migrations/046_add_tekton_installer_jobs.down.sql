-- Migration: 061_add_tekton_installer_jobs.down.sql
-- Category: Kubernetes Integration
-- Purpose: Rollback Tekton installer job tracking tables

DROP TABLE IF EXISTS tekton_installer_job_events;
DROP TABLE IF EXISTS tekton_installer_jobs;
DROP TABLE IF EXISTS provider_prepare_run_checks;
DROP TABLE IF EXISTS provider_prepare_runs;
DROP TABLE IF EXISTS provider_tenant_namespace_prepares;
