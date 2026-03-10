--
-- PostgreSQL database dump
--

\restrict DMXh4VktFgXOfdgS0iVxO9Cg4u6KsaTZlHA5Wk4qHPCc2aIDRAsSEP2BoXkg9eW

-- Dumped from database version 17.7 (Postgres.app)
-- Dumped by pg_dump version 17.7 (Postgres.app)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: image_status_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.image_status_enum AS ENUM (
    'draft',
    'published',
    'deprecated',
    'archived'
);


--
-- Name: image_visibility_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.image_visibility_enum AS ENUM (
    'public',
    'tenant',
    'private'
);


--
-- Name: generate_numeric_id(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.generate_numeric_id() RETURNS integer
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_id INTEGER;
BEGIN
    LOOP
        v_id := (FLOOR(RANDOM() * (999999 - 100000 + 1)) + 100000)::INTEGER;
        IF NOT EXISTS (SELECT 1 FROM tenants WHERE numeric_id = v_id) THEN
            RETURN v_id;
        END IF;
    END LOOP;
END;
$$;


--
-- Name: set_numeric_id(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.set_numeric_id() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.numeric_id IS NULL THEN
        NEW.numeric_id := generate_numeric_id();
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: update_build_triggers_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_build_triggers_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_config_templates_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_config_templates_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_email_queue_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_email_queue_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_project_members_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_project_members_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_rbac_roles_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_rbac_roles_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_system_configs_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_system_configs_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$;


--
-- Name: update_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_user_invitations_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_invitations_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_user_role_assignments_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_role_assignments_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_workers_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_workers_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: api_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_keys (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    key_hash character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    scopes text[] DEFAULT ARRAY[]::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp with time zone,
    last_used_at timestamp with time zone,
    revoked_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT ck_key_hash_not_empty CHECK (((key_hash)::text <> ''::text)),
    CONSTRAINT ck_name_not_empty CHECK (((name)::text <> ''::text))
);


--
-- Name: TABLE api_keys; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.api_keys IS 'External API keys for tenant-to-tenant authentication';


--
-- Name: COLUMN api_keys.key_hash; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_keys.key_hash IS 'Bcrypt-hashed API key for secure storage';


--
-- Name: COLUMN api_keys.scopes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_keys.scopes IS 'Array of permission scopes for this API key';


--
-- Name: COLUMN api_keys.last_used_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_keys.last_used_at IS 'Track last successful usage for security audit';


--
-- Name: COLUMN api_keys.revoked_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.api_keys.revoked_at IS 'Timestamp when key was revoked (soft delete)';


--
-- Name: approval_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.approval_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    approval_workflow_id uuid NOT NULL,
    request_type character varying(50),
    resource_type character varying(100),
    resource_id uuid,
    requested_by_user_id uuid NOT NULL,
    requested_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    request_context text,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    approved_by_user_id uuid,
    approved_at timestamp with time zone,
    rejection_reason text,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: approval_workflows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.approval_workflows (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    workflow_type character varying(50) NOT NULL,
    required_approvers_count integer DEFAULT 1,
    approver_roles text,
    approval_timeout_hours integer DEFAULT 24,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: audit_event_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_event_types (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_type character varying(100) NOT NULL,
    category character varying(50) NOT NULL,
    description text NOT NULL,
    severity character varying(20) DEFAULT 'info'::character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT audit_event_types_severity_check CHECK (((severity)::text = ANY ((ARRAY['info'::character varying, 'warning'::character varying, 'error'::character varying, 'critical'::character varying])::text[])))
);


--
-- Name: audit_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid,
    user_id uuid,
    event_type character varying(100) NOT NULL,
    severity character varying(20) DEFAULT 'info'::character varying NOT NULL,
    resource character varying(255) NOT NULL,
    action character varying(100) NOT NULL,
    ip_address character varying(45),
    user_agent text,
    details jsonb,
    message text NOT NULL,
    "timestamp" timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT audit_events_severity_check CHECK (((severity)::text = ANY ((ARRAY['info'::character varying, 'warning'::character varying, 'error'::character varying, 'critical'::character varying])::text[])))
);


--
-- Name: build_artifacts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_artifacts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    build_id uuid NOT NULL,
    artifact_type character varying(50) NOT NULL,
    artifact_name character varying(255) NOT NULL,
    artifact_version character varying(100),
    artifact_location character varying(500),
    artifact_mime_type character varying(100),
    artifact_size_bytes bigint,
    sha256_digest character varying(64),
    is_available boolean DEFAULT true,
    retention_policy character varying(50),
    expires_at timestamp with time zone,
    image_id uuid,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: build_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_configs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    build_id uuid NOT NULL,
    build_method character varying(50) NOT NULL,
    sbom_tool character varying(50),
    scan_tool character varying(50),
    registry_type character varying(50),
    secret_manager_type character varying(50),
    build_args jsonb,
    environment jsonb,
    secrets jsonb,
    metadata jsonb,
    dockerfile text,
    build_context character varying(255),
    cache_enabled boolean,
    cache_repo character varying(255),
    platforms jsonb,
    cache_from jsonb,
    cache_to character varying(255),
    target_stage character varying(255),
    builder character varying(255),
    buildpacks jsonb,
    packer_template text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_build_method CHECK (((build_method)::text = ANY ((ARRAY['kaniko'::character varying, 'buildx'::character varying, 'container'::character varying, 'docker'::character varying, 'nix'::character varying, 'paketo'::character varying, 'packer'::character varying])::text[])))
);


--
-- Name: build_execution_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_execution_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    execution_id uuid NOT NULL,
    "timestamp" timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    level character varying(20) NOT NULL,
    message text NOT NULL,
    metadata jsonb,
    CONSTRAINT build_execution_logs_level_check CHECK (((level)::text = ANY ((ARRAY['debug'::character varying, 'info'::character varying, 'warn'::character varying, 'error'::character varying])::text[])))
);


--
-- Name: build_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_executions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    build_id uuid NOT NULL,
    config_id uuid NOT NULL,
    status character varying(50) NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    duration_seconds integer,
    output text,
    error_message text,
    artifacts jsonb,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT build_executions_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'running'::character varying, 'success'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[])))
);


--
-- Name: build_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_history (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    build_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    build_method character varying(50) NOT NULL,
    worker_id uuid,
    duration_seconds integer NOT NULL,
    success boolean DEFAULT true NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT positive_duration CHECK ((duration_seconds > 0)),
    CONSTRAINT valid_build_method CHECK (((build_method)::text = ANY ((ARRAY['kaniko'::character varying, 'buildx'::character varying, 'container'::character varying, 'paketo'::character varying, 'packer'::character varying])::text[])))
);


--
-- Name: TABLE build_history; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.build_history IS 'Historical build metrics for ETA prediction and performance analysis';


--
-- Name: COLUMN build_history.build_method; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.build_history.build_method IS 'Build method used (for grouping by method in ETA calculations)';


--
-- Name: COLUMN build_history.duration_seconds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.build_history.duration_seconds IS 'Actual build execution time in seconds';


--
-- Name: COLUMN build_history.success; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.build_history.success IS 'Whether build succeeded (for filtering successful builds in ETA)';


--
-- Name: build_metrics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_metrics (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    build_id uuid NOT NULL,
    total_duration_seconds integer,
    docker_build_duration_seconds integer,
    docker_push_duration_seconds integer,
    peak_memory_usage_mb integer,
    cpu_usage_percent numeric(5,2),
    disk_read_bytes bigint,
    disk_write_bytes bigint,
    total_layers integer,
    reused_layers integer,
    new_layers integer,
    final_image_size_bytes bigint,
    uncompressed_size_bytes bigint,
    compression_ratio numeric(5,2),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: build_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    policy_type character varying(50) NOT NULL,
    policy_key character varying(100) NOT NULL,
    policy_value jsonb NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by uuid,
    updated_by uuid,
    CONSTRAINT build_policies_policy_type_check CHECK (((policy_type)::text = ANY ((ARRAY['resource_limit'::character varying, 'scheduling_rule'::character varying, 'approval_workflow'::character varying])::text[])))
);


--
-- Name: build_steps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_steps (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    build_id uuid NOT NULL,
    step_number integer NOT NULL,
    step_name character varying(255),
    instruction_type character varying(50),
    instruction_line character varying(2000),
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    duration_seconds integer,
    layer_digest character varying(255),
    layer_size_bytes bigint,
    cached boolean DEFAULT false,
    error_message text,
    error_code character varying(50),
    stdout text,
    stderr text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: build_triggers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.build_triggers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    build_id uuid NOT NULL,
    created_by uuid NOT NULL,
    trigger_type character varying(50) NOT NULL,
    trigger_name character varying(255) NOT NULL,
    trigger_description text,
    webhook_url character varying(512),
    webhook_secret character varying(255),
    webhook_events text[],
    cron_expression character varying(100),
    timezone character varying(50) DEFAULT 'UTC'::character varying,
    last_triggered_at timestamp with time zone,
    next_trigger_at timestamp with time zone,
    git_provider character varying(50),
    git_repository_url character varying(512),
    git_branch_pattern character varying(255),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT build_triggers_git_provider_check CHECK (((git_provider)::text = ANY ((ARRAY['github'::character varying, 'gitlab'::character varying, 'gitea'::character varying, 'bitbucket'::character varying])::text[]))),
    CONSTRAINT build_triggers_trigger_type_check CHECK (((trigger_type)::text = ANY ((ARRAY['webhook'::character varying, 'schedule'::character varying, 'git_event'::character varying])::text[])))
);


--
-- Name: builds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.builds (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    project_id uuid NOT NULL,
    image_id uuid,
    build_number integer NOT NULL,
    triggered_by_user_id uuid,
    triggered_by_git_event character varying(50),
    git_commit character varying(40),
    git_branch character varying(100),
    git_author_name character varying(255),
    git_author_email character varying(255),
    git_message text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    status character varying(50) DEFAULT 'queued'::character varying NOT NULL,
    infrastructure_type character varying(50),
    infrastructure_reason text,
    infrastructure_provider_id uuid,
    selected_at timestamp with time zone,
    error_message text,
    build_log_url character varying(500),
    cleanup_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    assigned_node_id uuid,
    cpu_required numeric(10,2) DEFAULT 1,
    memory_required_gb numeric(10,2) DEFAULT 2,
    finished_at timestamp with time zone,
    CONSTRAINT valid_infrastructure_type CHECK (((infrastructure_type IS NULL) OR ((infrastructure_type)::text = ANY ((ARRAY['kubernetes'::character varying, 'build_node'::character varying])::text[]))))
);


--
-- Name: COLUMN builds.infrastructure_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.builds.infrastructure_type IS 'Infrastructure type selected for build execution (kubernetes or build_node)';


--
-- Name: COLUMN builds.infrastructure_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.builds.infrastructure_reason IS 'Reason for infrastructure selection';


--
-- Name: COLUMN builds.infrastructure_provider_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.builds.infrastructure_provider_id IS 'Selected infrastructure provider ID for build execution';


--
-- Name: COLUMN builds.selected_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.builds.selected_at IS 'Timestamp when infrastructure was selected';


--
-- Name: catalog_image_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.catalog_image_tags (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    catalog_image_id uuid NOT NULL,
    tag character varying(100) NOT NULL,
    category character varying(50),
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: catalog_image_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.catalog_image_versions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    catalog_image_id uuid NOT NULL,
    version character varying(100) NOT NULL,
    digest character varying(255),
    size_bytes bigint,
    tags jsonb DEFAULT '[]'::jsonb,
    metadata jsonb DEFAULT '{}'::jsonb,
    published_at timestamp with time zone DEFAULT now(),
    deprecated_at timestamp with time zone
);


--
-- Name: catalog_images; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.catalog_images (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    visibility public.image_visibility_enum DEFAULT 'tenant'::public.image_visibility_enum NOT NULL,
    status public.image_status_enum DEFAULT 'draft'::public.image_status_enum NOT NULL,
    repository_url character varying(500),
    registry_provider character varying(50),
    architecture character varying(50),
    os character varying(50),
    language character varying(50),
    framework character varying(100),
    version character varying(50),
    tags jsonb DEFAULT '[]'::jsonb,
    metadata jsonb DEFAULT '{}'::jsonb,
    size_bytes bigint,
    pull_count bigint DEFAULT 0,
    created_by uuid,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deprecated_at timestamp with time zone,
    archived_at timestamp with time zone
);


--
-- Name: change_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.change_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    description text,
    change_type character varying(50) NOT NULL,
    impact_level character varying(50) NOT NULL,
    affected_org_unit_id uuid,
    affected_systems text,
    requested_by_user_id uuid NOT NULL,
    requested_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    implementation_plan text,
    rollback_plan text,
    estimated_duration_minutes integer,
    scheduled_start_time timestamp with time zone,
    scheduled_end_time timestamp with time zone,
    status character varying(50) DEFAULT 'draft'::character varying NOT NULL,
    approved_by_user_id uuid,
    approved_at timestamp with time zone,
    rejection_reason text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    actual_duration_minutes integer,
    requires_change_control boolean DEFAULT false,
    affected_controls text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: companies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.companies (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    industry character varying(100),
    website_url character varying(255),
    size character varying(50),
    headquarters_country character varying(100),
    subscription_tier character varying(50) DEFAULT 'standard'::character varying NOT NULL,
    billing_contact_email character varying(255),
    enforce_mfa boolean DEFAULT false,
    enforce_image_signing boolean DEFAULT true,
    max_concurrent_builds integer DEFAULT 10,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp with time zone
);


