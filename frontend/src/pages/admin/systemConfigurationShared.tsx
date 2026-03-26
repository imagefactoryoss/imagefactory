import { TektonTaskImagesConfig } from "@/types";
import { AlertTriangle } from "lucide-react";
import React from "react";

export interface SystemConfig {
    id: string;
    tenant_id: string;
    config_type: string;
    config_key: string;
    config_value: any;
    status: string;
    description: string;
}

export interface BuildConfigFormData {
    default_timeout_minutes: number;
    max_concurrent_jobs: number;
    worker_pool_size: number;
    max_queue_size: number;
    artifact_retention_days: number;
    tekton_enabled: boolean;
    monitor_event_driven_enabled: boolean;
    enable_temp_scan_stage: boolean;
}

export interface TektonCoreConfigFormData {
    install_source: "manifest" | "helm" | "preinstalled";
    manifest_urls: string[];
    helm_repo_url: string;
    helm_chart: string;
    helm_release_name: string;
    helm_namespace: string;
    assets_dir: string;
}

export type TektonTaskImagesFormData = TektonTaskImagesConfig;

export interface SecurityConfigFormData {
    jwt_expiration_hours: number;
    refresh_token_hours: number;
    max_login_attempts: number;
    account_lock_duration_minutes: number;
    password_min_length: number;
    require_special_chars: boolean;
    require_numbers: boolean;
    require_uppercase: boolean;
    session_timeout_minutes: number;
}

export interface LDAPConfigFormData {
    host: string;
    port: number;
    base_dn: string;
    user_search_base: string;
    group_search_base: string;
    bind_dn: string;
    bind_password: string;
    user_filter: string;
    group_filter: string;
    start_tls: boolean;
    ssl: boolean;
    allowed_domains: string[];
    enabled: boolean;
}

export interface SMTPConfigFormData {
    host: string;
    port: number;
    username: string;
    password: string;
    from: string;
    start_tls: boolean;
    ssl: boolean;
    enabled: boolean;
}

export interface GeneralConfigFormData {
    system_name: string;
    system_description: string;
    admin_email: string;
    support_email: string;
    time_zone: string;
    date_format: string;
    default_language: string;
    workflow_enabled: boolean;
    workflow_poll_interval: string;
    workflow_max_steps_per_tick: number;
    admin_dashboard_poll_interval_seconds: number;
    project_retention_days: number;
    project_last_purge_at?: string;
    project_last_purge_count?: number;
    maintenance_mode: boolean;
}

export interface MessagingConfigFormData {
    enable_nats: boolean;
    nats_required: boolean;
    external_only: boolean;
    outbox_enabled: boolean;
    outbox_relay_interval_seconds: number;
    outbox_relay_batch_size: number;
    outbox_claim_lease_seconds: number;
}

export interface RuntimeAssetStorageProfileFormData {
    type: "hostPath" | "pvc" | "emptyDir";
    host_path: string;
    host_path_type: string;
    pvc_name: string;
    pvc_size: string;
    pvc_storage_class: string;
    pvc_access_modes: string[];
}

export interface RuntimeAssetStorageProfilesFormData {
    internal_registry: RuntimeAssetStorageProfileFormData;
    trivy_cache: RuntimeAssetStorageProfileFormData;
}

export interface RuntimeServicesConfigFormData {
    dispatcher_url: string;
    dispatcher_port: number;
    dispatcher_mtls_enabled: boolean;
    dispatcher_ca_cert: string;
    dispatcher_client_cert: string;
    dispatcher_client_key: string;
    workflow_orchestrator_enabled: boolean;
    email_worker_url: string;
    email_worker_port: number;
    email_worker_tls_enabled: boolean;
    notification_worker_url: string;
    notification_worker_port: number;
    notification_tls_enabled: boolean;
    internal_registry_gc_worker_url: string;
    internal_registry_gc_worker_port: number;
    internal_registry_gc_worker_tls_enabled: boolean;
    health_check_timeout_seconds: number;
    internal_registry_temp_cleanup_enabled: boolean;
    internal_registry_temp_cleanup_retention_hours: number;
    internal_registry_temp_cleanup_interval_minutes: number;
    internal_registry_temp_cleanup_batch_size: number;
    internal_registry_temp_cleanup_dry_run: boolean;
    provider_readiness_watcher_enabled: boolean;
    provider_readiness_watcher_interval_seconds: number;
    provider_readiness_watcher_timeout_seconds: number;
    provider_readiness_watcher_batch_size: number;
    tenant_asset_reconcile_policy:
    | "full_reconcile_on_prepare"
    | "async_trigger_only"
    | "manual_only";
    tenant_asset_drift_watcher_enabled: boolean;
    tenant_asset_drift_watcher_interval_seconds: number;
    tekton_history_cleanup_enabled: boolean;
    tekton_history_cleanup_schedule: string;
    tekton_history_cleanup_keep_pipelineruns: number;
    tekton_history_cleanup_keep_taskruns: number;
    tekton_history_cleanup_keep_pods: number;
    image_import_notification_receipt_cleanup_enabled: boolean;
    image_import_notification_receipt_retention_days: number;
    image_import_notification_receipt_cleanup_interval_hours: number;
    storage_profiles: RuntimeAssetStorageProfilesFormData;
    // AppHQ tenant lookup service
    apphq_enabled: boolean;
    apphq_oauth_token_url: string;
    apphq_client_id: string;
    apphq_client_secret: string;
    apphq_api_url: string;
    apphq_system: string;
    apphq_system_name: string;
    apphq_run: string;
    apphq_obj_cd: string;
}