--
-- Name: compliance_assessments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.compliance_assessments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    compliance_framework_id uuid NOT NULL,
    assessment_name character varying(255) NOT NULL,
    assessment_type character varying(50),
    scope_org_unit_id uuid,
    scheduled_date date,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    conducted_by_user_id uuid,
    external_auditor_name character varying(255),
    external_auditor_company character varying(255),
    overall_status character varying(50),
    findings_critical_count integer DEFAULT 0,
    findings_major_count integer DEFAULT 0,
    findings_minor_count integer DEFAULT 0,
    report_location character varying(500),
    report_content text,
    remediation_deadline date,
    follow_up_assessment_required boolean DEFAULT false,
    follow_up_completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: compliance_controls; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.compliance_controls (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    compliance_framework_id uuid NOT NULL,
    control_code character varying(50) NOT NULL,
    control_name character varying(255) NOT NULL,
    description text,
    objective text,
    scope character varying(255),
    responsible_org_unit_id uuid,
    responsible_user_id uuid,
    evidence_location character varying(500),
    documentation_url character varying(500),
    status character varying(50) DEFAULT 'pending'::character varying,
    last_assessed_at timestamp with time zone,
    assessment_result character varying(50),
    requires_remediation boolean DEFAULT false,
    remediation_deadline date,
    remediation_notes text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: compliance_evidence; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.compliance_evidence (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    compliance_control_id uuid NOT NULL,
    evidence_type character varying(50) NOT NULL,
    evidence_title character varying(255) NOT NULL,
    description text,
    evidence_url character varying(500),
    evidence_file_path character varying(500),
    sha256_digest character varying(64),
    evidence_date date,
    expires_at timestamp with time zone,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: compliance_frameworks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.compliance_frameworks (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    display_name character varying(255),
    description text,
    framework_type character varying(50) NOT NULL,
    version character varying(20),
    documentation_url character varying(500),
    status character varying(50) DEFAULT 'active'::character varying,
    adopted_at date,
    certification_date date,
    next_audit_date date,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: config_template_shares; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_template_shares (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    template_id uuid NOT NULL,
    shared_with_user_id uuid NOT NULL,
    can_use boolean DEFAULT true,
    can_edit boolean DEFAULT false,
    can_delete boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: config_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    created_by_user_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    method character varying(50) NOT NULL,
    template_data jsonb NOT NULL,
    is_shared boolean DEFAULT false,
    is_public boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT config_templates_method_check CHECK (((method)::text = ANY ((ARRAY['packer'::character varying, 'buildx'::character varying, 'kaniko'::character varying, 'docker'::character varying, 'nix'::character varying])::text[])))
);


--
-- Name: container_registries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.container_registries (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    registry_type character varying(50) NOT NULL,
    registry_url character varying(500) NOT NULL,
    auth_method character varying(50) DEFAULT 'credentials'::character varying NOT NULL,
    username character varying(255),
    password_encrypted character varying(500),
    api_token_encrypted character varying(500),
    verify_ssl boolean DEFAULT true,
    ca_certificate_pem text,
    supports_push boolean DEFAULT true,
    supports_pull boolean DEFAULT true,
    supports_delete boolean DEFAULT false,
    status character varying(50) DEFAULT 'active'::character varying,
    is_default boolean DEFAULT false,
    last_connectivity_check_at timestamp with time zone,
    last_connectivity_status character varying(50),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: container_repositories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.container_repositories (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    registry_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    full_name character varying(500) NOT NULL,
    description text,
    owned_by_org_unit_id uuid,
    visibility character varying(50) DEFAULT 'private'::character varying,
    image_count integer DEFAULT 0,
    total_size_bytes bigint DEFAULT 0,
    immutable_tags boolean DEFAULT false,
    cleanup_policy character varying(50),
    max_retention_days integer,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: cve_database; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cve_database (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    cve_id character varying(20) NOT NULL,
    cve_description text,
    cvss_v3_score numeric(3,1),
    cvss_v3_vector character varying(100),
    cvss_v3_severity character varying(20),
    cvss_v2_score numeric(3,1),
    cvss_v2_vector character varying(100),
    cvss_v2_severity character varying(20),
    confidentiality_impact character varying(20),
    integrity_impact character varying(20),
    availability_impact character varying(20),
    published_date date,
    modified_date date,
    cwe_id character varying(10),
    "references" text,
    is_exploited_in_wild boolean DEFAULT false,
    exploit_count integer DEFAULT 0,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: deployment_environments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.deployment_environments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    display_name character varying(255),
    description text,
    environment_type character varying(50) NOT NULL,
    owned_by_org_unit_id uuid,
    infrastructure_provider character varying(50),
    cluster_name character varying(255),
    namespace character varying(100),
    requires_approval boolean DEFAULT false,
    max_concurrent_deployments integer DEFAULT 1,
    auto_rollback_on_failure boolean DEFAULT true,
    cpu_limit character varying(50),
    memory_limit character varying(50),
    storage_limit_gb integer,
    log_aggregation_enabled boolean DEFAULT true,
    metrics_collection_enabled boolean DEFAULT true,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: deployments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.deployments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    image_id uuid NOT NULL,
    deployment_environment_id uuid NOT NULL,
    config_manifest text,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    triggered_by_user_id uuid,
    triggered_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    desired_replicas integer DEFAULT 1,
    ready_replicas integer DEFAULT 0,
    updated_replicas integer DEFAULT 0,
    previous_deployment_id uuid,
    error_message text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: email_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_queue (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    to_email character varying(255) NOT NULL,
    from_email character varying(255) NOT NULL,
    subject text NOT NULL,
    body_text text,
    body_html text,
    email_type character varying(50) DEFAULT 'notification'::character varying NOT NULL,
    priority integer DEFAULT 5,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    retry_count integer DEFAULT 0,
    max_retries integer DEFAULT 3,
    last_error text,
    next_retry_at timestamp without time zone,
    smtp_host character varying(255),
    smtp_port integer,
    smtp_username character varying(255),
    smtp_password character varying(255),
    smtp_use_tls boolean DEFAULT false,
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    sent_at timestamp without time zone,
    processed_at timestamp without time zone,
    cc_email character varying(255)
);


--
-- Name: environment_access; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.environment_access (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    deployment_environment_id uuid NOT NULL,
    user_id uuid NOT NULL,
    access_level character varying(50) DEFAULT 'viewer'::character varying NOT NULL,
    ip_whitelist text,
    time_restriction character varying(100),
    requires_approval_for_deployment boolean DEFAULT false,
    granted_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    granted_by_user_id uuid,
    expires_at timestamp with time zone,
    last_accessed_at timestamp with time zone,
    access_count integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: external_services; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_services (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    url character varying(500) NOT NULL,
    api_key character varying(1000),
    enabled boolean DEFAULT true NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    headers jsonb DEFAULT '{}'::jsonb
);


--
-- Name: COLUMN external_services.headers; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_services.headers IS 'Custom HTTP headers for authentication and other purposes, stored as JSON object';


--
-- Name: external_tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_tenants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id character varying(8) NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    description text,
    contact_email character varying(255),
    industry character varying(100),
    country character varying(2),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: TABLE external_tenants; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.external_tenants IS 'Represents tenants from external systems';


--
-- Name: COLUMN external_tenants.tenant_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_tenants.tenant_id IS 'Unique 8-digit identifier for the tenant';


--
-- Name: COLUMN external_tenants.slug; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_tenants.slug IS 'URL-friendly identifier for the tenant';


--
-- Name: git_integration; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.git_integration (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    git_repository_id uuid NOT NULL,
    integration_type character varying(50) NOT NULL,
    is_enabled boolean DEFAULT true,
    webhook_event_types text,
    webhook_url character varying(500),
    webhook_secret_encrypted character varying(500),
    ci_cd_provider character varying(50),
    ci_cd_config_file_path character varying(255),
    auto_build_on_push boolean DEFAULT false,
    auto_build_branches text,
    auto_build_dockerfile_path character varying(255),
    run_security_scans boolean DEFAULT false,
    run_compliance_checks boolean DEFAULT false,
    scan_frequency character varying(50),
    status character varying(50) DEFAULT 'active'::character varying,
    last_sync_at timestamp with time zone,
    last_sync_status character varying(50),
    last_error_message text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: git_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.git_providers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    provider_key character varying(50) NOT NULL,
    display_name character varying(100) NOT NULL,
    provider_type character varying(50) NOT NULL,
    api_base_url character varying(255),
    supports_api boolean DEFAULT false NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT git_providers_provider_type_check CHECK (((provider_type)::text = ANY ((ARRAY['generic'::character varying, 'hosted'::character varying])::text[])))
);


--
-- Name: TABLE git_providers; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.git_providers IS 'Catalog of supported Git providers for repository configuration';


--
-- Name: git_repositories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.git_repositories (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    url character varying(500) NOT NULL,
    description text,
    provider character varying(50) NOT NULL,
    provider_repo_id character varying(255),
    ssh_key_name character varying(255),
    personal_access_token_encrypted character varying(500),
    default_branch character varying(100) DEFAULT 'main'::character varying,
    webhook_url character varying(500),
    webhook_secret_encrypted character varying(500),
    webhook_enabled boolean DEFAULT true,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: group_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_members (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    group_id uuid NOT NULL,
    user_id uuid NOT NULL,
    is_group_admin boolean DEFAULT false,
    added_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    added_by uuid,
    removed_at timestamp with time zone
);


--
-- Name: image_layers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.image_layers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    image_id uuid NOT NULL,
    layer_number integer NOT NULL,
    layer_digest character varying(255) NOT NULL,
    layer_size_bytes bigint,
    media_type character varying(100),
    is_base_layer boolean DEFAULT false,
    base_image_name character varying(255),
    base_image_tag character varying(100),
    used_in_builds_count integer DEFAULT 1,
    last_used_in_build_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: image_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.image_metadata (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    image_id uuid NOT NULL,
    docker_config_digest character varying(255),
    docker_manifest_digest character varying(255),
    total_layer_count integer,
    compressed_size_bytes bigint,
    uncompressed_size_bytes bigint,
    packages_count integer,
    vulnerabilities_high_count integer DEFAULT 0,
    vulnerabilities_medium_count integer DEFAULT 0,
    vulnerabilities_low_count integer DEFAULT 0,
    entrypoint text,
    cmd text,
    env_vars text,
    working_dir character varying(500),
    labels text,
    last_scanned_at timestamp with time zone,
    scan_tool character varying(100),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: image_sbom; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.image_sbom (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    image_id uuid NOT NULL,
    sbom_format character varying(50) DEFAULT 'spdx'::character varying NOT NULL,
    sbom_version character varying(20),
    sbom_content text NOT NULL,
    generated_by_tool character varying(100),
    tool_version character varying(50),
    sbom_checksum character varying(64),
    scan_timestamp timestamp with time zone,
    scan_duration_seconds integer,
    status character varying(50) DEFAULT 'valid'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: image_vulnerability_scans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.image_vulnerability_scans (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    image_id uuid NOT NULL,
    build_id uuid,
    scan_tool character varying(100) NOT NULL,
    tool_version character varying(50),
    scan_status character varying(50) DEFAULT 'in_progress'::character varying NOT NULL,
    started_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    completed_at timestamp with time zone,
    duration_seconds integer,
    vulnerabilities_critical integer DEFAULT 0,
    vulnerabilities_high integer DEFAULT 0,
    vulnerabilities_medium integer DEFAULT 0,
    vulnerabilities_low integer DEFAULT 0,
    vulnerabilities_negligible integer DEFAULT 0,
    vulnerabilities_unknown integer DEFAULT 0,
    pass_fail_result character varying(20),
    compliance_check_passed boolean,
    scan_report_location character varying(500),
    scan_report_json text,
    error_message text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: images; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.images (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    project_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    tag character varying(100) NOT NULL,
    full_image_name character varying(500) NOT NULL,
    docker_digest character varying(255),
    built_from_commit character varying(40),
    built_from_branch character varying(100),
    built_at timestamp with time zone,
    compressed_size_bytes bigint,
    uncompressed_size_bytes bigint,
    status character varying(50) DEFAULT 'available'::character varying,
    is_signed boolean DEFAULT false,
    signature_key_id character varying(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp with time zone
);


--
-- Name: incident_timelines; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.incident_timelines (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    incident_id uuid NOT NULL,
    event_type character varying(50) NOT NULL,
    event_title character varying(255),
    event_description text,
    actor_user_id uuid,
    actor_name character varying(255),
    event_timestamp timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    impact_assessment text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: incidents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.incidents (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    title character varying(255) NOT NULL,
    description text,
    severity character varying(50) DEFAULT 'medium'::character varying NOT NULL,
    incident_type character varying(50),
    affected_systems text,
    affected_users_count integer,
    discovered_at timestamp with time zone,
    reported_by_user_id uuid,
    reported_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at timestamp with time zone,
    acknowledged_by_user_id uuid,
    investigating_team character varying(255),
    root_cause text,
    status character varying(50) DEFAULT 'open'::character varying NOT NULL,
    resolved_at timestamp with time zone,
    resolution_notes text,
    lessons_learned text,
    system_downtime_minutes integer,
    data_affected boolean DEFAULT false,
    users_impacted integer,
    requires_disclosure boolean DEFAULT false,
    disclosure_date date,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: infrastructure_nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.infrastructure_nodes (
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    status character varying(50) DEFAULT 'ready'::character varying NOT NULL,
    total_cpu_cores numeric(10,2) NOT NULL,
    total_memory_gb numeric(10,2) NOT NULL,
    total_disk_gb numeric(10,2) NOT NULL,
    last_heartbeat timestamp with time zone,
    maintenance_mode boolean DEFAULT false,
    labels jsonb,
    metadata jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE infrastructure_nodes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.infrastructure_nodes IS 'Build nodes/runners managed by the system';


--
-- Name: infrastructure_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.infrastructure_providers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    is_global boolean DEFAULT false NOT NULL,
    provider_type character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    display_name character varying(255) NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    capabilities jsonb DEFAULT '[]'::jsonb,
    last_health_check timestamp with time zone,
    health_status character varying(50),
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT infrastructure_providers_provider_type_check CHECK (((provider_type)::text = ANY ((ARRAY['kubernetes'::character varying, 'aws-eks'::character varying, 'gcp-gke'::character varying, 'azure-aks'::character varying, 'oci-oke'::character varying, 'vmware-vks'::character varying, 'openshift'::character varying, 'rancher'::character varying, 'build_nodes'::character varying])::text[]))),
    CONSTRAINT infrastructure_providers_status_check CHECK (((status)::text = ANY ((ARRAY['online'::character varying, 'offline'::character varying, 'maintenance'::character varying, 'pending'::character varying])::text[])))
);


--
-- Name: infrastructure_usage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.infrastructure_usage (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    build_execution_id uuid,
    infrastructure_type character varying(50) NOT NULL,
    provider_type character varying(50),
    cluster_name character varying(255),
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    duration_seconds integer,
    cost_cents integer DEFAULT 0,
    resource_usage jsonb,
    success boolean,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT infrastructure_usage_infrastructure_type_check CHECK (((infrastructure_type)::text = ANY ((ARRAY['kubernetes'::character varying, 'build_nodes'::character varying])::text[])))
);


--
-- Name: node_resource_usage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.node_resource_usage (
    id uuid NOT NULL,
    node_id uuid NOT NULL,
    used_cpu_cores numeric(10,2) DEFAULT 0 NOT NULL,
    used_memory_gb numeric(10,2) DEFAULT 0 NOT NULL,
    used_disk_gb numeric(10,2) DEFAULT 0 NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: notification_channels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_channels (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    channel_type character varying(50) NOT NULL,
    config_json text NOT NULL,
    api_key_encrypted character varying(500),
    api_secret_encrypted character varying(500),
    status character varying(50) DEFAULT 'active'::character varying,
    last_verified_at timestamp with time zone,
    verification_status character varying(50),
    available_for_alerts boolean DEFAULT true,
    available_for_notifications boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: notification_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    company_id uuid,
    template_type character varying(50) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    subject_template text NOT NULL,
    body_template text,
    html_template text,
    is_default boolean DEFAULT false,
    enabled boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    title character varying(255),
    message text,
    notification_type character varying(50),
    related_resource_type character varying(100),
    related_resource_id uuid,
    is_read boolean DEFAULT false,
    read_at timestamp with time zone,
    channel character varying(50),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: org_unit_access; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.org_unit_access (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_unit_id uuid NOT NULL,
    user_id uuid NOT NULL,
    access_level character varying(50) DEFAULT 'member'::character varying NOT NULL,
    granted_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    granted_by uuid
);


--
-- Name: org_units; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.org_units (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(100) NOT NULL,
    description text,
    parent_org_unit_id uuid,
    department_name character varying(100),
    manager_email character varying(255),
    team_size integer,
    resource_quota_cpu_cores numeric(10,2) DEFAULT 10,
    resource_quota_memory_gb numeric(10,2) DEFAULT 20,
    resource_quota_storage_gb numeric(10,2) DEFAULT 100,
    resource_quota_concurrent_builds integer DEFAULT 5,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: package_vulnerabilities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.package_vulnerabilities (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    package_name character varying(255) NOT NULL,
    package_type character varying(50) NOT NULL,
    package_version character varying(100) NOT NULL,
    cve_id character varying(20) NOT NULL,
    vulnerable_version_range character varying(100),
    patched_version character varying(100),
    source_database character varying(100),
    discovered_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.permissions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    resource character varying(100) NOT NULL,
    action character varying(50) NOT NULL,
    description text,
    category character varying(50),
    is_system_permission boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: project_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_members (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    project_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role_id uuid,
    assigned_by_user_id uuid,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: projects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.projects (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(100) NOT NULL,
    description text,
    git_repository_url character varying(500),
    git_branch character varying(100) DEFAULT 'main'::character varying,
    dockerfile_path character varying(255) DEFAULT 'Dockerfile'::character varying,
    build_context_path character varying(255) DEFAULT '.'::character varying,
    image_name_prefix character varying(100),
    enable_cache boolean DEFAULT true,
    build_timeout_minutes integer DEFAULT 30,
    status character varying(50) DEFAULT 'active'::character varying,
    visibility character varying(50) DEFAULT 'private'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp with time zone,
    created_by uuid,
    repository_auth_id uuid,
    git_provider_key character varying(50) DEFAULT 'generic'::character varying NOT NULL
);


--
-- Name: COLUMN projects.repository_auth_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.projects.repository_auth_id IS 'Reference to the active repository authentication configuration';


--
-- Name: COLUMN projects.git_provider_key; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.projects.git_provider_key IS 'Selected Git provider key for repository operations';


--
-- Name: provider_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.provider_permissions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    provider_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    permission character varying(100) NOT NULL,
    granted_by uuid NOT NULL,
    granted_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone
);


--
-- Name: rbac_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rbac_roles (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid,
    name character varying(100) NOT NULL,
    description text,
    is_system boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    version integer DEFAULT 1
);


--
-- Name: repository_auth; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repository_auth (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    project_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    auth_type character varying(50) NOT NULL,
    credential_data bytea NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT repository_auth_auth_type_check CHECK (((auth_type)::text = ANY ((ARRAY['ssh_key'::character varying, 'token'::character varying, 'basic_auth'::character varying, 'oauth'::character varying])::text[])))
);


--
-- Name: TABLE repository_auth; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.repository_auth IS 'Stores encrypted authentication credentials for private source code repositories';


--
-- Name: COLUMN repository_auth.credential_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repository_auth.credential_data IS 'AES-256-GCM encrypted JSON containing authentication credentials';


--
-- Name: resource_quotas; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.resource_quotas (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    org_unit_id uuid,
    scope character varying(50) DEFAULT 'company'::character varying NOT NULL,
    cpu_cores_limit numeric(10,2),
    memory_gb_limit numeric(10,2),
    storage_gb_limit numeric(10,2),
    artifact_storage_gb_limit numeric(10,2),
    concurrent_builds_limit integer,
    monthly_builds_limit integer,
    images_per_repository_limit integer,
    repositories_limit integer,
    api_calls_per_minute_limit integer,
    api_calls_per_month_limit integer,
    concurrent_deployments_limit integer,
    enforce_hard_limit boolean DEFAULT true,
    status character varying(50) DEFAULT 'active'::character varying,
    effective_from date,
    effective_until date,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: role_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_permissions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    role_id uuid NOT NULL,
    permission_id uuid NOT NULL,
    granted_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: sbom_packages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sbom_packages (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    image_sbom_id uuid NOT NULL,
    image_id uuid NOT NULL,
    package_name character varying(255) NOT NULL,
    package_version character varying(100),
    package_type character varying(50),
    package_url character varying(500),
    homepage_url character varying(500),
    license_name character varying(255),
    license_spdx_id character varying(50),
    package_path character varying(500),
    known_vulnerabilities_count integer DEFAULT 0,
    critical_vulnerabilities_count integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: security_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.security_policies (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    policy_type character varying(50) NOT NULL,
    is_enabled boolean DEFAULT true,
    enforce_on_deployment boolean DEFAULT true,
    policy_config text,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: system_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_configs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid,
    config_type character varying(50) NOT NULL,
    config_key character varying(255) NOT NULL,
    config_value jsonb NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    description text,
    is_default boolean DEFAULT false NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    version integer DEFAULT 1 NOT NULL,
    CONSTRAINT system_configs_config_type_check CHECK (((config_type)::text = ANY ((ARRAY['security'::character varying, 'build'::character varying, 'ldap'::character varying, 'smtp'::character varying, 'general'::character varying, 'rate_limit'::character varying, 'feature_flags'::character varying, 'tool_settings'::character varying, 'external_services'::character varying, 'messaging'::character varying])::text[]))),
    CONSTRAINT system_configs_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'inactive'::character varying, 'testing'::character varying])::text[])))
);


--
-- Name: tenant_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_groups (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(100) NOT NULL,
    description text,
    role_type character varying(50) NOT NULL,
    is_system_group boolean DEFAULT false,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenants (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_code character varying(8) NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(100) NOT NULL,
    description text,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: usage_tracking; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usage_tracking (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    org_unit_id uuid,
    usage_date date NOT NULL,
    usage_month character varying(7) NOT NULL,
    cpu_core_hours numeric(10,2) DEFAULT 0,
    memory_gb_hours numeric(10,2) DEFAULT 0,
    storage_gb_days numeric(10,2) DEFAULT 0,
    builds_executed integer DEFAULT 0,
    builds_succeeded integer DEFAULT 0,
    builds_failed integer DEFAULT 0,
    total_build_minutes integer DEFAULT 0,
    deployments_executed integer DEFAULT 0,
    deployments_succeeded integer DEFAULT 0,
    deployments_failed integer DEFAULT 0,
    images_created integer DEFAULT 0,
    images_total_size_gb numeric(10,2) DEFAULT 0,
    api_calls_count bigint DEFAULT 0,
    api_errors_count bigint DEFAULT 0,
    data_transferred_gb numeric(10,2) DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: user_invitations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_invitations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email character varying(255) NOT NULL,
    tenant_id uuid NOT NULL,
    role_id uuid,
    invite_token character varying(255) NOT NULL,
    invited_by_id uuid NOT NULL,
    status character varying(50) DEFAULT 'pending'::character varying,
    accepted_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    message text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: user_role_assignments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_role_assignments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    tenant_id uuid,
    assigned_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    assigned_by_user_id uuid,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: user_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_sessions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    tenant_id uuid,
    access_token character varying(500),
    refresh_token character varying(500),
    ip_address character varying(45),
    user_agent character varying(500),
    expires_at timestamp with time zone,
    refreshed_at timestamp with time zone,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email character varying(255) NOT NULL,
    first_name character varying(100),
    last_name character varying(100),
    phone character varying(20),
    password_hash character varying(255),
    is_ldap_user boolean DEFAULT false,
    ldap_username character varying(255),
    auth_method character varying(50) DEFAULT 'credentials'::character varying,
    mfa_enabled boolean DEFAULT false,
    mfa_type character varying(50),
    mfa_secret character varying(255),
    backup_codes text,
    status character varying(50) DEFAULT 'active'::character varying,
    email_verified boolean DEFAULT false,
    email_verified_at timestamp with time zone,
    profile_picture_url character varying(500),
    timezone character varying(50),
    preferred_language character varying(10),
    last_login_at timestamp with time zone,
    password_changed_at timestamp with time zone,
    failed_login_count integer DEFAULT 0,
    locked_until timestamp with time zone,
    version integer DEFAULT 1,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp with time zone,
    tenant_id uuid
);


--
-- Name: v_build_analytics; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_build_analytics AS
 SELECT count(*) AS total_builds,
    COALESCE(sum(
        CASE
            WHEN ((status)::text = 'in_progress'::text) THEN 1
            ELSE 0
        END), (0)::bigint) AS running_builds,
    COALESCE(sum(
        CASE
            WHEN ((status)::text = 'success'::text) THEN 1
            ELSE 0
        END), (0)::bigint) AS completed_builds,
    COALESCE(sum(
        CASE
            WHEN ((status)::text = 'failed'::text) THEN 1
            ELSE 0
        END), (0)::bigint) AS failed_builds,
    COALESCE(round((((sum(
        CASE
            WHEN ((status)::text = 'success'::text) THEN 1
            ELSE 0
        END))::numeric / (NULLIF(count(*), 0))::numeric) * (100)::numeric), 2), (0)::numeric) AS success_rate,
    COALESCE((round(avg(EXTRACT(epoch FROM (completed_at - started_at))), 0))::integer, 0) AS average_duration_seconds,
    ( SELECT count(*) AS count
           FROM public.builds builds_1
          WHERE ((builds_1.status)::text = 'queued'::text)) AS queue_depth,
    CURRENT_TIMESTAMP AS last_updated
   FROM public.builds
  WHERE ((status)::text = ANY ((ARRAY['queued'::character varying, 'in_progress'::character varying, 'success'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[]));


--
-- Name: v_build_failure_rate_by_project; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_build_failure_rate_by_project AS
 SELECT p.id,
    p.name AS project_name,
    count(*) AS total_builds,
    sum(
        CASE
            WHEN ((b.status)::text = 'failed'::text) THEN 1
            ELSE 0
        END) AS failed_builds,
    round((((sum(
        CASE
            WHEN ((b.status)::text = 'failed'::text) THEN 1
            ELSE 0
        END))::numeric / (NULLIF(count(*), 0))::numeric) * (100)::numeric), 2) AS failure_rate
   FROM (public.projects p
     LEFT JOIN public.builds b ON ((p.id = b.project_id)))
  WHERE ((b.created_at >= (CURRENT_DATE - '30 days'::interval)) AND ((b.status)::text = ANY ((ARRAY['success'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[])))
  GROUP BY p.id, p.name
  ORDER BY (round((((sum(
        CASE
            WHEN ((b.status)::text = 'failed'::text) THEN 1
            ELSE 0
        END))::numeric / (NULLIF(count(*), 0))::numeric) * (100)::numeric), 2)) DESC;


--
-- Name: v_build_failure_reasons; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_build_failure_reasons AS
 SELECT COALESCE(SUBSTRING(error_message FROM 1 FOR 50), 'Unknown Error'::text) AS failure_reason,
    count(*) AS failure_count,
    round((((count(*))::numeric / (( SELECT count(*) AS count
           FROM public.builds builds_1
          WHERE ((builds_1.status)::text = 'failed'::text)))::numeric) * (100)::numeric), 2) AS percentage
   FROM public.builds
  WHERE (((status)::text = 'failed'::text) AND (completed_at >= (CURRENT_DATE - '30 days'::interval)))
  GROUP BY (SUBSTRING(error_message FROM 1 FOR 50))
  ORDER BY (count(*)) DESC
 LIMIT 20;


--
-- Name: v_build_failures; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_build_failures AS
 SELECT id,
    project_id,
    status,
    (EXTRACT(epoch FROM (completed_at - started_at)))::integer AS duration_seconds,
    error_message,
    created_at,
    completed_at
   FROM public.builds
  WHERE (((status)::text = ANY ((ARRAY['failed'::character varying, 'cancelled'::character varying])::text[])) AND (completed_at IS NOT NULL))
  ORDER BY completed_at DESC;


--
-- Name: v_build_performance_trends; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_build_performance_trends AS
 SELECT date(created_at) AS trend_date,
    count(*) AS build_count,
    (round(avg(EXTRACT(epoch FROM (completed_at - started_at))), 0))::integer AS average_duration_seconds,
    round((((sum(
        CASE
            WHEN ((status)::text = 'success'::text) THEN 1
            ELSE 0
        END))::numeric / (NULLIF(count(*), 0))::numeric) * (100)::numeric), 2) AS success_rate,
    (round(avg(EXTRACT(epoch FROM (started_at - created_at))), 0))::integer AS average_queue_time_seconds
   FROM public.builds
  WHERE ((created_at >= (CURRENT_DATE - '7 days'::interval)) AND ((status)::text = ANY ((ARRAY['success'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[])))
  GROUP BY (date(created_at))
  ORDER BY (date(created_at)) DESC;


--
-- Name: v_build_slowest_builds; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_build_slowest_builds AS
 SELECT b.id,
    b.project_id,
    p.name AS project_name,
    (EXTRACT(epoch FROM (b.completed_at - b.started_at)))::integer AS duration_seconds,
    b.status,
    b.created_at,
    b.completed_at
   FROM (public.builds b
     LEFT JOIN public.projects p ON ((b.project_id = p.id)))
  WHERE (((b.status)::text = ANY ((ARRAY['success'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[])) AND (b.completed_at IS NOT NULL))
  ORDER BY ((EXTRACT(epoch FROM (b.completed_at - b.started_at)))::integer) DESC
 LIMIT 100;


--
-- Name: v_infrastructure_health; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_infrastructure_health AS
 SELECT (count(*))::integer AS total_nodes,
    (COALESCE(sum(
        CASE
            WHEN (((status)::text = 'ready'::text) AND (maintenance_mode = false)) THEN 1
            ELSE 0
        END), (0)::bigint))::integer AS healthy_nodes,
    (COALESCE(sum(
        CASE
            WHEN ((status)::text = 'offline'::text) THEN 1
            ELSE 0
        END), (0)::bigint))::integer AS offline_nodes,
    (COALESCE(sum(
        CASE
            WHEN (maintenance_mode = true) THEN 1
            ELSE 0
        END), (0)::bigint))::integer AS maintenance_nodes,
    (sum(total_cpu_cores))::numeric(10,2) AS total_cpu_capacity,
    (sum(total_memory_gb))::numeric(10,2) AS total_memory_capacity_gb,
    (sum(total_disk_gb))::numeric(10,2) AS total_disk_capacity_gb,
    (0)::numeric(10,2) AS used_cpu_cores,
    (0)::numeric(10,2) AS used_memory_gb,
    (0)::numeric(10,2) AS used_disk_gb,
    (0)::numeric(5,2) AS avg_cpu_usage_percent,
    (0)::numeric(5,2) AS avg_memory_usage_percent,
    (0)::numeric(5,2) AS avg_disk_usage_percent
   FROM public.infrastructure_nodes;


--
-- Name: VIEW v_infrastructure_health; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON VIEW public.v_infrastructure_health IS 'System-wide infrastructure health summary';


--
-- Name: v_infrastructure_nodes; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.v_infrastructure_nodes AS
SELECT
    NULL::uuid AS id,
    NULL::character varying(255) AS name,
    NULL::character varying(50) AS status,
    NULL::timestamp with time zone AS last_heartbeat,
    NULL::integer AS heartbeat_age_seconds,
    NULL::numeric(10,2) AS total_cpu_capacity,
    NULL::numeric(10,2) AS total_memory_capacity_gb,
    NULL::numeric(10,2) AS total_disk_capacity_gb,
    NULL::boolean AS maintenance_mode,
    NULL::jsonb AS labels,
    NULL::integer AS current_builds,
    NULL::numeric(10,2) AS used_cpu_cores,
    NULL::numeric(10,2) AS used_memory_gb,
    NULL::numeric(10,2) AS available_cpu_cores,
    NULL::numeric(10,2) AS available_memory_gb,
    NULL::numeric(5,2) AS cpu_usage_percent,
    NULL::numeric(5,2) AS memory_usage_percent,
    NULL::timestamp with time zone AS created_at,
    NULL::timestamp with time zone AS updated_at;


--
-- Name: VIEW v_infrastructure_nodes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON VIEW public.v_infrastructure_nodes IS 'Real-time view of node status and resource usage';


--
-- Name: vulnerability_suppressions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vulnerability_suppressions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    company_id uuid NOT NULL,
    cve_id character varying(20) NOT NULL,
    package_name character varying(255),
    suppression_scope character varying(50) DEFAULT 'organization'::character varying NOT NULL,
    project_id uuid,
    image_id uuid,
    reason text NOT NULL,
    justification character varying(50) NOT NULL,
    suppressed_by_user_id uuid NOT NULL,
    suppressed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamp with time zone,
    approved_by_user_id uuid,
    approved_at timestamp with time zone,
    status character varying(50) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: workers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    worker_name character varying(255) NOT NULL,
    worker_type character varying(50) NOT NULL,
    capacity integer DEFAULT 4 NOT NULL,
    current_load integer DEFAULT 0 NOT NULL,
    status character varying(50) DEFAULT 'available'::character varying NOT NULL,
    last_heartbeat timestamp with time zone,
    consecutive_failures integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT non_negative_load CHECK ((current_load >= 0)),
    CONSTRAINT positive_capacity CHECK ((capacity > 0)),
    CONSTRAINT valid_load CHECK ((current_load <= capacity)),
    CONSTRAINT valid_status CHECK (((status)::text = ANY ((ARRAY['available'::character varying, 'busy'::character varying, 'offline'::character varying])::text[]))),
    CONSTRAINT valid_worker_type CHECK (((worker_type)::text = ANY ((ARRAY['docker'::character varying, 'kubernetes'::character varying, 'lambda'::character varying])::text[])))
);


--
-- Name: TABLE workers; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.workers IS 'Worker pool for distributed build execution';


--
-- Name: COLUMN workers.capacity; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.workers.capacity IS 'Maximum concurrent builds this worker can handle';


--
-- Name: COLUMN workers.current_load; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.workers.current_load IS 'Number of builds currently running on this worker';


--
-- Name: COLUMN workers.consecutive_failures; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.workers.consecutive_failures IS 'Count of consecutive failures for health monitoring';


--
-- Name: workflow_definitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_definitions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(100) NOT NULL,
    version integer NOT NULL,
    definition jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: workflow_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    instance_id uuid NOT NULL,
    step_id uuid,
    type character varying(50) NOT NULL,
    payload jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: workflow_instances; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_instances (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    definition_id uuid NOT NULL,
    tenant_id uuid,
    subject_type character varying(50) NOT NULL,
    subject_id uuid NOT NULL,
    status character varying(20) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: workflow_steps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_steps (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    instance_id uuid NOT NULL,
    step_key character varying(100) NOT NULL,
    payload jsonb,
    status character varying(20) NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_error text,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: api_keys api_keys_key_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash);


--
-- Name: api_keys api_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);


--
-- Name: approval_requests approval_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_pkey PRIMARY KEY (id);


--
-- Name: approval_workflows approval_workflows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_workflows
    ADD CONSTRAINT approval_workflows_pkey PRIMARY KEY (id);


--
-- Name: approval_workflows approval_workflows_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_workflows
    ADD CONSTRAINT approval_workflows_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: audit_event_types audit_event_types_event_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_event_types
    ADD CONSTRAINT audit_event_types_event_type_key UNIQUE (event_type);


--
-- Name: audit_event_types audit_event_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_event_types
    ADD CONSTRAINT audit_event_types_pkey PRIMARY KEY (id);


--
-- Name: audit_events audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_events
    ADD CONSTRAINT audit_events_pkey PRIMARY KEY (id);


--
-- Name: build_artifacts build_artifacts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_artifacts
    ADD CONSTRAINT build_artifacts_pkey PRIMARY KEY (id);


--
-- Name: build_configs build_configs_build_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_configs
    ADD CONSTRAINT build_configs_build_id_key UNIQUE (build_id);


--
-- Name: build_configs build_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_configs
    ADD CONSTRAINT build_configs_pkey PRIMARY KEY (id);


--
-- Name: build_execution_logs build_execution_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_execution_logs
    ADD CONSTRAINT build_execution_logs_pkey PRIMARY KEY (id);


--
-- Name: build_executions build_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_executions
    ADD CONSTRAINT build_executions_pkey PRIMARY KEY (id);


--
-- Name: build_history build_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_history
    ADD CONSTRAINT build_history_pkey PRIMARY KEY (id);


--
-- Name: build_metrics build_metrics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_metrics
    ADD CONSTRAINT build_metrics_pkey PRIMARY KEY (id);


--
-- Name: build_policies build_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_policies
    ADD CONSTRAINT build_policies_pkey PRIMARY KEY (id);


--
-- Name: build_steps build_steps_build_id_step_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_steps
    ADD CONSTRAINT build_steps_build_id_step_number_key UNIQUE (build_id, step_number);


--
-- Name: build_steps build_steps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_steps
    ADD CONSTRAINT build_steps_pkey PRIMARY KEY (id);


--
-- Name: build_triggers build_triggers_build_id_trigger_type_webhook_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_triggers
    ADD CONSTRAINT build_triggers_build_id_trigger_type_webhook_url_key UNIQUE (build_id, trigger_type, webhook_url);


--
-- Name: build_triggers build_triggers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_triggers
    ADD CONSTRAINT build_triggers_pkey PRIMARY KEY (id);


--
-- Name: builds builds_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.builds
    ADD CONSTRAINT builds_pkey PRIMARY KEY (id);


--
-- Name: builds builds_project_id_build_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.builds
    ADD CONSTRAINT builds_project_id_build_number_key UNIQUE (project_id, build_number);


--
-- Name: catalog_image_tags catalog_image_tags_catalog_image_id_tag_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_image_tags
    ADD CONSTRAINT catalog_image_tags_catalog_image_id_tag_key UNIQUE (catalog_image_id, tag);


--
-- Name: catalog_image_tags catalog_image_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_image_tags
    ADD CONSTRAINT catalog_image_tags_pkey PRIMARY KEY (id);


--
-- Name: catalog_image_versions catalog_image_versions_catalog_image_id_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_image_versions
    ADD CONSTRAINT catalog_image_versions_catalog_image_id_version_key UNIQUE (catalog_image_id, version);


--
-- Name: catalog_image_versions catalog_image_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_image_versions
    ADD CONSTRAINT catalog_image_versions_pkey PRIMARY KEY (id);


--
-- Name: catalog_images catalog_images_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_images
    ADD CONSTRAINT catalog_images_pkey PRIMARY KEY (id);


--
-- Name: catalog_images catalog_images_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_images
    ADD CONSTRAINT catalog_images_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: change_requests change_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.change_requests
    ADD CONSTRAINT change_requests_pkey PRIMARY KEY (id);


--
-- Name: companies companies_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.companies
    ADD CONSTRAINT companies_name_key UNIQUE (name);


--
-- Name: companies companies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.companies
    ADD CONSTRAINT companies_pkey PRIMARY KEY (id);


--
-- Name: compliance_assessments compliance_assessments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_assessments
    ADD CONSTRAINT compliance_assessments_pkey PRIMARY KEY (id);


--
-- Name: compliance_controls compliance_controls_compliance_framework_id_control_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_controls
    ADD CONSTRAINT compliance_controls_compliance_framework_id_control_code_key UNIQUE (compliance_framework_id, control_code);


--
-- Name: compliance_controls compliance_controls_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_controls
    ADD CONSTRAINT compliance_controls_pkey PRIMARY KEY (id);


--
-- Name: compliance_evidence compliance_evidence_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_evidence
    ADD CONSTRAINT compliance_evidence_pkey PRIMARY KEY (id);


--
-- Name: compliance_frameworks compliance_frameworks_company_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_frameworks
    ADD CONSTRAINT compliance_frameworks_company_id_name_key UNIQUE (company_id, name);


--
-- Name: compliance_frameworks compliance_frameworks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_frameworks
    ADD CONSTRAINT compliance_frameworks_pkey PRIMARY KEY (id);


--
-- Name: config_template_shares config_template_shares_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_template_shares
    ADD CONSTRAINT config_template_shares_pkey PRIMARY KEY (id);


--
-- Name: config_template_shares config_template_shares_template_id_shared_with_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_template_shares
    ADD CONSTRAINT config_template_shares_template_id_shared_with_user_id_key UNIQUE (template_id, shared_with_user_id);


--
-- Name: config_templates config_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_templates
    ADD CONSTRAINT config_templates_pkey PRIMARY KEY (id);


--
-- Name: config_templates config_templates_project_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_templates
    ADD CONSTRAINT config_templates_project_id_name_key UNIQUE (project_id, name);


--
-- Name: container_registries container_registries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_registries
    ADD CONSTRAINT container_registries_pkey PRIMARY KEY (id);


--
-- Name: container_registries container_registries_registry_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_registries
    ADD CONSTRAINT container_registries_registry_url_key UNIQUE (registry_url);


--
-- Name: container_repositories container_repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_repositories
    ADD CONSTRAINT container_repositories_pkey PRIMARY KEY (id);


--
-- Name: container_repositories container_repositories_registry_id_full_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_repositories
    ADD CONSTRAINT container_repositories_registry_id_full_name_key UNIQUE (registry_id, full_name);


--
-- Name: cve_database cve_database_cve_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cve_database
    ADD CONSTRAINT cve_database_cve_id_key UNIQUE (cve_id);


--
-- Name: cve_database cve_database_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cve_database
    ADD CONSTRAINT cve_database_pkey PRIMARY KEY (id);


--
-- Name: deployment_environments deployment_environments_company_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_environments
    ADD CONSTRAINT deployment_environments_company_id_name_key UNIQUE (company_id, name);


--
-- Name: deployment_environments deployment_environments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_environments
    ADD CONSTRAINT deployment_environments_pkey PRIMARY KEY (id);


--
-- Name: deployments deployments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployments
    ADD CONSTRAINT deployments_pkey PRIMARY KEY (id);


--
-- Name: email_queue email_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_queue
    ADD CONSTRAINT email_queue_pkey PRIMARY KEY (id);


--
-- Name: environment_access environment_access_deployment_environment_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment_access
    ADD CONSTRAINT environment_access_deployment_environment_id_user_id_key UNIQUE (deployment_environment_id, user_id);


--
-- Name: environment_access environment_access_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment_access
    ADD CONSTRAINT environment_access_pkey PRIMARY KEY (id);


--
-- Name: external_services external_services_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_name_key UNIQUE (name);


--
-- Name: external_services external_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_pkey PRIMARY KEY (id);


--
-- Name: external_tenants external_tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_tenants
    ADD CONSTRAINT external_tenants_pkey PRIMARY KEY (id);


--
-- Name: external_tenants external_tenants_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_tenants
    ADD CONSTRAINT external_tenants_slug_key UNIQUE (slug);


--
-- Name: external_tenants external_tenants_tenant_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_tenants
    ADD CONSTRAINT external_tenants_tenant_id_key UNIQUE (tenant_id);


--
-- Name: git_integration git_integration_git_repository_id_integration_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_integration
    ADD CONSTRAINT git_integration_git_repository_id_integration_type_key UNIQUE (git_repository_id, integration_type);


--
-- Name: git_integration git_integration_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_integration
    ADD CONSTRAINT git_integration_pkey PRIMARY KEY (id);


--
-- Name: git_providers git_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_providers
    ADD CONSTRAINT git_providers_pkey PRIMARY KEY (id);


--
-- Name: git_providers git_providers_provider_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_providers
    ADD CONSTRAINT git_providers_provider_key_key UNIQUE (provider_key);


--
-- Name: git_repositories git_repositories_company_id_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_repositories
    ADD CONSTRAINT git_repositories_company_id_url_key UNIQUE (company_id, url);


--
-- Name: git_repositories git_repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_repositories
    ADD CONSTRAINT git_repositories_pkey PRIMARY KEY (id);


--
-- Name: group_members group_members_group_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_members
    ADD CONSTRAINT group_members_group_id_user_id_key UNIQUE (group_id, user_id);


--
-- Name: group_members group_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_members
    ADD CONSTRAINT group_members_pkey PRIMARY KEY (id);


--
-- Name: image_layers image_layers_image_id_layer_digest_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_layers
    ADD CONSTRAINT image_layers_image_id_layer_digest_key UNIQUE (image_id, layer_digest);


--
-- Name: image_layers image_layers_image_id_layer_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_layers
    ADD CONSTRAINT image_layers_image_id_layer_number_key UNIQUE (image_id, layer_number);


--
-- Name: image_layers image_layers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_layers
    ADD CONSTRAINT image_layers_pkey PRIMARY KEY (id);


--
-- Name: image_metadata image_metadata_image_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_metadata
    ADD CONSTRAINT image_metadata_image_id_key UNIQUE (image_id);


--
-- Name: image_metadata image_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_metadata
    ADD CONSTRAINT image_metadata_pkey PRIMARY KEY (id);


--
-- Name: image_sbom image_sbom_image_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_sbom
    ADD CONSTRAINT image_sbom_image_id_key UNIQUE (image_id);


--
-- Name: image_sbom image_sbom_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_sbom
    ADD CONSTRAINT image_sbom_pkey PRIMARY KEY (id);


--
-- Name: image_vulnerability_scans image_vulnerability_scans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_vulnerability_scans
    ADD CONSTRAINT image_vulnerability_scans_pkey PRIMARY KEY (id);


--
-- Name: images images_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.images
    ADD CONSTRAINT images_pkey PRIMARY KEY (id);


--
-- Name: images images_project_id_name_tag_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.images
    ADD CONSTRAINT images_project_id_name_tag_key UNIQUE (project_id, name, tag);


--
-- Name: incident_timelines incident_timelines_incident_id_event_timestamp_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incident_timelines
    ADD CONSTRAINT incident_timelines_incident_id_event_timestamp_key UNIQUE (incident_id, event_timestamp);


--
-- Name: incident_timelines incident_timelines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incident_timelines
    ADD CONSTRAINT incident_timelines_pkey PRIMARY KEY (id);


--
-- Name: incidents incidents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidents
    ADD CONSTRAINT incidents_pkey PRIMARY KEY (id);


--
-- Name: infrastructure_nodes infrastructure_nodes_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_nodes
    ADD CONSTRAINT infrastructure_nodes_name_key UNIQUE (name);


--
-- Name: infrastructure_nodes infrastructure_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_nodes
    ADD CONSTRAINT infrastructure_nodes_pkey PRIMARY KEY (id);


--
-- Name: infrastructure_providers infrastructure_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_providers
    ADD CONSTRAINT infrastructure_providers_pkey PRIMARY KEY (id);


--
-- Name: infrastructure_usage infrastructure_usage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_usage
    ADD CONSTRAINT infrastructure_usage_pkey PRIMARY KEY (id);


--
-- Name: node_resource_usage node_resource_usage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_resource_usage
    ADD CONSTRAINT node_resource_usage_pkey PRIMARY KEY (id);


--
-- Name: notification_channels notification_channels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_channels
    ADD CONSTRAINT notification_channels_pkey PRIMARY KEY (id);


--
-- Name: notification_templates notification_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_templates
    ADD CONSTRAINT notification_templates_pkey PRIMARY KEY (id);


--
-- Name: notification_templates notification_templates_type_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_templates
    ADD CONSTRAINT notification_templates_type_unique UNIQUE (template_type);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: org_unit_access org_unit_access_org_unit_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_unit_access
    ADD CONSTRAINT org_unit_access_org_unit_id_user_id_key UNIQUE (org_unit_id, user_id);


--
-- Name: org_unit_access org_unit_access_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_unit_access
    ADD CONSTRAINT org_unit_access_pkey PRIMARY KEY (id);


--
-- Name: org_units org_units_company_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_units
    ADD CONSTRAINT org_units_company_id_name_key UNIQUE (company_id, name);


--
-- Name: org_units org_units_company_id_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_units
    ADD CONSTRAINT org_units_company_id_slug_key UNIQUE (company_id, slug);


--
-- Name: org_units org_units_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_units
    ADD CONSTRAINT org_units_pkey PRIMARY KEY (id);


--
-- Name: package_vulnerabilities package_vulnerabilities_package_name_package_type_package_v_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_vulnerabilities
    ADD CONSTRAINT package_vulnerabilities_package_name_package_type_package_v_key UNIQUE (package_name, package_type, package_version, cve_id);


--
-- Name: package_vulnerabilities package_vulnerabilities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_vulnerabilities
    ADD CONSTRAINT package_vulnerabilities_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_resource_action_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_resource_action_key UNIQUE (resource, action);


--
-- Name: project_members project_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_pkey PRIMARY KEY (id);


--
-- Name: project_members project_members_project_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_project_id_user_id_key UNIQUE (project_id, user_id);


--
-- Name: projects projects_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);


--
-- Name: projects projects_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: projects projects_tenant_id_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_tenant_id_slug_key UNIQUE (tenant_id, slug);


--
-- Name: provider_permissions provider_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provider_permissions
    ADD CONSTRAINT provider_permissions_pkey PRIMARY KEY (id);


--
-- Name: provider_permissions provider_permissions_provider_id_tenant_id_permission_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provider_permissions
    ADD CONSTRAINT provider_permissions_provider_id_tenant_id_permission_key UNIQUE (provider_id, tenant_id, permission);


--
-- Name: rbac_roles rbac_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rbac_roles
    ADD CONSTRAINT rbac_roles_pkey PRIMARY KEY (id);


--
-- Name: rbac_roles rbac_roles_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rbac_roles
    ADD CONSTRAINT rbac_roles_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: repository_auth repository_auth_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repository_auth
    ADD CONSTRAINT repository_auth_pkey PRIMARY KEY (id);


--
-- Name: repository_auth repository_auth_project_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repository_auth
    ADD CONSTRAINT repository_auth_project_id_name_key UNIQUE (project_id, name);


--
-- Name: resource_quotas resource_quotas_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_quotas
    ADD CONSTRAINT resource_quotas_pkey PRIMARY KEY (id);


--
-- Name: role_permissions role_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (id);


--
-- Name: role_permissions role_permissions_role_id_permission_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_role_id_permission_id_key UNIQUE (role_id, permission_id);


--
-- Name: sbom_packages sbom_packages_image_sbom_id_package_name_package_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sbom_packages
    ADD CONSTRAINT sbom_packages_image_sbom_id_package_name_package_version_key UNIQUE (image_sbom_id, package_name, package_version);


--
-- Name: sbom_packages sbom_packages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sbom_packages
    ADD CONSTRAINT sbom_packages_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: security_policies security_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.security_policies
    ADD CONSTRAINT security_policies_pkey PRIMARY KEY (id);


--
-- Name: security_policies security_policies_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.security_policies
    ADD CONSTRAINT security_policies_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: system_configs system_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_configs
    ADD CONSTRAINT system_configs_pkey PRIMARY KEY (id);


--
-- Name: system_configs system_configs_tenant_id_config_type_config_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_configs
    ADD CONSTRAINT system_configs_tenant_id_config_type_config_key_key UNIQUE (tenant_id, config_type, config_key);


--
-- Name: tenant_groups tenant_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_groups
    ADD CONSTRAINT tenant_groups_pkey PRIMARY KEY (id);


--
-- Name: tenant_groups tenant_groups_tenant_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_groups
    ADD CONSTRAINT tenant_groups_tenant_id_name_key UNIQUE (tenant_id, name);


--
-- Name: tenant_groups tenant_groups_tenant_id_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_groups
    ADD CONSTRAINT tenant_groups_tenant_id_slug_key UNIQUE (tenant_id, slug);


--
-- Name: tenants tenants_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_name_key UNIQUE (name);


--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);


--
-- Name: tenants tenants_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_slug_key UNIQUE (slug);


--
-- Name: tenants tenants_tenant_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_tenant_code_key UNIQUE (tenant_code);


--
-- Name: usage_tracking usage_tracking_company_id_usage_date_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_tracking
    ADD CONSTRAINT usage_tracking_company_id_usage_date_key UNIQUE (company_id, usage_date);


--
-- Name: usage_tracking usage_tracking_company_id_usage_month_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_tracking
    ADD CONSTRAINT usage_tracking_company_id_usage_month_key UNIQUE (company_id, usage_month);


--
-- Name: usage_tracking usage_tracking_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_tracking
    ADD CONSTRAINT usage_tracking_pkey PRIMARY KEY (id);


--
-- Name: user_invitations user_invitations_email_tenant_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_invitations
    ADD CONSTRAINT user_invitations_email_tenant_id_key UNIQUE (email, tenant_id);


--
-- Name: user_invitations user_invitations_invite_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_invitations
    ADD CONSTRAINT user_invitations_invite_token_key UNIQUE (invite_token);


--
-- Name: user_invitations user_invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_invitations
    ADD CONSTRAINT user_invitations_pkey PRIMARY KEY (id);


--
-- Name: user_role_assignments user_role_assignments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_role_assignments
    ADD CONSTRAINT user_role_assignments_pkey PRIMARY KEY (id);


--
-- Name: user_role_assignments user_role_assignments_user_id_role_id_tenant_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_role_assignments
    ADD CONSTRAINT user_role_assignments_user_id_role_id_tenant_id_key UNIQUE (user_id, role_id, tenant_id);


--
-- Name: user_sessions user_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: vulnerability_suppressions vulnerability_suppressions_cve_id_suppression_scope_project_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_cve_id_suppression_scope_project_key UNIQUE (cve_id, suppression_scope, project_id, image_id);


--
-- Name: vulnerability_suppressions vulnerability_suppressions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_pkey PRIMARY KEY (id);


--
-- Name: workers workers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workers
    ADD CONSTRAINT workers_pkey PRIMARY KEY (id);


--
-- Name: workflow_definitions workflow_definitions_name_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_definitions
    ADD CONSTRAINT workflow_definitions_name_version_key UNIQUE (name, version);


--
-- Name: workflow_definitions workflow_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_definitions
    ADD CONSTRAINT workflow_definitions_pkey PRIMARY KEY (id);


--
-- Name: workflow_events workflow_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_pkey PRIMARY KEY (id);


--
-- Name: workflow_instances workflow_instances_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_instances
    ADD CONSTRAINT workflow_instances_pkey PRIMARY KEY (id);


--
-- Name: workflow_steps workflow_steps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_steps
    ADD CONSTRAINT workflow_steps_pkey PRIMARY KEY (id);


--
-- Name: idx_api_keys_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_keys_expires_at ON public.api_keys USING btree (expires_at);


--
-- Name: idx_api_keys_key_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_keys_key_hash ON public.api_keys USING btree (key_hash);


--
-- Name: idx_api_keys_revoked_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_keys_revoked_at ON public.api_keys USING btree (revoked_at);


--
-- Name: idx_api_keys_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_keys_tenant_id ON public.api_keys USING btree (tenant_id);


--
-- Name: idx_approval_requests_requested_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_approval_requests_requested_at ON public.approval_requests USING btree (requested_at DESC);


--
-- Name: idx_approval_requests_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_approval_requests_status ON public.approval_requests USING btree (status);


--
-- Name: idx_approval_requests_workflow_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_approval_requests_workflow_id ON public.approval_requests USING btree (approval_workflow_id);


--
-- Name: idx_approval_workflows_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_approval_workflows_tenant_id ON public.approval_workflows USING btree (tenant_id);


--
-- Name: idx_audit_events_action; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_action ON public.audit_events USING btree (action);


--
-- Name: idx_audit_events_event_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_event_type ON public.audit_events USING btree (event_type);


--
-- Name: idx_audit_events_resource; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_resource ON public.audit_events USING btree (resource);


--
-- Name: idx_audit_events_severity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_severity ON public.audit_events USING btree (severity);


--
-- Name: idx_audit_events_tenant_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_tenant_event ON public.audit_events USING btree (tenant_id, event_type);


--
-- Name: idx_audit_events_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_tenant_id ON public.audit_events USING btree (tenant_id);


--
-- Name: idx_audit_events_tenant_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_tenant_timestamp ON public.audit_events USING btree (tenant_id, "timestamp" DESC);


--
-- Name: idx_audit_events_tenant_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_tenant_user ON public.audit_events USING btree (tenant_id, user_id);


--
-- Name: idx_audit_events_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_timestamp ON public.audit_events USING btree ("timestamp" DESC);


--
-- Name: idx_audit_events_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_events_user_id ON public.audit_events USING btree (user_id);


--
-- Name: idx_build_artifacts_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_artifacts_build_id ON public.build_artifacts USING btree (build_id);


--
-- Name: idx_build_artifacts_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_artifacts_image_id ON public.build_artifacts USING btree (image_id);


--
-- Name: idx_build_artifacts_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_artifacts_type ON public.build_artifacts USING btree (artifact_type);


--
-- Name: idx_build_configs_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_configs_build_id ON public.build_configs USING btree (build_id);


--
-- Name: idx_build_configs_build_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_configs_build_method ON public.build_configs USING btree (build_method);


--
-- Name: idx_build_execution_logs_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_execution_logs_execution_id ON public.build_execution_logs USING btree (execution_id);


--
-- Name: idx_build_execution_logs_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_execution_logs_timestamp ON public.build_execution_logs USING btree ("timestamp" DESC);


--
-- Name: idx_build_executions_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_executions_build_id ON public.build_executions USING btree (build_id);


--
-- Name: idx_build_executions_config_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_executions_config_id ON public.build_executions USING btree (config_id);


--
-- Name: idx_build_executions_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_executions_created_at ON public.build_executions USING btree (created_at DESC);


--
-- Name: idx_build_executions_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_executions_status ON public.build_executions USING btree (status);


--
-- Name: idx_build_history_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_build_id ON public.build_history USING btree (build_id);


--
-- Name: idx_build_history_build_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_build_method ON public.build_history USING btree (build_method);


--
-- Name: idx_build_history_completed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_completed_at ON public.build_history USING btree (completed_at DESC);


--
-- Name: idx_build_history_method_success; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_method_success ON public.build_history USING btree (build_method, success, completed_at DESC);


--
-- Name: idx_build_history_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_project_id ON public.build_history USING btree (project_id);


--
-- Name: idx_build_history_project_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_project_method ON public.build_history USING btree (project_id, build_method, completed_at DESC);


--
-- Name: idx_build_history_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_tenant_id ON public.build_history USING btree (tenant_id);


--
-- Name: idx_build_history_tenant_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_history_tenant_method ON public.build_history USING btree (tenant_id, build_method, completed_at DESC);


--
-- Name: idx_build_metrics_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_metrics_build_id ON public.build_metrics USING btree (build_id);


--
-- Name: idx_build_policies_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_policies_active ON public.build_policies USING btree (is_active);


--
-- Name: idx_build_policies_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_policies_key ON public.build_policies USING btree (policy_key);


--
-- Name: idx_build_policies_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_policies_tenant_id ON public.build_policies USING btree (tenant_id);


--
-- Name: idx_build_policies_tenant_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_build_policies_tenant_key ON public.build_policies USING btree (tenant_id, policy_key);


--
-- Name: idx_build_policies_tenant_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_policies_tenant_type ON public.build_policies USING btree (tenant_id, policy_type);


--
-- Name: idx_build_policies_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_policies_type ON public.build_policies USING btree (policy_type);


--
-- Name: idx_build_steps_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_steps_build_id ON public.build_steps USING btree (build_id);


--
-- Name: idx_build_steps_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_steps_status ON public.build_steps USING btree (status);


--
-- Name: idx_build_triggers_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_triggers_build_id ON public.build_triggers USING btree (build_id);


--
-- Name: idx_build_triggers_git_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_triggers_git_event ON public.build_triggers USING btree (git_repository_url) WHERE ((trigger_type)::text = 'git_event'::text);


--
-- Name: idx_build_triggers_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_triggers_project_id ON public.build_triggers USING btree (project_id);


--
-- Name: idx_build_triggers_scheduled_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_triggers_scheduled_active ON public.build_triggers USING btree (is_active, next_trigger_at) WHERE (((trigger_type)::text = 'schedule'::text) AND (is_active = true));


--
-- Name: idx_build_triggers_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_triggers_tenant_id ON public.build_triggers USING btree (tenant_id);


--
-- Name: idx_build_triggers_webhook_url; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_build_triggers_webhook_url ON public.build_triggers USING btree (webhook_url) WHERE ((trigger_type)::text = 'webhook'::text);


--
-- Name: idx_builds_assigned_node; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_assigned_node ON public.builds USING btree (assigned_node_id) WHERE ((status)::text = 'running'::text);


--
-- Name: idx_builds_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_created_at ON public.builds USING btree (created_at DESC);


--
-- Name: idx_builds_created_at_completed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_created_at_completed_at ON public.builds USING btree (created_at DESC, completed_at);


--
-- Name: idx_builds_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_image_id ON public.builds USING btree (image_id);


--
-- Name: idx_builds_infrastructure_provider_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_infrastructure_provider_id ON public.builds USING btree (infrastructure_provider_id);


--
-- Name: idx_builds_infrastructure_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_infrastructure_type ON public.builds USING btree (infrastructure_type);


--
-- Name: idx_builds_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_project_id ON public.builds USING btree (project_id);


--
-- Name: idx_builds_project_status_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_project_status_created ON public.builds USING btree (project_id, status, created_at DESC);


--
-- Name: idx_builds_selected_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_selected_at ON public.builds USING btree (selected_at);


--
-- Name: idx_builds_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_status ON public.builds USING btree (status);


--
-- Name: idx_builds_status_completed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_status_completed_at ON public.builds USING btree (status, completed_at DESC) WHERE ((status)::text = ANY ((ARRAY['failed'::character varying, 'cancelled'::character varying])::text[]));


--
-- Name: idx_builds_status_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_status_created_at ON public.builds USING btree (status, created_at DESC);


--
-- Name: idx_builds_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_tenant_id ON public.builds USING btree (tenant_id);


--
-- Name: idx_builds_tenant_id_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_tenant_id_status ON public.builds USING btree (tenant_id, status);


--
-- Name: idx_builds_tenant_status_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_builds_tenant_status_created ON public.builds USING btree (tenant_id, status, created_at DESC);


--
-- Name: idx_catalog_image_tags_catalog_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_image_tags_catalog_image_id ON public.catalog_image_tags USING btree (catalog_image_id);


--
-- Name: idx_catalog_image_tags_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_image_tags_category ON public.catalog_image_tags USING btree (category);


--
-- Name: idx_catalog_image_tags_tag; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_image_tags_tag ON public.catalog_image_tags USING btree (tag);


--
-- Name: idx_catalog_image_versions_catalog_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_image_versions_catalog_image_id ON public.catalog_image_versions USING btree (catalog_image_id);


--
-- Name: idx_catalog_image_versions_published_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_image_versions_published_at ON public.catalog_image_versions USING btree (published_at);


--
-- Name: idx_catalog_image_versions_tags; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_image_versions_tags ON public.catalog_image_versions USING gin (tags);


--
-- Name: idx_catalog_image_versions_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_image_versions_version ON public.catalog_image_versions USING btree (version);


--
-- Name: idx_catalog_images_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_created_at ON public.catalog_images USING btree (created_at);


--
-- Name: idx_catalog_images_metadata; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_metadata ON public.catalog_images USING gin (metadata);


--
-- Name: idx_catalog_images_search; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_search ON public.catalog_images USING gin (to_tsvector('english'::regconfig, (((name)::text || ' '::text) || COALESCE(description, ''::text))));


--
-- Name: idx_catalog_images_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_status ON public.catalog_images USING btree (status);


--
-- Name: idx_catalog_images_tags; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_tags ON public.catalog_images USING gin (tags);


--
-- Name: idx_catalog_images_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_tenant_id ON public.catalog_images USING btree (tenant_id);


--
-- Name: idx_catalog_images_updated_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_updated_at ON public.catalog_images USING btree (updated_at);


--
-- Name: idx_catalog_images_visibility; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_catalog_images_visibility ON public.catalog_images USING btree (visibility);


--
-- Name: idx_change_requests_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_change_requests_company_id ON public.change_requests USING btree (company_id);


--
-- Name: idx_change_requests_impact_level; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_change_requests_impact_level ON public.change_requests USING btree (impact_level);


--
-- Name: idx_change_requests_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_change_requests_status ON public.change_requests USING btree (status);


--
-- Name: idx_companies_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_companies_created_at ON public.companies USING btree (created_at);


--
-- Name: idx_companies_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_companies_status ON public.companies USING btree (status);


--
-- Name: idx_compliance_assessments_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_assessments_company_id ON public.compliance_assessments USING btree (company_id);


--
-- Name: idx_compliance_assessments_framework_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_assessments_framework_id ON public.compliance_assessments USING btree (compliance_framework_id);


--
-- Name: idx_compliance_assessments_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_assessments_status ON public.compliance_assessments USING btree (overall_status);


--
-- Name: idx_compliance_controls_framework_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_controls_framework_id ON public.compliance_controls USING btree (compliance_framework_id);


--
-- Name: idx_compliance_controls_org_unit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_controls_org_unit_id ON public.compliance_controls USING btree (responsible_org_unit_id);


--
-- Name: idx_compliance_controls_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_controls_status ON public.compliance_controls USING btree (status);


--
-- Name: idx_compliance_evidence_control_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_evidence_control_id ON public.compliance_evidence USING btree (compliance_control_id);


--
-- Name: idx_compliance_evidence_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_evidence_status ON public.compliance_evidence USING btree (status);


--
-- Name: idx_compliance_evidence_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_evidence_type ON public.compliance_evidence USING btree (evidence_type);


--
-- Name: idx_compliance_frameworks_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_frameworks_company_id ON public.compliance_frameworks USING btree (company_id);


--
-- Name: idx_compliance_frameworks_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_compliance_frameworks_type ON public.compliance_frameworks USING btree (framework_type);


--
-- Name: idx_config_template_shares_permissions; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_template_shares_permissions ON public.config_template_shares USING btree (template_id, can_use, can_edit);


--
-- Name: idx_config_template_shares_shared_with; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_template_shares_shared_with ON public.config_template_shares USING btree (shared_with_user_id);


--
-- Name: idx_config_template_shares_template_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_template_shares_template_id ON public.config_template_shares USING btree (template_id);


--
-- Name: idx_config_templates_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_templates_created_at ON public.config_templates USING btree (created_at DESC);


--
-- Name: idx_config_templates_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_templates_created_by ON public.config_templates USING btree (created_by_user_id);


--
-- Name: idx_config_templates_is_public; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_templates_is_public ON public.config_templates USING btree (is_public) WHERE (is_public = true);


--
-- Name: idx_config_templates_is_shared; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_templates_is_shared ON public.config_templates USING btree (is_shared) WHERE (is_shared = true);


--
-- Name: idx_config_templates_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_templates_method ON public.config_templates USING btree (method);


--
-- Name: idx_config_templates_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_config_templates_project_id ON public.config_templates USING btree (project_id);


--
-- Name: idx_container_registries_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_container_registries_company_id ON public.container_registries USING btree (company_id);


--
-- Name: idx_container_registries_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_container_registries_status ON public.container_registries USING btree (status);


--
-- Name: idx_container_registries_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_container_registries_type ON public.container_registries USING btree (registry_type);


--
-- Name: idx_container_repositories_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_container_repositories_company_id ON public.container_repositories USING btree (company_id);


--
-- Name: idx_container_repositories_org_unit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_container_repositories_org_unit_id ON public.container_repositories USING btree (owned_by_org_unit_id);


--
-- Name: idx_container_repositories_registry_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_container_repositories_registry_id ON public.container_repositories USING btree (registry_id);


--
-- Name: idx_cve_database_cve_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cve_database_cve_id ON public.cve_database USING btree (cve_id);


--
-- Name: idx_cve_database_published_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cve_database_published_date ON public.cve_database USING btree (published_date DESC);


--
-- Name: idx_cve_database_severity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cve_database_severity ON public.cve_database USING btree (cvss_v3_severity);


--
-- Name: idx_deployment_environments_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deployment_environments_company_id ON public.deployment_environments USING btree (company_id);


--
-- Name: idx_deployment_environments_org_unit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deployment_environments_org_unit_id ON public.deployment_environments USING btree (owned_by_org_unit_id);


--
-- Name: idx_deployments_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deployments_company_id ON public.deployments USING btree (company_id);


--
-- Name: idx_deployments_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deployments_created_at ON public.deployments USING btree (created_at DESC);


--
-- Name: idx_deployments_environment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deployments_environment_id ON public.deployments USING btree (deployment_environment_id);


--
-- Name: idx_deployments_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deployments_image_id ON public.deployments USING btree (image_id);


--
-- Name: idx_deployments_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deployments_status ON public.deployments USING btree (status);


--
-- Name: idx_email_queue_cc_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_queue_cc_email ON public.email_queue USING btree (cc_email) WHERE (cc_email IS NOT NULL);


--
-- Name: idx_email_queue_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_queue_created ON public.email_queue USING btree (created_at DESC);


--
-- Name: idx_email_queue_next_retry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_queue_next_retry ON public.email_queue USING btree (next_retry_at) WHERE ((status)::text = 'failed'::text);


--
-- Name: idx_email_queue_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_queue_status ON public.email_queue USING btree (status) WHERE ((status)::text = ANY ((ARRAY['pending'::character varying, 'processing'::character varying])::text[]));


--
-- Name: idx_email_queue_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_queue_tenant ON public.email_queue USING btree (tenant_id);


--
-- Name: idx_environment_access_environment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_environment_access_environment_id ON public.environment_access USING btree (deployment_environment_id);


--
-- Name: idx_environment_access_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_environment_access_expires_at ON public.environment_access USING btree (expires_at);


--
-- Name: idx_environment_access_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_environment_access_user_id ON public.environment_access USING btree (user_id);


--
-- Name: idx_external_services_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_services_created_at ON public.external_services USING btree (created_at DESC);


--
-- Name: idx_external_services_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_services_enabled ON public.external_services USING btree (enabled);


--
-- Name: idx_external_services_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_services_name ON public.external_services USING btree (name);


--
-- Name: idx_external_tenants_slug; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_tenants_slug ON public.external_tenants USING btree (slug);


--
-- Name: idx_external_tenants_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_tenants_tenant_id ON public.external_tenants USING btree (tenant_id);


--
-- Name: idx_git_integration_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_git_integration_company_id ON public.git_integration USING btree (company_id);


--
-- Name: idx_git_integration_is_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_git_integration_is_enabled ON public.git_integration USING btree (is_enabled);


--
-- Name: idx_git_integration_repository_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_git_integration_repository_id ON public.git_integration USING btree (git_repository_id);


--
-- Name: idx_git_providers_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_git_providers_active ON public.git_providers USING btree (is_active);


--
-- Name: idx_git_repositories_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_git_repositories_company_id ON public.git_repositories USING btree (company_id);


--
-- Name: idx_git_repositories_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_git_repositories_provider ON public.git_repositories USING btree (provider);


--
-- Name: idx_group_members_group_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_members_group_id ON public.group_members USING btree (group_id);


--
-- Name: idx_group_members_is_admin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_members_is_admin ON public.group_members USING btree (is_group_admin);


--
-- Name: idx_group_members_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_members_user_id ON public.group_members USING btree (user_id);


--
-- Name: idx_image_layers_digest; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_layers_digest ON public.image_layers USING btree (layer_digest);


--
-- Name: idx_image_layers_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_layers_image_id ON public.image_layers USING btree (image_id);


--
-- Name: idx_image_metadata_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_metadata_image_id ON public.image_metadata USING btree (image_id);


--
-- Name: idx_image_metadata_vulnerabilities; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_metadata_vulnerabilities ON public.image_metadata USING btree (vulnerabilities_high_count, vulnerabilities_medium_count);


--
-- Name: idx_image_sbom_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_sbom_image_id ON public.image_sbom USING btree (image_id);


--
-- Name: idx_image_vulnerability_scans_build_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_vulnerability_scans_build_id ON public.image_vulnerability_scans USING btree (build_id);


--
-- Name: idx_image_vulnerability_scans_completed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_vulnerability_scans_completed_at ON public.image_vulnerability_scans USING btree (completed_at DESC);


--
-- Name: idx_image_vulnerability_scans_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_image_vulnerability_scans_image_id ON public.image_vulnerability_scans USING btree (image_id);


--
-- Name: idx_images_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_images_created_at ON public.images USING btree (created_at DESC);


--
-- Name: idx_images_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_images_project_id ON public.images USING btree (project_id);


--
-- Name: idx_images_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_images_status ON public.images USING btree (status);


--
-- Name: idx_incident_timelines_event_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_incident_timelines_event_type ON public.incident_timelines USING btree (event_type);


--
-- Name: idx_incident_timelines_incident_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_incident_timelines_incident_id ON public.incident_timelines USING btree (incident_id);


--
-- Name: idx_incidents_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_incidents_company_id ON public.incidents USING btree (company_id);


--
-- Name: idx_incidents_reported_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_incidents_reported_at ON public.incidents USING btree (reported_at DESC);


--
-- Name: idx_incidents_severity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_incidents_severity ON public.incidents USING btree (severity);


--
-- Name: idx_incidents_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_incidents_status ON public.incidents USING btree (status);


--
-- Name: idx_infrastructure_nodes_last_heartbeat; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_infrastructure_nodes_last_heartbeat ON public.infrastructure_nodes USING btree (last_heartbeat DESC);


--
-- Name: idx_infrastructure_nodes_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_infrastructure_nodes_status ON public.infrastructure_nodes USING btree (status);


--
-- Name: idx_infrastructure_providers_is_global; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_infrastructure_providers_is_global ON public.infrastructure_providers USING btree (is_global);


--
-- Name: idx_infrastructure_usage_build; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_infrastructure_usage_build ON public.infrastructure_usage USING btree (build_execution_id);


--
-- Name: idx_infrastructure_usage_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_infrastructure_usage_tenant ON public.infrastructure_usage USING btree (tenant_id);


--
-- Name: idx_infrastructure_usage_type_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_infrastructure_usage_type_time ON public.infrastructure_usage USING btree (infrastructure_type, start_time);


--
-- Name: idx_node_resource_usage_node_id_recorded; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_node_resource_usage_node_id_recorded ON public.node_resource_usage USING btree (node_id, recorded_at DESC);


--
-- Name: idx_notification_channels_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_channels_company_id ON public.notification_channels USING btree (company_id);


--
-- Name: idx_notification_channels_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_channels_status ON public.notification_channels USING btree (status);


--
-- Name: idx_notification_channels_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_channels_type ON public.notification_channels USING btree (channel_type);


--
-- Name: idx_notification_templates_company; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_templates_company ON public.notification_templates USING btree (company_id);


--
-- Name: idx_notification_templates_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_templates_enabled ON public.notification_templates USING btree (enabled);


--
-- Name: idx_notification_templates_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_templates_type ON public.notification_templates USING btree (template_type);


--
-- Name: idx_notifications_is_read; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_is_read ON public.notifications USING btree (is_read);


--
-- Name: idx_notifications_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_tenant_id ON public.notifications USING btree (tenant_id);


--
-- Name: idx_notifications_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_id ON public.notifications USING btree (user_id);


--
-- Name: idx_org_unit_access_level; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_org_unit_access_level ON public.org_unit_access USING btree (access_level);


--
-- Name: idx_org_unit_access_org_unit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_org_unit_access_org_unit_id ON public.org_unit_access USING btree (org_unit_id);


--
-- Name: idx_org_unit_access_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_org_unit_access_user_id ON public.org_unit_access USING btree (user_id);


--
-- Name: idx_org_units_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_org_units_company_id ON public.org_units USING btree (company_id);


--
-- Name: idx_org_units_parent_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_org_units_parent_id ON public.org_units USING btree (parent_org_unit_id);


--
-- Name: idx_org_units_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_org_units_status ON public.org_units USING btree (status);


--
-- Name: idx_package_vulnerabilities_cve_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_package_vulnerabilities_cve_id ON public.package_vulnerabilities USING btree (cve_id);


--
-- Name: idx_package_vulnerabilities_package; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_package_vulnerabilities_package ON public.package_vulnerabilities USING btree (package_name, package_type);


--
-- Name: idx_permissions_action; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_permissions_action ON public.permissions USING btree (action);


--
-- Name: idx_permissions_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_permissions_category ON public.permissions USING btree (category);


--
-- Name: idx_permissions_resource; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_permissions_resource ON public.permissions USING btree (resource);


--
-- Name: idx_project_members_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_project_members_project_id ON public.project_members USING btree (project_id);


--
-- Name: idx_project_members_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_project_members_role_id ON public.project_members USING btree (role_id);


--
-- Name: idx_project_members_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_project_members_user_id ON public.project_members USING btree (user_id);


--
-- Name: idx_projects_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_created_by ON public.projects USING btree (created_by);


--
-- Name: idx_projects_git_provider_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_git_provider_key ON public.projects USING btree (git_provider_key);


--
-- Name: idx_projects_repository_auth_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_repository_auth_id ON public.projects USING btree (repository_auth_id);


--
-- Name: idx_projects_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_status ON public.projects USING btree (status);


--
-- Name: idx_projects_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_projects_tenant_id ON public.projects USING btree (tenant_id);


--
-- Name: idx_provider_permissions_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_provider_permissions_provider ON public.provider_permissions USING btree (provider_id);


--
-- Name: idx_provider_permissions_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_provider_permissions_tenant ON public.provider_permissions USING btree (tenant_id);


--
-- Name: idx_providers_tenant_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_providers_tenant_type ON public.infrastructure_providers USING btree (tenant_id, provider_type);


--
-- Name: idx_rbac_roles_is_system; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rbac_roles_is_system ON public.rbac_roles USING btree (is_system);


--
-- Name: idx_rbac_roles_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rbac_roles_name ON public.rbac_roles USING btree (name);


--
-- Name: idx_rbac_roles_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rbac_roles_tenant_id ON public.rbac_roles USING btree (tenant_id);


--
-- Name: idx_repository_auth_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_repository_auth_active ON public.repository_auth USING btree (is_active);


--
-- Name: idx_repository_auth_created_by; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_repository_auth_created_by ON public.repository_auth USING btree (created_by);


--
-- Name: idx_repository_auth_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_repository_auth_project_id ON public.repository_auth USING btree (project_id);


--
-- Name: idx_resource_quotas_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_resource_quotas_company_id ON public.resource_quotas USING btree (company_id);


--
-- Name: idx_resource_quotas_org_unit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_resource_quotas_org_unit_id ON public.resource_quotas USING btree (org_unit_id);


--
-- Name: idx_resource_quotas_scope; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_resource_quotas_scope ON public.resource_quotas USING btree (scope);


--
-- Name: idx_role_permissions_permission_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_permissions_permission_id ON public.role_permissions USING btree (permission_id);


--
-- Name: idx_role_permissions_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_permissions_role_id ON public.role_permissions USING btree (role_id);


--
-- Name: idx_sbom_packages_image_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sbom_packages_image_id ON public.sbom_packages USING btree (image_id);


--
-- Name: idx_sbom_packages_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sbom_packages_name ON public.sbom_packages USING btree (package_name);


--
-- Name: idx_sbom_packages_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sbom_packages_type ON public.sbom_packages USING btree (package_type);


--
-- Name: idx_security_policies_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_security_policies_tenant_id ON public.security_policies USING btree (tenant_id);


--
-- Name: idx_security_policies_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_security_policies_type ON public.security_policies USING btree (policy_type);


--
-- Name: idx_system_configs_config_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_configs_config_key ON public.system_configs USING btree (config_key);


--
-- Name: idx_system_configs_config_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_configs_config_type ON public.system_configs USING btree (config_type);


--
-- Name: idx_system_configs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_configs_status ON public.system_configs USING btree (status);


--
-- Name: idx_system_configs_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_system_configs_tenant_id ON public.system_configs USING btree (tenant_id);


--
-- Name: idx_system_configs_tenant_type_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_system_configs_tenant_type_key ON public.system_configs USING btree (tenant_id, config_type, config_key) WHERE (tenant_id IS NOT NULL);


--
-- Name: idx_system_configs_universal_type_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_system_configs_universal_type_key ON public.system_configs USING btree (config_type, config_key) WHERE (tenant_id IS NULL);


--
-- Name: idx_tenant_groups_role_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_groups_role_type ON public.tenant_groups USING btree (role_type);


--
-- Name: idx_tenant_groups_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_groups_status ON public.tenant_groups USING btree (status);


--
-- Name: idx_tenant_groups_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_groups_tenant_id ON public.tenant_groups USING btree (tenant_id);


--
-- Name: idx_tenants_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_status ON public.tenants USING btree (status);


--
-- Name: idx_tenants_tenant_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenants_tenant_code ON public.tenants USING btree (tenant_code);


--
-- Name: idx_usage_tracking_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_tracking_company_id ON public.usage_tracking USING btree (company_id);


--
-- Name: idx_usage_tracking_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_tracking_date ON public.usage_tracking USING btree (usage_date DESC);


--
-- Name: idx_usage_tracking_org_unit_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_tracking_org_unit_id ON public.usage_tracking USING btree (org_unit_id);


--
-- Name: idx_user_invitations_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_invitations_email ON public.user_invitations USING btree (email);


--
-- Name: idx_user_invitations_invite_token; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_invitations_invite_token ON public.user_invitations USING btree (invite_token);


--
-- Name: idx_user_invitations_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_invitations_status ON public.user_invitations USING btree (status);


--
-- Name: idx_user_invitations_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_invitations_tenant_id ON public.user_invitations USING btree (tenant_id);


--
-- Name: idx_user_role_assignments_role_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_role_assignments_role_id ON public.user_role_assignments USING btree (role_id);


--
-- Name: idx_user_role_assignments_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_role_assignments_tenant_id ON public.user_role_assignments USING btree (tenant_id);


--
-- Name: idx_user_role_assignments_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_role_assignments_user_id ON public.user_role_assignments USING btree (user_id);


--
-- Name: idx_user_role_assignments_user_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_role_assignments_user_role ON public.user_role_assignments USING btree (user_id, role_id);


--
-- Name: idx_user_sessions_is_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_sessions_is_active ON public.user_sessions USING btree (is_active);


--
-- Name: idx_user_sessions_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_sessions_tenant_id ON public.user_sessions USING btree (tenant_id);


--
-- Name: idx_user_sessions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_sessions_user_id ON public.user_sessions USING btree (user_id);


--
-- Name: idx_users_auth_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_auth_method ON public.users USING btree (auth_method);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_ldap_username; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_ldap_username ON public.users USING btree (ldap_username);


--
-- Name: idx_users_locked_until; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_locked_until ON public.users USING btree (locked_until) WHERE (locked_until IS NOT NULL);


--
-- Name: idx_users_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_status ON public.users USING btree (status);


--
-- Name: idx_users_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_tenant_id ON public.users USING btree (tenant_id);


--
-- Name: idx_vulnerability_suppressions_company_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vulnerability_suppressions_company_id ON public.vulnerability_suppressions USING btree (company_id);


--
-- Name: idx_vulnerability_suppressions_cve_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vulnerability_suppressions_cve_id ON public.vulnerability_suppressions USING btree (cve_id);


--
-- Name: idx_vulnerability_suppressions_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vulnerability_suppressions_expires_at ON public.vulnerability_suppressions USING btree (expires_at);


--
-- Name: idx_vulnerability_suppressions_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vulnerability_suppressions_status ON public.vulnerability_suppressions USING btree (status);


--
-- Name: idx_workers_available_capacity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workers_available_capacity ON public.workers USING btree (current_load) WHERE ((status)::text = 'available'::text);


--
-- Name: idx_workers_last_heartbeat; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workers_last_heartbeat ON public.workers USING btree (last_heartbeat DESC);


--
-- Name: idx_workers_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workers_status ON public.workers USING btree (status);


--
-- Name: idx_workers_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workers_tenant_id ON public.workers USING btree (tenant_id);


--
-- Name: idx_workers_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workers_type ON public.workers USING btree (worker_type);


--
-- Name: idx_workflow_events_instance; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_instance ON public.workflow_events USING btree (instance_id);


--
-- Name: idx_workflow_events_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_type ON public.workflow_events USING btree (type);


--
-- Name: idx_workflow_instances_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_instances_status ON public.workflow_instances USING btree (status);


--
-- Name: idx_workflow_instances_subject; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_instances_subject ON public.workflow_instances USING btree (subject_type, subject_id);


--
-- Name: idx_workflow_instances_tenant; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_instances_tenant ON public.workflow_instances USING btree (tenant_id);


--
-- Name: idx_workflow_steps_instance; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_steps_instance ON public.workflow_steps USING btree (instance_id);


--
-- Name: idx_workflow_steps_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_steps_status ON public.workflow_steps USING btree (status);


--
-- Name: v_infrastructure_nodes _RETURN; Type: RULE; Schema: public; Owner: -
--

CREATE OR REPLACE VIEW public.v_infrastructure_nodes AS
 SELECT n.id,
    n.name,
    n.status,
    n.last_heartbeat,
    (EXTRACT(epoch FROM (now() - n.last_heartbeat)))::integer AS heartbeat_age_seconds,
    n.total_cpu_cores AS total_cpu_capacity,
    n.total_memory_gb AS total_memory_capacity_gb,
    n.total_disk_gb AS total_disk_capacity_gb,
    n.maintenance_mode,
    n.labels,
    (COALESCE(sum(
        CASE
            WHEN ((b.status)::text = 'running'::text) THEN 1
            ELSE 0
        END), (0)::bigint))::integer AS current_builds,
    (COALESCE(sum(
        CASE
            WHEN ((b.status)::text = 'running'::text) THEN b.cpu_required
            ELSE (0)::numeric
        END), (0)::numeric))::numeric(10,2) AS used_cpu_cores,
    (COALESCE(sum(
        CASE
            WHEN ((b.status)::text = 'running'::text) THEN b.memory_required_gb
            ELSE (0)::numeric
        END), (0)::numeric))::numeric(10,2) AS used_memory_gb,
    ((n.total_cpu_cores - COALESCE(sum(
        CASE
            WHEN ((b.status)::text = 'running'::text) THEN b.cpu_required
            ELSE (0)::numeric
        END), (0)::numeric)))::numeric(10,2) AS available_cpu_cores,
    ((n.total_memory_gb - COALESCE(sum(
        CASE
            WHEN ((b.status)::text = 'running'::text) THEN b.memory_required_gb
            ELSE (0)::numeric
        END), (0)::numeric)))::numeric(10,2) AS available_memory_gb,
    (
        CASE
            WHEN (n.total_cpu_cores > (0)::numeric) THEN round(((COALESCE(sum(
            CASE
                WHEN ((b.status)::text = 'running'::text) THEN b.cpu_required
                ELSE (0)::numeric
            END), (0)::numeric) / n.total_cpu_cores) * (100)::numeric), 2)
            ELSE (0)::numeric
        END)::numeric(5,2) AS cpu_usage_percent,
    (
        CASE
            WHEN (n.total_memory_gb > (0)::numeric) THEN round(((COALESCE(sum(
            CASE
                WHEN ((b.status)::text = 'running'::text) THEN b.memory_required_gb
                ELSE (0)::numeric
            END), (0)::numeric) / n.total_memory_gb) * (100)::numeric), 2)
            ELSE (0)::numeric
        END)::numeric(5,2) AS memory_usage_percent,
    n.created_at,
    n.updated_at
   FROM (public.infrastructure_nodes n
     LEFT JOIN public.builds b ON ((n.id = b.assigned_node_id)))
  GROUP BY n.id;


--
-- Name: email_queue email_queue_timestamp_trigger; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER email_queue_timestamp_trigger BEFORE UPDATE ON public.email_queue FOR EACH ROW EXECUTE FUNCTION public.update_email_queue_timestamp();


--
-- Name: config_template_shares trigger_update_config_template_shares_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_config_template_shares_timestamp BEFORE UPDATE ON public.config_template_shares FOR EACH ROW EXECUTE FUNCTION public.update_config_templates_timestamp();


--
-- Name: config_templates trigger_update_config_templates_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_config_templates_timestamp BEFORE UPDATE ON public.config_templates FOR EACH ROW EXECUTE FUNCTION public.update_config_templates_timestamp();


--
-- Name: approval_requests update_approval_requests_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_approval_requests_updated_at BEFORE UPDATE ON public.approval_requests FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: approval_workflows update_approval_workflows_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_approval_workflows_updated_at BEFORE UPDATE ON public.approval_workflows FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: build_configs update_build_configs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_build_configs_updated_at BEFORE UPDATE ON public.build_configs FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: build_executions update_build_executions_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_build_executions_timestamp BEFORE UPDATE ON public.build_executions FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: build_policies update_build_policies_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_build_policies_updated_at BEFORE UPDATE ON public.build_policies FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: build_triggers update_build_triggers_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_build_triggers_timestamp BEFORE UPDATE ON public.build_triggers FOR EACH ROW EXECUTE FUNCTION public.update_build_triggers_updated_at();


--
-- Name: builds update_builds_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_builds_updated_at BEFORE UPDATE ON public.builds FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: change_requests update_change_requests_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_change_requests_updated_at BEFORE UPDATE ON public.change_requests FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: companies update_companies_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_companies_updated_at BEFORE UPDATE ON public.companies FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: compliance_assessments update_compliance_assessments_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_compliance_assessments_updated_at BEFORE UPDATE ON public.compliance_assessments FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: compliance_controls update_compliance_controls_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_compliance_controls_updated_at BEFORE UPDATE ON public.compliance_controls FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: compliance_evidence update_compliance_evidence_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_compliance_evidence_updated_at BEFORE UPDATE ON public.compliance_evidence FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: compliance_frameworks update_compliance_frameworks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_compliance_frameworks_updated_at BEFORE UPDATE ON public.compliance_frameworks FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: container_registries update_container_registries_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_container_registries_updated_at BEFORE UPDATE ON public.container_registries FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: container_repositories update_container_repositories_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_container_repositories_updated_at BEFORE UPDATE ON public.container_repositories FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: deployment_environments update_deployment_environments_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_deployment_environments_updated_at BEFORE UPDATE ON public.deployment_environments FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: deployments update_deployments_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_deployments_updated_at BEFORE UPDATE ON public.deployments FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: environment_access update_environment_access_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_environment_access_updated_at BEFORE UPDATE ON public.environment_access FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: external_services update_external_services_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_external_services_timestamp BEFORE UPDATE ON public.external_services FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: git_integration update_git_integration_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_git_integration_updated_at BEFORE UPDATE ON public.git_integration FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: git_providers update_git_providers_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_git_providers_timestamp BEFORE UPDATE ON public.git_providers FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: git_repositories update_git_repositories_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_git_repositories_updated_at BEFORE UPDATE ON public.git_repositories FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: image_metadata update_image_metadata_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_image_metadata_updated_at BEFORE UPDATE ON public.image_metadata FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: image_sbom update_image_sbom_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_image_sbom_updated_at BEFORE UPDATE ON public.image_sbom FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: images update_images_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_images_updated_at BEFORE UPDATE ON public.images FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: incidents update_incidents_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_incidents_updated_at BEFORE UPDATE ON public.incidents FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: notification_channels update_notification_channels_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_notification_channels_updated_at BEFORE UPDATE ON public.notification_channels FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: org_units update_org_units_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_org_units_updated_at BEFORE UPDATE ON public.org_units FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: project_members update_project_members_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_project_members_updated_at BEFORE UPDATE ON public.project_members FOR EACH ROW EXECUTE FUNCTION public.update_project_members_timestamp();


--
-- Name: projects update_projects_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON public.projects FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: rbac_roles update_rbac_roles_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_rbac_roles_updated_at BEFORE UPDATE ON public.rbac_roles FOR EACH ROW EXECUTE FUNCTION public.update_rbac_roles_timestamp();


--
-- Name: repository_auth update_repository_auth_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_repository_auth_timestamp BEFORE UPDATE ON public.repository_auth FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: resource_quotas update_resource_quotas_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_resource_quotas_updated_at BEFORE UPDATE ON public.resource_quotas FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: security_policies update_security_policies_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_security_policies_updated_at BEFORE UPDATE ON public.security_policies FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: system_configs update_system_configs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_system_configs_updated_at BEFORE UPDATE ON public.system_configs FOR EACH ROW EXECUTE FUNCTION public.update_system_configs_updated_at();


--
-- Name: tenant_groups update_tenant_groups_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_tenant_groups_updated_at BEFORE UPDATE ON public.tenant_groups FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: tenants update_tenants_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON public.tenants FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: user_invitations update_user_invitations_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_user_invitations_updated_at BEFORE UPDATE ON public.user_invitations FOR EACH ROW EXECUTE FUNCTION public.update_user_invitations_timestamp();


--
-- Name: user_role_assignments update_user_role_assignments_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_user_role_assignments_updated_at BEFORE UPDATE ON public.user_role_assignments FOR EACH ROW EXECUTE FUNCTION public.update_user_role_assignments_timestamp();


--
-- Name: user_sessions update_user_sessions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_user_sessions_updated_at BEFORE UPDATE ON public.user_sessions FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: users update_users_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: vulnerability_suppressions update_vulnerability_suppressions_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_vulnerability_suppressions_updated_at BEFORE UPDATE ON public.vulnerability_suppressions FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();


--
-- Name: workers update_workers_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_workers_updated_at BEFORE UPDATE ON public.workers FOR EACH ROW EXECUTE FUNCTION public.update_workers_timestamp();


--
-- Name: api_keys api_keys_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: api_keys api_keys_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: approval_requests approval_requests_approval_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_approval_workflow_id_fkey FOREIGN KEY (approval_workflow_id) REFERENCES public.approval_workflows(id) ON DELETE CASCADE;


--
-- Name: approval_requests approval_requests_approved_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_approved_by_user_id_fkey FOREIGN KEY (approved_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: approval_requests approval_requests_requested_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_requested_by_user_id_fkey FOREIGN KEY (requested_by_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: approval_workflows approval_workflows_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_workflows
    ADD CONSTRAINT approval_workflows_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: audit_events audit_events_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_events
    ADD CONSTRAINT audit_events_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: audit_events audit_events_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_events
    ADD CONSTRAINT audit_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: build_artifacts build_artifacts_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_artifacts
    ADD CONSTRAINT build_artifacts_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE CASCADE;


--
-- Name: build_artifacts build_artifacts_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_artifacts
    ADD CONSTRAINT build_artifacts_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE SET NULL;


--
-- Name: build_configs build_configs_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_configs
    ADD CONSTRAINT build_configs_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE CASCADE;


--
-- Name: build_execution_logs build_execution_logs_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_execution_logs
    ADD CONSTRAINT build_execution_logs_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.build_executions(id) ON DELETE CASCADE;


--
-- Name: build_executions build_executions_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_executions
    ADD CONSTRAINT build_executions_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE CASCADE;


--
-- Name: build_executions build_executions_config_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_executions
    ADD CONSTRAINT build_executions_config_id_fkey FOREIGN KEY (config_id) REFERENCES public.build_configs(id);


--
-- Name: build_executions build_executions_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_executions
    ADD CONSTRAINT build_executions_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: build_history build_history_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_history
    ADD CONSTRAINT build_history_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE CASCADE;


--
-- Name: build_history build_history_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_history
    ADD CONSTRAINT build_history_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: build_history build_history_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_history
    ADD CONSTRAINT build_history_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: build_history build_history_worker_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_history
    ADD CONSTRAINT build_history_worker_id_fkey FOREIGN KEY (worker_id) REFERENCES public.workers(id) ON DELETE SET NULL;


--
-- Name: build_metrics build_metrics_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_metrics
    ADD CONSTRAINT build_metrics_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE CASCADE;


--
-- Name: build_policies build_policies_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_policies
    ADD CONSTRAINT build_policies_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: build_policies build_policies_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_policies
    ADD CONSTRAINT build_policies_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: build_policies build_policies_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_policies
    ADD CONSTRAINT build_policies_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id);


--
-- Name: build_steps build_steps_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_steps
    ADD CONSTRAINT build_steps_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE CASCADE;


--
-- Name: build_triggers build_triggers_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_triggers
    ADD CONSTRAINT build_triggers_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE CASCADE;


--
-- Name: build_triggers build_triggers_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_triggers
    ADD CONSTRAINT build_triggers_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: build_triggers build_triggers_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_triggers
    ADD CONSTRAINT build_triggers_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: build_triggers build_triggers_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.build_triggers
    ADD CONSTRAINT build_triggers_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: builds builds_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.builds
    ADD CONSTRAINT builds_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE SET NULL;


--
-- Name: builds builds_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.builds
    ADD CONSTRAINT builds_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: builds builds_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.builds
    ADD CONSTRAINT builds_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: builds builds_triggered_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.builds
    ADD CONSTRAINT builds_triggered_by_user_id_fkey FOREIGN KEY (triggered_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: catalog_image_tags catalog_image_tags_catalog_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_image_tags
    ADD CONSTRAINT catalog_image_tags_catalog_image_id_fkey FOREIGN KEY (catalog_image_id) REFERENCES public.catalog_images(id) ON DELETE CASCADE;


--
-- Name: catalog_image_tags catalog_image_tags_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_image_tags
    ADD CONSTRAINT catalog_image_tags_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: catalog_image_versions catalog_image_versions_catalog_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_image_versions
    ADD CONSTRAINT catalog_image_versions_catalog_image_id_fkey FOREIGN KEY (catalog_image_id) REFERENCES public.catalog_images(id) ON DELETE CASCADE;


--
-- Name: catalog_images catalog_images_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_images
    ADD CONSTRAINT catalog_images_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: catalog_images catalog_images_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_images
    ADD CONSTRAINT catalog_images_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: catalog_images catalog_images_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.catalog_images
    ADD CONSTRAINT catalog_images_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id);


--
-- Name: change_requests change_requests_affected_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.change_requests
    ADD CONSTRAINT change_requests_affected_org_unit_id_fkey FOREIGN KEY (affected_org_unit_id) REFERENCES public.org_units(id) ON DELETE SET NULL;


--
-- Name: change_requests change_requests_approved_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.change_requests
    ADD CONSTRAINT change_requests_approved_by_user_id_fkey FOREIGN KEY (approved_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: change_requests change_requests_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.change_requests
    ADD CONSTRAINT change_requests_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: change_requests change_requests_requested_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.change_requests
    ADD CONSTRAINT change_requests_requested_by_user_id_fkey FOREIGN KEY (requested_by_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: compliance_assessments compliance_assessments_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_assessments
    ADD CONSTRAINT compliance_assessments_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: compliance_assessments compliance_assessments_compliance_framework_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_assessments
    ADD CONSTRAINT compliance_assessments_compliance_framework_id_fkey FOREIGN KEY (compliance_framework_id) REFERENCES public.compliance_frameworks(id) ON DELETE CASCADE;


--
-- Name: compliance_assessments compliance_assessments_conducted_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_assessments
    ADD CONSTRAINT compliance_assessments_conducted_by_user_id_fkey FOREIGN KEY (conducted_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: compliance_assessments compliance_assessments_scope_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_assessments
    ADD CONSTRAINT compliance_assessments_scope_org_unit_id_fkey FOREIGN KEY (scope_org_unit_id) REFERENCES public.org_units(id) ON DELETE SET NULL;


--
-- Name: compliance_controls compliance_controls_compliance_framework_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_controls
    ADD CONSTRAINT compliance_controls_compliance_framework_id_fkey FOREIGN KEY (compliance_framework_id) REFERENCES public.compliance_frameworks(id) ON DELETE CASCADE;


--
-- Name: compliance_controls compliance_controls_responsible_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_controls
    ADD CONSTRAINT compliance_controls_responsible_org_unit_id_fkey FOREIGN KEY (responsible_org_unit_id) REFERENCES public.org_units(id) ON DELETE SET NULL;


--
-- Name: compliance_controls compliance_controls_responsible_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_controls
    ADD CONSTRAINT compliance_controls_responsible_user_id_fkey FOREIGN KEY (responsible_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: compliance_evidence compliance_evidence_compliance_control_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_evidence
    ADD CONSTRAINT compliance_evidence_compliance_control_id_fkey FOREIGN KEY (compliance_control_id) REFERENCES public.compliance_controls(id) ON DELETE CASCADE;


--
-- Name: compliance_frameworks compliance_frameworks_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.compliance_frameworks
    ADD CONSTRAINT compliance_frameworks_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: config_template_shares config_template_shares_shared_with_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_template_shares
    ADD CONSTRAINT config_template_shares_shared_with_user_id_fkey FOREIGN KEY (shared_with_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: config_template_shares config_template_shares_template_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_template_shares
    ADD CONSTRAINT config_template_shares_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.config_templates(id) ON DELETE CASCADE;


--
-- Name: config_templates config_templates_created_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_templates
    ADD CONSTRAINT config_templates_created_by_user_id_fkey FOREIGN KEY (created_by_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: config_templates config_templates_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_templates
    ADD CONSTRAINT config_templates_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: container_registries container_registries_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_registries
    ADD CONSTRAINT container_registries_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: container_repositories container_repositories_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_repositories
    ADD CONSTRAINT container_repositories_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: container_repositories container_repositories_owned_by_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_repositories
    ADD CONSTRAINT container_repositories_owned_by_org_unit_id_fkey FOREIGN KEY (owned_by_org_unit_id) REFERENCES public.org_units(id) ON DELETE SET NULL;


--
-- Name: container_repositories container_repositories_registry_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.container_repositories
    ADD CONSTRAINT container_repositories_registry_id_fkey FOREIGN KEY (registry_id) REFERENCES public.container_registries(id) ON DELETE CASCADE;


--
-- Name: deployment_environments deployment_environments_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_environments
    ADD CONSTRAINT deployment_environments_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: deployment_environments deployment_environments_owned_by_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployment_environments
    ADD CONSTRAINT deployment_environments_owned_by_org_unit_id_fkey FOREIGN KEY (owned_by_org_unit_id) REFERENCES public.org_units(id) ON DELETE SET NULL;


--
-- Name: deployments deployments_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployments
    ADD CONSTRAINT deployments_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: deployments deployments_deployment_environment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployments
    ADD CONSTRAINT deployments_deployment_environment_id_fkey FOREIGN KEY (deployment_environment_id) REFERENCES public.deployment_environments(id) ON DELETE CASCADE;


--
-- Name: deployments deployments_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployments
    ADD CONSTRAINT deployments_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE RESTRICT;


--
-- Name: deployments deployments_previous_deployment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployments
    ADD CONSTRAINT deployments_previous_deployment_id_fkey FOREIGN KEY (previous_deployment_id) REFERENCES public.deployments(id) ON DELETE SET NULL;


--
-- Name: deployments deployments_triggered_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deployments
    ADD CONSTRAINT deployments_triggered_by_user_id_fkey FOREIGN KEY (triggered_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: email_queue email_queue_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_queue
    ADD CONSTRAINT email_queue_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: environment_access environment_access_deployment_environment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment_access
    ADD CONSTRAINT environment_access_deployment_environment_id_fkey FOREIGN KEY (deployment_environment_id) REFERENCES public.deployment_environments(id) ON DELETE CASCADE;


--
-- Name: environment_access environment_access_granted_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment_access
    ADD CONSTRAINT environment_access_granted_by_user_id_fkey FOREIGN KEY (granted_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: environment_access environment_access_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.environment_access
    ADD CONSTRAINT environment_access_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: external_services external_services_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: external_services external_services_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: node_resource_usage fk_node_resource_usage_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_resource_usage
    ADD CONSTRAINT fk_node_resource_usage_node FOREIGN KEY (node_id) REFERENCES public.infrastructure_nodes(id);


--
-- Name: git_integration git_integration_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_integration
    ADD CONSTRAINT git_integration_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: git_integration git_integration_git_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_integration
    ADD CONSTRAINT git_integration_git_repository_id_fkey FOREIGN KEY (git_repository_id) REFERENCES public.git_repositories(id) ON DELETE CASCADE;


--
-- Name: git_repositories git_repositories_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.git_repositories
    ADD CONSTRAINT git_repositories_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: group_members group_members_added_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_members
    ADD CONSTRAINT group_members_added_by_fkey FOREIGN KEY (added_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: group_members group_members_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_members
    ADD CONSTRAINT group_members_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.tenant_groups(id) ON DELETE CASCADE;


--
-- Name: group_members group_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_members
    ADD CONSTRAINT group_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: image_layers image_layers_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_layers
    ADD CONSTRAINT image_layers_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE CASCADE;


--
-- Name: image_metadata image_metadata_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_metadata
    ADD CONSTRAINT image_metadata_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE CASCADE;


--
-- Name: image_sbom image_sbom_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_sbom
    ADD CONSTRAINT image_sbom_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE CASCADE;


--
-- Name: image_vulnerability_scans image_vulnerability_scans_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_vulnerability_scans
    ADD CONSTRAINT image_vulnerability_scans_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builds(id) ON DELETE SET NULL;


--
-- Name: image_vulnerability_scans image_vulnerability_scans_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.image_vulnerability_scans
    ADD CONSTRAINT image_vulnerability_scans_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE CASCADE;


--
-- Name: images images_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.images
    ADD CONSTRAINT images_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: incident_timelines incident_timelines_actor_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incident_timelines
    ADD CONSTRAINT incident_timelines_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: incident_timelines incident_timelines_incident_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incident_timelines
    ADD CONSTRAINT incident_timelines_incident_id_fkey FOREIGN KEY (incident_id) REFERENCES public.incidents(id) ON DELETE CASCADE;


--
-- Name: incidents incidents_acknowledged_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidents
    ADD CONSTRAINT incidents_acknowledged_by_user_id_fkey FOREIGN KEY (acknowledged_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: incidents incidents_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidents
    ADD CONSTRAINT incidents_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: incidents incidents_reported_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidents
    ADD CONSTRAINT incidents_reported_by_user_id_fkey FOREIGN KEY (reported_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: infrastructure_providers infrastructure_providers_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_providers
    ADD CONSTRAINT infrastructure_providers_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: infrastructure_providers infrastructure_providers_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_providers
    ADD CONSTRAINT infrastructure_providers_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: infrastructure_usage infrastructure_usage_build_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_usage
    ADD CONSTRAINT infrastructure_usage_build_execution_id_fkey FOREIGN KEY (build_execution_id) REFERENCES public.build_executions(id) ON DELETE SET NULL;


--
-- Name: infrastructure_usage infrastructure_usage_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.infrastructure_usage
    ADD CONSTRAINT infrastructure_usage_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: node_resource_usage node_resource_usage_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_resource_usage
    ADD CONSTRAINT node_resource_usage_node_id_fkey FOREIGN KEY (node_id) REFERENCES public.infrastructure_nodes(id) ON DELETE CASCADE;


--
-- Name: notification_channels notification_channels_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_channels
    ADD CONSTRAINT notification_channels_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: notifications notifications_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: notifications notifications_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: org_unit_access org_unit_access_granted_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_unit_access
    ADD CONSTRAINT org_unit_access_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: org_unit_access org_unit_access_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_unit_access
    ADD CONSTRAINT org_unit_access_org_unit_id_fkey FOREIGN KEY (org_unit_id) REFERENCES public.org_units(id) ON DELETE CASCADE;


--
-- Name: org_unit_access org_unit_access_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_unit_access
    ADD CONSTRAINT org_unit_access_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: org_units org_units_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_units
    ADD CONSTRAINT org_units_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: org_units org_units_parent_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_units
    ADD CONSTRAINT org_units_parent_org_unit_id_fkey FOREIGN KEY (parent_org_unit_id) REFERENCES public.org_units(id) ON DELETE SET NULL;


--
-- Name: package_vulnerabilities package_vulnerabilities_cve_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_vulnerabilities
    ADD CONSTRAINT package_vulnerabilities_cve_id_fkey FOREIGN KEY (cve_id) REFERENCES public.cve_database(cve_id) ON DELETE CASCADE;


--
-- Name: project_members project_members_assigned_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_assigned_by_user_id_fkey FOREIGN KEY (assigned_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: project_members project_members_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: project_members project_members_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.rbac_roles(id) ON DELETE SET NULL;


--
-- Name: project_members project_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: projects projects_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: projects projects_repository_auth_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_repository_auth_id_fkey FOREIGN KEY (repository_auth_id) REFERENCES public.repository_auth(id);


--
-- Name: projects projects_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: provider_permissions provider_permissions_granted_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provider_permissions
    ADD CONSTRAINT provider_permissions_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES public.users(id);


--
-- Name: provider_permissions provider_permissions_provider_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provider_permissions
    ADD CONSTRAINT provider_permissions_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES public.infrastructure_providers(id) ON DELETE CASCADE;


--
-- Name: provider_permissions provider_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provider_permissions
    ADD CONSTRAINT provider_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: rbac_roles rbac_roles_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rbac_roles
    ADD CONSTRAINT rbac_roles_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE SET NULL;


--
-- Name: repository_auth repository_auth_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repository_auth
    ADD CONSTRAINT repository_auth_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: repository_auth repository_auth_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repository_auth
    ADD CONSTRAINT repository_auth_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: resource_quotas resource_quotas_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_quotas
    ADD CONSTRAINT resource_quotas_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: resource_quotas resource_quotas_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.resource_quotas
    ADD CONSTRAINT resource_quotas_org_unit_id_fkey FOREIGN KEY (org_unit_id) REFERENCES public.org_units(id) ON DELETE CASCADE;


--
-- Name: role_permissions role_permissions_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE;


--
-- Name: sbom_packages sbom_packages_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sbom_packages
    ADD CONSTRAINT sbom_packages_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE CASCADE;


--
-- Name: sbom_packages sbom_packages_image_sbom_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sbom_packages
    ADD CONSTRAINT sbom_packages_image_sbom_id_fkey FOREIGN KEY (image_sbom_id) REFERENCES public.image_sbom(id) ON DELETE CASCADE;


--
-- Name: security_policies security_policies_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.security_policies
    ADD CONSTRAINT security_policies_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: system_configs system_configs_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_configs
    ADD CONSTRAINT system_configs_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: system_configs system_configs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_configs
    ADD CONSTRAINT system_configs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: system_configs system_configs_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_configs
    ADD CONSTRAINT system_configs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: tenant_groups tenant_groups_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_groups
    ADD CONSTRAINT tenant_groups_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: usage_tracking usage_tracking_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_tracking
    ADD CONSTRAINT usage_tracking_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: usage_tracking usage_tracking_org_unit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_tracking
    ADD CONSTRAINT usage_tracking_org_unit_id_fkey FOREIGN KEY (org_unit_id) REFERENCES public.org_units(id) ON DELETE CASCADE;


--
-- Name: user_invitations user_invitations_invited_by_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_invitations
    ADD CONSTRAINT user_invitations_invited_by_id_fkey FOREIGN KEY (invited_by_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_invitations user_invitations_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_invitations
    ADD CONSTRAINT user_invitations_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.rbac_roles(id) ON DELETE SET NULL;


--
-- Name: user_invitations user_invitations_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_invitations
    ADD CONSTRAINT user_invitations_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: user_role_assignments user_role_assignments_assigned_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_role_assignments
    ADD CONSTRAINT user_role_assignments_assigned_by_user_id_fkey FOREIGN KEY (assigned_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: user_role_assignments user_role_assignments_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_role_assignments
    ADD CONSTRAINT user_role_assignments_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.rbac_roles(id) ON DELETE CASCADE;


--
-- Name: user_role_assignments user_role_assignments_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_role_assignments
    ADD CONSTRAINT user_role_assignments_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: user_role_assignments user_role_assignments_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_role_assignments
    ADD CONSTRAINT user_role_assignments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_sessions user_sessions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: user_sessions user_sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE SET NULL;


--
-- Name: vulnerability_suppressions vulnerability_suppressions_approved_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_approved_by_user_id_fkey FOREIGN KEY (approved_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: vulnerability_suppressions vulnerability_suppressions_company_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.companies(id) ON DELETE CASCADE;


--
-- Name: vulnerability_suppressions vulnerability_suppressions_cve_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_cve_id_fkey FOREIGN KEY (cve_id) REFERENCES public.cve_database(cve_id) ON DELETE CASCADE;


--
-- Name: vulnerability_suppressions vulnerability_suppressions_image_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_image_id_fkey FOREIGN KEY (image_id) REFERENCES public.images(id) ON DELETE CASCADE;


--
-- Name: vulnerability_suppressions vulnerability_suppressions_project_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.projects(id) ON DELETE CASCADE;


--
-- Name: vulnerability_suppressions vulnerability_suppressions_suppressed_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_suppressions
    ADD CONSTRAINT vulnerability_suppressions_suppressed_by_user_id_fkey FOREIGN KEY (suppressed_by_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: workers workers_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workers
    ADD CONSTRAINT workers_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: workflow_events workflow_events_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.workflow_instances(id) ON DELETE CASCADE;


--
-- Name: workflow_events workflow_events_step_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_step_id_fkey FOREIGN KEY (step_id) REFERENCES public.workflow_steps(id) ON DELETE SET NULL;


--
-- Name: workflow_instances workflow_instances_definition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_instances
    ADD CONSTRAINT workflow_instances_definition_id_fkey FOREIGN KEY (definition_id) REFERENCES public.workflow_definitions(id) ON DELETE RESTRICT;


--
-- Name: workflow_steps workflow_steps_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_steps
    ADD CONSTRAINT workflow_steps_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.workflow_instances(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict DMXh4VktFgXOfdgS0iVxO9Cg4u6KsaTZlHA5Wk4qHPCc2aIDRAsSEP2BoXkg9eW