export interface RuntimeServicesFieldErrors {
    dispatcher_url?: string;
    dispatcher_port?: string;
    email_worker_url?: string;
    email_worker_port?: string;
    notification_worker_url?: string;
    notification_worker_port?: string;
    internal_registry_gc_worker_url?: string;
    internal_registry_gc_worker_port?: string;
    health_check_timeout_seconds?: string;
    internal_registry_temp_cleanup_retention_hours?: string;
    internal_registry_temp_cleanup_interval_minutes?: string;
    internal_registry_temp_cleanup_batch_size?: string;
    tekton_history_cleanup_schedule?: string;
    image_import_notification_receipt_retention_days?: string;
    image_import_notification_receipt_cleanup_interval_hours?: string;
    provider_readiness_watcher_interval_seconds?: string;
    provider_readiness_watcher_timeout_seconds?: string;
    provider_readiness_watcher_batch_size?: string;
    tekton_history_cleanup_keep_pipelineruns?: string;
    tekton_history_cleanup_keep_taskruns?: string;
    tekton_history_cleanup_keep_pods?: string;
    storage_profiles_internal_registry_type?: string;
    storage_profiles_internal_registry_host_path?: string;
    storage_profiles_internal_registry_host_path_type?: string;
    storage_profiles_internal_registry_pvc_name?: string;
    storage_profiles_internal_registry_pvc_access_modes_0?: string;
    apphq_oauth_token_url?: string;
    apphq_client_id?: string;
    apphq_client_secret?: string;
    apphq_api_url?: string;
    apphq_system?: string;
    apphq_system_name?: string;
    apphq_run?: string;
    apphq_obj_cd?: string;
}

export type RuntimeServicesSubTab =
    | "services"
    | "registry_gc"
    | "watchers"
    | "cleanup"
    | "storage"
    | "tenant_service";

export type TektonSubTab = "core" | "images" | "tenant" | "storage";

export interface QuarantinePolicyFormData {
    enabled: boolean;
    mode: "enforce" | "dry_run";
    max_critical: number;
    max_p2: number;
    max_p3: number;
    max_cvss: number;
    severity_mapping: {
        p1: string[];
        p2: string[];
        p3: string[];
        p4: string[];
    };
}

export interface QuarantinePolicyValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
}

export interface QuarantinePolicySimulationResult {
    decision: string;
    mode: string;
    reasons: string[];
}

export interface SORRegistrationFormData {
    enforce: boolean;
    runtime_error_mode: "error" | "deny" | "allow";
}

export interface RobotSRERemediationPackFormData {
    key: string;
    version: string;
    name: string;
    summary: string;
    risk_tier: "low" | "medium" | "high";
    action_class: string;
    requires_approval: boolean;
    incident_types: string[];
}

export interface RobotSREPolicyFormData {
    display_name: string;
    enabled: boolean;
    environment_mode: string;
    auto_observe_enabled: boolean;
    auto_notify_enabled: boolean;
    auto_contain_enabled: boolean;
    auto_recover_enabled: boolean;
    require_approval_for_recover: boolean;
    require_approval_for_disruptive: boolean;
    duplicate_alert_suppression_seconds: number;
    action_cooldown_seconds: number;
    enabled_domains: string[];
    remediation_packs: RobotSRERemediationPackFormData[];
}

export type SystemConfigurationTabId =
    | "build"
    | "tekton"
    | "security"
    | "ldap"
    | "smtp"
    | "general"
    | "messaging"
    | "runtime_services"
    | "quarantine_policy"
    | "sor_registration"
    | "robot_sre";

export const SYSTEM_CONFIGURATION_TABS: Array<{
    id: SystemConfigurationTabId;
    label: string;
}> = [
        { id: "general", label: "General" },
        { id: "messaging", label: "Messaging" },
        { id: "runtime_services", label: "Runtime Services" },
        { id: "security", label: "Security" },
        { id: "ldap", label: "LDAP" },
        { id: "smtp", label: "SMTP" },
        { id: "build", label: "Build" },
        { id: "tekton", label: "Tekton" },
        { id: "quarantine_policy", label: "Quarantine Policy" },
        { id: "sor_registration", label: "EPR Registration" },
        { id: "robot_sre", label: "Robot SRE" },
    ];

export const getInitialSystemConfigurationTab = (): SystemConfigurationTabId => {
    const saved = localStorage.getItem("system-config-active-tab");
    if (
        saved &&
        SYSTEM_CONFIGURATION_TABS.some((tab) => tab.id === saved)
    ) {
        return saved as SystemConfigurationTabId;
    }
    return "general";
};

export const RestartRequiredBadge: React.FC = () => (
    <span className="inline-flex items-center gap-1 rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-200">
        <AlertTriangle className="h-3 w-3" />
        Restart required
    </span>
);
