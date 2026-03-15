import { api } from "@/services/api";
import { adminService } from "@/services/adminService";
import { auditService } from "@/services/auditService";
import { useCanManageAdmin } from "@/hooks/useAccess";
import { TektonTaskImagesConfig } from "@/types";
import { AuditEvent } from "@/types/audit";
import { AlertTriangle, HelpCircle, X } from "lucide-react";
import React, { useEffect, useState } from "react";
import toast from "react-hot-toast";

interface SystemConfig {
  id: string;
  tenant_id: string;
  config_type: string;
  config_key: string;
  config_value: any;
  status: string;
  description: string;
}

interface BuildConfigFormData {
  default_timeout_minutes: number;
  max_concurrent_jobs: number;
  worker_pool_size: number;
  max_queue_size: number;
  artifact_retention_days: number;
  tekton_enabled: boolean;
  monitor_event_driven_enabled: boolean;
  enable_temp_scan_stage: boolean;
}

interface TektonCoreConfigFormData {
  install_source: "manifest" | "helm" | "preinstalled";
  manifest_urls: string[];
  helm_repo_url: string;
  helm_chart: string;
  helm_release_name: string;
  helm_namespace: string;
  assets_dir: string;
}

type TektonTaskImagesFormData = TektonTaskImagesConfig;

interface SecurityConfigFormData {
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

interface LDAPConfigFormData {
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

interface SMTPConfigFormData {
  host: string;
  port: number;
  username: string;
  password: string;
  from: string;
  start_tls: boolean;
  ssl: boolean;
  enabled: boolean;
}

interface GeneralConfigFormData {
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

interface MessagingConfigFormData {
  enable_nats: boolean;
  nats_required: boolean;
  external_only: boolean;
  outbox_enabled: boolean;
  outbox_relay_interval_seconds: number;
  outbox_relay_batch_size: number;
  outbox_claim_lease_seconds: number;
}

interface RuntimeAssetStorageProfileFormData {
  type: "hostPath" | "pvc" | "emptyDir";
  host_path: string;
  host_path_type: string;
  pvc_name: string;
  pvc_size: string;
  pvc_storage_class: string;
  pvc_access_modes: string[];
}

interface RuntimeAssetStorageProfilesFormData {
  internal_registry: RuntimeAssetStorageProfileFormData;
  trivy_cache: RuntimeAssetStorageProfileFormData;
}

interface RuntimeServicesConfigFormData {
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
}

type RuntimeServicesSubTab =
  | "services"
  | "registry_gc"
  | "watchers"
  | "cleanup"
  | "storage";

type TektonSubTab = "core" | "images" | "tenant" | "storage";

interface QuarantinePolicyFormData {
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

interface QuarantinePolicyValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

interface QuarantinePolicySimulationResult {
  decision: string;
  mode: string;
  reasons: string[];
}

interface SORRegistrationFormData {
  enforce: boolean;
  runtime_error_mode: "error" | "deny" | "allow";
}

const SystemConfigurationPage: React.FC = () => {
  const canManageAdmin = useCanManageAdmin();
  const [loading, setLoading] = useState(true);

  // Get initial tab from localStorage, default to 'general'
  const getInitialTab = ():
    | "build"
    | "tekton"
    | "security"
    | "ldap"
    | "smtp"
    | "general"
    | "messaging"
    | "runtime_services"
    | "quarantine_policy"
    | "sor_registration" => {
    const saved = localStorage.getItem("system-config-active-tab");
    if (
      saved &&
      [
        "build",
        "tekton",
        "security",
        "ldap",
        "smtp",
        "general",
        "messaging",
        "runtime_services",
        "quarantine_policy",
        "sor_registration",
      ].includes(saved)
    ) {
      return saved as
        | "build"
        | "tekton"
        | "security"
        | "ldap"
        | "smtp"
        | "general"
        | "messaging"
        | "runtime_services"
        | "quarantine_policy"
        | "sor_registration";
    }
    return "general";
  };

  const [activeTab, setActiveTab] = useState<
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
  >(getInitialTab);

  // Save active tab to localStorage whenever it changes
  useEffect(() => {
    localStorage.setItem("system-config-active-tab", activeTab);
  }, [activeTab]);

  // Build configuration state
  const [buildConfig, setBuildConfig] = useState<BuildConfigFormData>({
    default_timeout_minutes: 30,
    max_concurrent_jobs: 10,
    worker_pool_size: 5,
    max_queue_size: 100,
    artifact_retention_days: 30,
    tekton_enabled: true,
    monitor_event_driven_enabled: true,
    enable_temp_scan_stage: true,
  });

  const [tektonCoreConfig, setTektonCoreConfig] =
    useState<TektonCoreConfigFormData>({
      install_source: "manifest",
      manifest_urls: [
        "https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml",
      ],
      helm_repo_url: "",
      helm_chart: "tekton-pipeline",
      helm_release_name: "tekton-pipelines",
      helm_namespace: "tekton-pipelines",
      assets_dir: "",
    });
  const [tektonTaskImages, setTektonTaskImages] =
    useState<TektonTaskImagesFormData>({
      git_clone: "docker.io/alpine/git:2.45.2",
      kaniko_executor: "gcr.io/kaniko-project/executor:v1.23.2",
      buildkit: "docker.io/moby/buildkit:v0.13.2",
      skopeo: "quay.io/skopeo/stable:v1.15.0",
      trivy: "docker.io/aquasec/trivy:0.57.1",
      syft: "docker.io/anchore/syft:v1.18.1",
      cosign: "gcr.io/projectsigstore/cosign:v2.4.1",
      packer: "docker.io/hashicorp/packer:1.10.2",
      python_alpine: "docker.io/library/python:3.12-alpine",
      alpine: "docker.io/library/alpine:3.20",
      cleanup_kubectl: "docker.io/bitnami/kubectl:latest",
    });

  // Security configuration state
  const [securityConfig, setSecurityConfig] = useState<SecurityConfigFormData>({
    jwt_expiration_hours: 24,
    refresh_token_hours: 168,
    max_login_attempts: 5,
    account_lock_duration_minutes: 30,
    password_min_length: 8,
    require_special_chars: true,
    require_numbers: true,
    require_uppercase: true,
    session_timeout_minutes: 60,
  });

  // LDAP configuration state
  const [ldapConfig, setLdapConfig] = useState<LDAPConfigFormData>({
    host: "",
    port: 389,
    base_dn: "",
    user_search_base: "",
    group_search_base: "",
    bind_dn: "",
    bind_password: "",
    user_filter: "",
    group_filter: "",
    start_tls: false,
    ssl: false,
    allowed_domains: [],
    enabled: true,
  });
  const [ldapConfigKey, setLdapConfigKey] = useState<string>(
    "ldap_active_directory",
  );

  // SMTP configuration state
  const [smtpConfig, setSmtpConfig] = useState<SMTPConfigFormData>({
    host: "",
    port: 587,
    username: "",
    password: "",
    from: "",
    start_tls: true,
    ssl: false,
    enabled: true,
  });

  // General configuration state
  const [generalConfig, setGeneralConfig] = useState<GeneralConfigFormData>({
    system_name: "",
    system_description: "",
    admin_email: "",
    support_email: "",
    time_zone: "UTC",
    date_format: "YYYY-MM-DD",
    default_language: "en",
    workflow_enabled: true,
    workflow_poll_interval: "3s",
    workflow_max_steps_per_tick: 1,
    admin_dashboard_poll_interval_seconds: 15,
    project_retention_days: 30,
    maintenance_mode: false,
  });

  const [messagingConfig, setMessagingConfig] =
    useState<MessagingConfigFormData>({
      enable_nats: false,
      nats_required: false,
      external_only: false,
      outbox_enabled: true,
      outbox_relay_interval_seconds: 5,
      outbox_relay_batch_size: 100,
      outbox_claim_lease_seconds: 30,
    });

  const [runtimeServicesConfig, setRuntimeServicesConfig] =
    useState<RuntimeServicesConfigFormData>({
      dispatcher_url: "http://localhost",
      dispatcher_port: 8084,
      dispatcher_mtls_enabled: false,
      dispatcher_ca_cert: "",
      dispatcher_client_cert: "",
      dispatcher_client_key: "",
      workflow_orchestrator_enabled: true,
      email_worker_url: "http://localhost",
      email_worker_port: 8081,
      email_worker_tls_enabled: false,
      notification_worker_url: "http://localhost",
      notification_worker_port: 8083,
      notification_tls_enabled: false,
      internal_registry_gc_worker_url: "http://localhost",
      internal_registry_gc_worker_port: 8085,
      internal_registry_gc_worker_tls_enabled: false,
      health_check_timeout_seconds: 5,
      internal_registry_temp_cleanup_enabled: true,
      internal_registry_temp_cleanup_retention_hours: 72,
      internal_registry_temp_cleanup_interval_minutes: 60,
      internal_registry_temp_cleanup_batch_size: 100,
      internal_registry_temp_cleanup_dry_run: false,
      provider_readiness_watcher_enabled: true,
      provider_readiness_watcher_interval_seconds: 180,
      provider_readiness_watcher_timeout_seconds: 90,
      provider_readiness_watcher_batch_size: 200,
      tenant_asset_reconcile_policy: "full_reconcile_on_prepare",
      tenant_asset_drift_watcher_enabled: true,
      tenant_asset_drift_watcher_interval_seconds: 300,
      tekton_history_cleanup_enabled: true,
      tekton_history_cleanup_schedule: "30 2 * * *",
      tekton_history_cleanup_keep_pipelineruns: 120,
      tekton_history_cleanup_keep_taskruns: 240,
      tekton_history_cleanup_keep_pods: 240,
      image_import_notification_receipt_cleanup_enabled: true,
      image_import_notification_receipt_retention_days: 30,
      image_import_notification_receipt_cleanup_interval_hours: 24,
      storage_profiles: {
        internal_registry: {
          type: "hostPath",
          host_path: "/var/lib/image-factory/registry",
          host_path_type: "DirectoryOrCreate",
          pvc_name: "image-factory-registry-data",
          pvc_size: "20Gi",
          pvc_storage_class: "",
          pvc_access_modes: ["ReadWriteOnce"],
        },
        trivy_cache: {
          type: "emptyDir",
          host_path: "",
          host_path_type: "DirectoryOrCreate",
          pvc_name: "image-factory-trivy-cache",
          pvc_size: "10Gi",
          pvc_storage_class: "",
          pvc_access_modes: ["ReadWriteOnce"],
        },
      },
    });
  const [runtimeServicesFieldErrors, setRuntimeServicesFieldErrors] = useState<{
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
  }>({});
  const [runtimeServicesSubTab, setRuntimeServicesSubTab] =
    useState<RuntimeServicesSubTab>("services");
  const [tektonSubTab, setTektonSubTab] = useState<TektonSubTab>("images");
  const [quarantinePolicyScope, setQuarantinePolicyScope] = useState<
    "global" | "tenant"
  >("global");
  const [quarantinePolicyTenantID, setQuarantinePolicyTenantID] =
    useState<string>("");
  const [quarantinePolicyLoading, setQuarantinePolicyLoading] =
    useState<boolean>(false);
  const [quarantinePolicyConfig, setQuarantinePolicyConfig] =
    useState<QuarantinePolicyFormData>({
      enabled: true,
      mode: "dry_run",
      max_critical: 0,
      max_p2: 0,
      max_p3: 0,
      max_cvss: 0,
      severity_mapping: {
        p1: ["critical"],
        p2: ["high"],
        p3: ["medium"],
        p4: ["low", "unknown"],
      },
    });
  const [quarantinePolicyValidation, setQuarantinePolicyValidation] =
    useState<QuarantinePolicyValidationResult | null>(null);
  const [quarantinePolicySimulationInput, setQuarantinePolicySimulationInput] =
    useState({
      critical: 0,
      high: 0,
      medium: 0,
      maxCVSS: 0,
    });
  const [quarantinePolicySimulationResult, setQuarantinePolicySimulationResult] =
    useState<QuarantinePolicySimulationResult | null>(null);
  const [sorRegistrationScope, setSorRegistrationScope] = useState<
    "global" | "tenant"
  >("global");
  const [sorRegistrationTenantID, setSorRegistrationTenantID] =
    useState<string>("");
  const [sorRegistrationLoading, setSorRegistrationLoading] =
    useState<boolean>(false);
  const [sorRegistrationConfig, setSorRegistrationConfig] =
    useState<SORRegistrationFormData>({
      enforce: true,
      runtime_error_mode: "error",
    });

  const [purgeLogs, setPurgeLogs] = useState<AuditEvent[]>([]);
  const [purgeLogsLoading, setPurgeLogsLoading] = useState(false);
  const [purgeLogsLoaded, setPurgeLogsLoaded] = useState(false);
  const [purgeLogsError, setPurgeLogsError] = useState<string | null>(null);
  const [showMessagingTooltip, setShowMessagingTooltip] = useState(false);
  const [rebooting, setRebooting] = useState(false);

  const RestartRequiredBadge: React.FC = () => (
    <span className="inline-flex items-center gap-1 rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-200">
      <AlertTriangle className="h-3 w-3" />
      Restart required
    </span>
  );

  // LDAP allowed domains state
  const [newDomain, setNewDomain] = useState("");
  const isProd = import.meta.env.PROD;

  useEffect(() => {
    loadConfigurations();
  }, []);

  useEffect(() => {
    if (activeTab !== "general" || purgeLogsLoaded) {
      return;
    }

    loadPurgeLogs();
  }, [activeTab, purgeLogsLoaded]);

  useEffect(() => {
    if (activeTab !== "quarantine_policy") {
      return;
    }
    if (quarantinePolicyScope === "tenant" && !quarantinePolicyTenantID.trim()) {
      return;
    }
    loadQuarantinePolicyConfig();
  }, [activeTab, quarantinePolicyScope, quarantinePolicyTenantID]);

  useEffect(() => {
    if (activeTab !== "sor_registration") {
      return;
    }
    if (sorRegistrationScope === "tenant" && !sorRegistrationTenantID.trim()) {
      return;
    }
    loadSORRegistrationConfig();
  }, [activeTab, sorRegistrationScope, sorRegistrationTenantID]);

  const loadConfigurations = async () => {
    try {
      setLoading(true);
      const response = await api.get("/system-configs");
      const data = response.data;
      // setConfigs(data.configs || [])

      // Parse build configurations
      const buildConfigData = data.configs.find(
        (c: SystemConfig) =>
          c.config_type === "build" && c.config_key === "build",
      );
      if (buildConfigData && buildConfigData.config_value) {
        setBuildConfig((prev) => ({
          ...prev,
          ...buildConfigData.config_value,
        }));
      }

      // Parse security configurations
      const securityConfigData = data.configs.find(
        (c: SystemConfig) =>
          c.config_type === "security" && c.config_key === "security",
      );
      if (securityConfigData && securityConfigData.config_value) {
        setSecurityConfig((prev) => ({
          ...prev,
          ...securityConfigData.config_value,
        }));
      }

      // Parse LDAP configuration (support any key; prefer active+enabled, then active, then first)
      const ldapConfigs = (data.configs || []).filter(
        (c: SystemConfig) => c.config_type === "ldap",
      );
      const activeEnabledLdap = ldapConfigs.find(
        (c: SystemConfig) =>
          c.status === "active" && c.config_value?.enabled === true,
      );
      const activeLdap = ldapConfigs.find(
        (c: SystemConfig) => c.status === "active",
      );
      const ldapConfigData = activeEnabledLdap || activeLdap || ldapConfigs[0];
      if (ldapConfigData && ldapConfigData.config_value) {
        setLdapConfigKey(ldapConfigData.config_key || "ldap_active_directory");
        setLdapConfig((prev) => ({
          ...prev,
          ...ldapConfigData.config_value,
          allowed_domains: ldapConfigData.config_value.allowed_domains || [],
        }));
      }

      // Parse SMTP configurations
      const smtpConfigData = data.configs.find(
        (c: SystemConfig) =>
          c.config_type === "smtp" && c.config_key === "smtp",
      );
      if (smtpConfigData && smtpConfigData.config_value) {
        setSmtpConfig((prev) => ({ ...prev, ...smtpConfigData.config_value }));
      }

      // Parse general configurations
      const generalConfigData = data.configs.find(
        (c: SystemConfig) =>
          c.config_type === "general" && c.config_key === "general",
      );
      if (generalConfigData && generalConfigData.config_value) {
        setGeneralConfig((prev) => ({
          ...prev,
          ...generalConfigData.config_value,
        }));
      }

      // Parse messaging configurations
      const messagingConfigData = data.configs.find(
        (c: SystemConfig) =>
          c.config_type === "messaging" && c.config_key === "messaging",
      );
      if (messagingConfigData && messagingConfigData.config_value) {
        setMessagingConfig((prev) => ({
          ...prev,
          ...messagingConfigData.config_value,
        }));
      }

      const runtimeServicesConfigData = data.configs.find(
        (c: SystemConfig) =>
          c.config_type === "runtime_services" &&
          c.config_key === "runtime_services",
      );
      if (runtimeServicesConfigData && runtimeServicesConfigData.config_value) {
        setRuntimeServicesConfig((prev) => ({
          ...prev,
          ...runtimeServicesConfigData.config_value,
          storage_profiles: {
            ...prev.storage_profiles,
            ...(runtimeServicesConfigData.config_value.storage_profiles || {}),
            internal_registry: {
              ...prev.storage_profiles.internal_registry,
              ...(runtimeServicesConfigData.config_value.storage_profiles
                ?.internal_registry || {}),
            },
            trivy_cache: {
              ...prev.storage_profiles.trivy_cache,
              ...(runtimeServicesConfigData.config_value.storage_profiles
                ?.trivy_cache || {}),
            },
          },
        }));
      }

      const tektonCoreConfigData = data.configs.find(
        (c: SystemConfig) =>
          c.config_type === "tekton" && c.config_key === "tekton_core",
      );
      if (tektonCoreConfigData && tektonCoreConfigData.config_value) {
        setTektonCoreConfig((prev) => ({
          ...prev,
          ...tektonCoreConfigData.config_value,
          manifest_urls: tektonCoreConfigData.config_value.manifest_urls || [],
        }));
      }
      try {
        const tektonTaskImagesResponse = await adminService.getTektonTaskImages();
        if (tektonTaskImagesResponse) {
          setTektonTaskImages((prev) => ({
            ...prev,
            ...tektonTaskImagesResponse,
          }));
        }
      } catch (tektonTaskImagesError) {
        console.warn(
          "Failed to load tekton task images configuration",
          tektonTaskImagesError,
        );
      }
    } catch (error) {
      toast.error("Failed to load system configurations");
    } finally {
      setLoading(false);
    }
  };

  const saveBuildConfig = async () => {
    try {
      await api.post("/system-configs/category", {
        config_type: "build",
        config_key: "build",
        config_value: buildConfig,
      });
      toast.success("Build configuration saved successfully");
      loadConfigurations();
    } catch (error) {
      toast.error("Failed to save build configuration");
    }
  };

  const saveTektonCoreConfig = async () => {
    try {
      await api.post("/system-configs/category", {
        config_type: "tekton",
        config_key: "tekton_core",
        config_value: tektonCoreConfig,
      });
      toast.success("Tekton configuration saved successfully");
      loadConfigurations();
    } catch (error) {
      toast.error("Failed to save Tekton configuration");
    }
  };

  const saveTektonTaskImagesConfig = async () => {
    try {
      await adminService.updateTektonTaskImages(tektonTaskImages);
      toast.success("Tekton task images saved successfully");
      loadConfigurations();
    } catch (error: any) {
      const message =
        error?.response?.data?.message ||
        error?.response?.data?.error ||
        error?.message ||
        "Failed to save Tekton task images";
      toast.error(message);
    }
  };

  const saveSecurityConfig = async () => {
    try {
      await api.post("/system-configs/category", {
        config_type: "security",
        config_key: "security",
        config_value: securityConfig,
      });
      toast.success("Security configuration saved successfully");
      loadConfigurations();
    } catch (error) {
      toast.error("Failed to save security configuration");
    }
  };

  const saveLdapConfig = async () => {
    try {
      await api.post("/system-configs/category", {
        config_type: "ldap",
        config_key: ldapConfigKey || "ldap_active_directory",
        config_value: ldapConfig,
      });
      toast.success("LDAP configuration saved successfully");
      loadConfigurations();
    } catch (error) {
      toast.error("Failed to save LDAP configuration");
    }
  };

  const saveSmtpConfig = async () => {
    try {
      await api.post("/system-configs/category", {
        config_type: "smtp",
        config_key: "smtp",
        config_value: smtpConfig,
      });
      toast.success("SMTP configuration saved successfully");
      loadConfigurations();
    } catch (error) {
      toast.error("Failed to save SMTP configuration");
    }
  };

  const saveGeneralConfig = async () => {
    try {
      await api.post("/system-configs/category", {
        config_type: "general",
        config_key: "general",
        config_value: generalConfig,
      });
      toast.success("General configuration saved successfully");
      loadConfigurations();
    } catch (error) {
      toast.error("Failed to save general configuration");
    }
  };

  const saveMessagingConfig = async () => {
    try {
      await api.post("/system-configs/category", {
        config_type: "messaging",
        config_key: "messaging",
        config_value: messagingConfig,
      });
      toast.success("Messaging configuration saved successfully");
      loadConfigurations();
    } catch (error) {
      toast.error("Failed to save messaging configuration");
    }
  };

  const saveRuntimeServicesConfig = async () => {
    try {
      setRuntimeServicesFieldErrors({});
      await api.post("/system-configs/category", {
        config_type: "runtime_services",
        config_key: "runtime_services",
        config_value: runtimeServicesConfig,
      });
      toast.success("Runtime services configuration saved successfully");
      loadConfigurations();
    } catch (error: any) {
      const fieldErrorsFromAPI =
        error?.response?.data?.field_errors &&
        typeof error.response.data.field_errors === "object"
          ? error.response.data.field_errors
          : {};
      const message =
        error?.response?.data?.message ||
        error?.response?.data?.error ||
        "Failed to save runtime services configuration";
      const nextErrors: {
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
      } = {};
      if (
        typeof fieldErrorsFromAPI.dispatcher_url === "string"
      ) {
        nextErrors.dispatcher_url = fieldErrorsFromAPI.dispatcher_url;
      }
      if (
        typeof fieldErrorsFromAPI.dispatcher_port ===
        "string"
      ) {
        nextErrors.dispatcher_port =
          fieldErrorsFromAPI.dispatcher_port;
      }
      if (
        typeof fieldErrorsFromAPI.email_worker_url === "string"
      ) {
        nextErrors.email_worker_url = fieldErrorsFromAPI.email_worker_url;
      }
      if (
        typeof fieldErrorsFromAPI.email_worker_port ===
        "string"
      ) {
        nextErrors.email_worker_port =
          fieldErrorsFromAPI.email_worker_port;
      }
      if (
        typeof fieldErrorsFromAPI.notification_worker_url === "string"
      ) {
        nextErrors.notification_worker_url =
          fieldErrorsFromAPI.notification_worker_url;
      }
      if (
        typeof fieldErrorsFromAPI.notification_worker_port ===
        "string"
      ) {
        nextErrors.notification_worker_port =
          fieldErrorsFromAPI.notification_worker_port;
      }
      if (
        typeof fieldErrorsFromAPI.internal_registry_gc_worker_url === "string"
      ) {
        nextErrors.internal_registry_gc_worker_url =
          fieldErrorsFromAPI.internal_registry_gc_worker_url;
      }
      if (
        typeof fieldErrorsFromAPI.internal_registry_gc_worker_port === "string"
      ) {
        nextErrors.internal_registry_gc_worker_port =
          fieldErrorsFromAPI.internal_registry_gc_worker_port;
      }
      if (
        typeof fieldErrorsFromAPI.health_check_timeout_seconds ===
        "string"
      ) {
        nextErrors.health_check_timeout_seconds =
          fieldErrorsFromAPI.health_check_timeout_seconds;
      }
      if (
        typeof fieldErrorsFromAPI.internal_registry_temp_cleanup_retention_hours ===
        "string"
      ) {
        nextErrors.internal_registry_temp_cleanup_retention_hours =
          fieldErrorsFromAPI.internal_registry_temp_cleanup_retention_hours;
      }
      if (
        typeof fieldErrorsFromAPI.internal_registry_temp_cleanup_interval_minutes ===
        "string"
      ) {
        nextErrors.internal_registry_temp_cleanup_interval_minutes =
          fieldErrorsFromAPI.internal_registry_temp_cleanup_interval_minutes;
      }
      if (
        typeof fieldErrorsFromAPI.internal_registry_temp_cleanup_batch_size ===
        "string"
      ) {
        nextErrors.internal_registry_temp_cleanup_batch_size =
          fieldErrorsFromAPI.internal_registry_temp_cleanup_batch_size;
      }
      if (
        typeof fieldErrorsFromAPI.tekton_history_cleanup_schedule === "string"
      ) {
        nextErrors.tekton_history_cleanup_schedule =
          fieldErrorsFromAPI.tekton_history_cleanup_schedule;
      }
      if (
        typeof fieldErrorsFromAPI.image_import_notification_receipt_retention_days ===
        "string"
      ) {
        nextErrors.image_import_notification_receipt_retention_days =
          fieldErrorsFromAPI.image_import_notification_receipt_retention_days;
      }
      if (
        typeof fieldErrorsFromAPI.image_import_notification_receipt_cleanup_interval_hours ===
        "string"
      ) {
        nextErrors.image_import_notification_receipt_cleanup_interval_hours =
          fieldErrorsFromAPI.image_import_notification_receipt_cleanup_interval_hours;
      }
      if (
        typeof fieldErrorsFromAPI.provider_readiness_watcher_interval_seconds ===
        "string"
      ) {
        nextErrors.provider_readiness_watcher_interval_seconds =
          fieldErrorsFromAPI.provider_readiness_watcher_interval_seconds;
      }
      if (
        typeof fieldErrorsFromAPI.provider_readiness_watcher_timeout_seconds ===
        "string"
      ) {
        nextErrors.provider_readiness_watcher_timeout_seconds =
          fieldErrorsFromAPI.provider_readiness_watcher_timeout_seconds;
      }
      if (
        typeof fieldErrorsFromAPI.provider_readiness_watcher_batch_size ===
        "string"
      ) {
        nextErrors.provider_readiness_watcher_batch_size =
          fieldErrorsFromAPI.provider_readiness_watcher_batch_size;
      }
      if (
        typeof fieldErrorsFromAPI.tekton_history_cleanup_keep_pipelineruns ===
        "string"
      ) {
        nextErrors.tekton_history_cleanup_keep_pipelineruns =
          fieldErrorsFromAPI.tekton_history_cleanup_keep_pipelineruns;
      }
      if (
        typeof fieldErrorsFromAPI.tekton_history_cleanup_keep_taskruns ===
        "string"
      ) {
        nextErrors.tekton_history_cleanup_keep_taskruns =
          fieldErrorsFromAPI.tekton_history_cleanup_keep_taskruns;
      }
      if (
        typeof fieldErrorsFromAPI.tekton_history_cleanup_keep_pods ===
        "string"
      ) {
        nextErrors.tekton_history_cleanup_keep_pods =
          fieldErrorsFromAPI.tekton_history_cleanup_keep_pods;
      }
      if (
        typeof fieldErrorsFromAPI["storage_profiles.internal_registry.type"] ===
        "string"
      ) {
        nextErrors.storage_profiles_internal_registry_type =
          fieldErrorsFromAPI["storage_profiles.internal_registry.type"];
      }
      if (
        typeof fieldErrorsFromAPI[
          "storage_profiles.internal_registry.host_path"
        ] === "string"
      ) {
        nextErrors.storage_profiles_internal_registry_host_path =
          fieldErrorsFromAPI["storage_profiles.internal_registry.host_path"];
      }
      if (
        typeof fieldErrorsFromAPI[
          "storage_profiles.internal_registry.host_path_type"
        ] === "string"
      ) {
        nextErrors.storage_profiles_internal_registry_host_path_type =
          fieldErrorsFromAPI[
            "storage_profiles.internal_registry.host_path_type"
          ];
      }
      if (
        typeof fieldErrorsFromAPI[
          "storage_profiles.internal_registry.pvc_name"
        ] === "string"
      ) {
        nextErrors.storage_profiles_internal_registry_pvc_name =
          fieldErrorsFromAPI["storage_profiles.internal_registry.pvc_name"];
      }
      if (
        typeof fieldErrorsFromAPI[
          "storage_profiles.internal_registry.pvc_access_modes.0"
        ] === "string"
      ) {
        nextErrors.storage_profiles_internal_registry_pvc_access_modes_0 =
          fieldErrorsFromAPI[
            "storage_profiles.internal_registry.pvc_access_modes.0"
          ];
      }
      setRuntimeServicesFieldErrors(nextErrors);
      toast.error(message);
    }
  };

  const quarantinePolicyRequestParams = () => {
    if (quarantinePolicyScope === "global") {
      return { all_tenants: true };
    }
    return { tenant_id: quarantinePolicyTenantID.trim() };
  };

  const loadQuarantinePolicyConfig = async () => {
    if (quarantinePolicyScope === "tenant" && !quarantinePolicyTenantID.trim()) {
      toast.error("Enter a tenant ID to load tenant policy override");
      return;
    }
    try {
      setQuarantinePolicyLoading(true);
      const response = await api.get("/admin/settings/quarantine-policy", {
        params: quarantinePolicyRequestParams(),
      });
      setQuarantinePolicyConfig((prev) => ({
        ...prev,
        ...response.data,
        severity_mapping: {
          p1: response.data?.severity_mapping?.p1 || prev.severity_mapping.p1,
          p2: response.data?.severity_mapping?.p2 || prev.severity_mapping.p2,
          p3: response.data?.severity_mapping?.p3 || prev.severity_mapping.p3,
          p4: response.data?.severity_mapping?.p4 || prev.severity_mapping.p4,
        },
      }));
      toast.success("Quarantine policy loaded");
    } catch {
      toast.error("Failed to load quarantine policy configuration");
    } finally {
      setQuarantinePolicyLoading(false);
    }
  };

  const saveQuarantinePolicyConfig = async () => {
    if (quarantinePolicyScope === "tenant" && !quarantinePolicyTenantID.trim()) {
      toast.error("Enter a tenant ID to save tenant policy override");
      return;
    }
    try {
      await api.put(
        "/admin/settings/quarantine-policy",
        quarantinePolicyConfig,
        {
          params: quarantinePolicyRequestParams(),
        },
      );
      toast.success("Quarantine policy configuration saved successfully");
      await loadQuarantinePolicyConfig();
    } catch (error: any) {
      const message =
        error?.response?.data?.message ||
        error?.response?.data?.error ||
        "Failed to save quarantine policy configuration";
      toast.error(message);
    }
  };

  const validateQuarantinePolicyConfig = async () => {
    try {
      const response = await api.post(
        "/admin/settings/quarantine-policy/validate",
        {
          policy: quarantinePolicyConfig,
        },
      );
      setQuarantinePolicyValidation(response.data);
      if (response.data?.valid) {
        toast.success("Quarantine policy is valid");
      } else {
        toast.error("Quarantine policy validation failed");
      }
    } catch (error: any) {
      const message =
        error?.response?.data?.errors?.[0] ||
        error?.response?.data?.message ||
        "Failed to validate quarantine policy";
      setQuarantinePolicyValidation({
        valid: false,
        errors: [message],
        warnings: [],
      });
      toast.error(message);
    }
  };

  const simulateQuarantinePolicyConfig = async () => {
    try {
      const response = await api.post(
        "/admin/settings/quarantine-policy/simulate",
        {
          policy: quarantinePolicyConfig,
          scan_summary: {
            vulnerabilities: {
              critical: quarantinePolicySimulationInput.critical,
              high: quarantinePolicySimulationInput.high,
              medium: quarantinePolicySimulationInput.medium,
            },
            max_cvss: quarantinePolicySimulationInput.maxCVSS,
          },
        },
      );
      setQuarantinePolicySimulationResult(response.data);
      toast.success("Policy simulation completed");
    } catch (error: any) {
      const message =
        error?.response?.data?.message ||
        error?.response?.data?.error ||
        "Failed to simulate quarantine policy";
      toast.error(message);
    }
  };

  const sorRegistrationRequestParams = () => {
    if (sorRegistrationScope === "global") {
      return { all_tenants: true };
    }
    return { tenant_id: sorRegistrationTenantID.trim() };
  };

  const loadSORRegistrationConfig = async () => {
    if (sorRegistrationScope === "tenant" && !sorRegistrationTenantID.trim()) {
      toast.error("Enter a tenant ID to load tenant SOR override");
      return;
    }
    try {
      setSorRegistrationLoading(true);
      const response = await api.get("/admin/settings/epr-registration", {
        params: sorRegistrationRequestParams(),
      });
      setSorRegistrationConfig((prev) => ({
        ...prev,
        ...response.data,
      }));
      toast.success("EPR registration policy loaded");
    } catch {
      toast.error("Failed to load EPR registration policy");
    } finally {
      setSorRegistrationLoading(false);
    }
  };

  const saveSORRegistrationConfig = async () => {
    if (sorRegistrationScope === "tenant" && !sorRegistrationTenantID.trim()) {
      toast.error("Enter a tenant ID to save tenant SOR override");
      return;
    }
    try {
      await api.put(
        "/admin/settings/epr-registration",
        sorRegistrationConfig,
        {
          params: sorRegistrationRequestParams(),
        },
      );
      toast.success("EPR registration policy saved successfully");
      await loadSORRegistrationConfig();
    } catch (error: any) {
      const message =
        error?.response?.data?.message ||
        error?.response?.data?.error ||
        "Failed to save EPR registration policy";
      toast.error(message);
    }
  };

  const handlePurgeDeletedProjects = async () => {
    const confirmed = window.confirm(
      "Purge all soft-deleted projects older than the retention window? This action cannot be undone.",
    );
    if (!confirmed) {
      return;
    }

    try {
      const response = await api.post("/admin/projects/purge-deleted", null, {
        params: { retention_days: generalConfig.project_retention_days },
      });
      const count = response.data?.deleted_count ?? 0;
      toast.success(`Purged ${count} deleted project${count === 1 ? "" : "s"}`);
      await loadConfigurations();
      await loadPurgeLogs(true);
    } catch (error) {
      toast.error("Failed to purge deleted projects");
    }
  };

  const loadPurgeLogs = async (force = false) => {
    if (purgeLogsLoading || (purgeLogsLoaded && !force)) {
      return;
    }

    try {
      setPurgeLogsLoading(true);
      setPurgeLogsError(null);
      const response = await auditService.getAuditEvents({
        event_type: "project_purge",
        resource: "projects",
        action: "purge_deleted",
        limit: 10,
        offset: 0,
      });
      setPurgeLogs(response.events || []);
      setPurgeLogsLoaded(true);
    } catch (error) {
      setPurgeLogsError("Failed to load purge logs");
    } finally {
      setPurgeLogsLoading(false);
    }
  };

  const handleRebootServer = async () => {
    if (isProd) {
      toast.error("Reboot is disabled in production");
      return;
    }

    const confirmed = window.confirm(
      "Reboot the server now? Active sessions may be interrupted.",
    );
    if (!confirmed) {
      return;
    }

    try {
      setRebooting(true);
      await api.post("/admin/system/reboot");
      toast.success("Reboot initiated");
    } catch (error) {
      toast.error("Failed to reboot server");
    } finally {
      setRebooting(false);
    }
  };

  const addAllowedDomain = () => {
    const domain = newDomain.trim();
    if (!domain) return;

    // Basic validation: should start with @
    if (!domain.startsWith("@")) {
      toast.error("Domain must start with @ (e.g., @example.com)");
      return;
    }

    // Check if already exists
    if (ldapConfig.allowed_domains.includes(domain)) {
      toast.error("Domain already exists");
      return;
    }

    setLdapConfig((prev) => ({
      ...prev,
      allowed_domains: [...prev.allowed_domains, domain],
    }));
    setNewDomain("");
  };

  const removeAllowedDomain = (domain: string) => {
    setLdapConfig((prev) => ({
      ...prev,
      allowed_domains: prev.allowed_domains.filter((d) => d !== domain),
    }));
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-2">Loading configurations...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {!canManageAdmin && (
        <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
          Read-only mode: configuration save and destructive system actions are hidden for System Administrator Viewer.
        </div>
      )}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          System Configuration
        </h1>
      </div>
      <div className="rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-900/20">
        <h2 className="text-sm font-semibold text-amber-900 dark:text-amber-200">
          Operational Dependency Notice
        </h2>
        <p className="mt-1 text-sm text-amber-800 dark:text-amber-300">
          Dispatcher, Email Worker, and Notification Worker are independent
          services. They must run in parallel with the API for queueing, email
          delivery, and notifications to work.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200 dark:border-slate-700">
        <nav className="-mb-px flex space-x-8 overflow-x-auto">
          <button
            onClick={() => setActiveTab("general")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "general"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            General
          </button>
          <button
            onClick={() => setActiveTab("messaging")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "messaging"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            Messaging
          </button>
          <button
            onClick={() => setActiveTab("runtime_services")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "runtime_services"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            Runtime Services
          </button>
          <button
            onClick={() => setActiveTab("security")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "security"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            Security
          </button>
          <button
            onClick={() => setActiveTab("ldap")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "ldap"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            LDAP
          </button>
          <button
            onClick={() => setActiveTab("smtp")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "smtp"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            SMTP
          </button>
          <button
            onClick={() => setActiveTab("build")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "build"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            Build
          </button>
          <button
            onClick={() => setActiveTab("tekton")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "tekton"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            Tekton
          </button>
          <button
            onClick={() => setActiveTab("quarantine_policy")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "quarantine_policy"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            Quarantine Policy
          </button>
          <button
            onClick={() => setActiveTab("sor_registration")}
            className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
              activeTab === "sor_registration"
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
            }`}
          >
            EPR Registration
          </button>
        </nav>
      </div>

      {/* Build Configuration Tab */}
      {activeTab === "build" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            Build System Settings
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Default Timeout (minutes)
              </label>
              <input
                type="number"
                value={buildConfig.default_timeout_minutes}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    default_timeout_minutes: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Max Concurrent Jobs
              </label>
              <input
                type="number"
                value={buildConfig.max_concurrent_jobs}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    max_concurrent_jobs: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Worker Pool Size
              </label>
              <input
                type="number"
                value={buildConfig.worker_pool_size}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    worker_pool_size: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Max Queue Size
              </label>
              <input
                type="number"
                value={buildConfig.max_queue_size}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    max_queue_size: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Artifact Retention (days)
              </label>
              <input
                type="number"
                value={buildConfig.artifact_retention_days}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    artifact_retention_days: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 px-4 py-3">
              <input
                id="tekton-enabled"
                type="checkbox"
                checked={buildConfig.tekton_enabled}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    tekton_enabled: e.target.checked,
                  }))
                }
                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              <label
                htmlFor="tekton-enabled"
                className="text-sm text-slate-700 dark:text-slate-300"
              >
                Enable Tekton pipeline execution
              </label>
            </div>
            <div className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 px-4 py-3 md:col-span-2 bg-slate-50 dark:bg-slate-900/40">
              <input
                id="build-monitor-event-driven-enabled"
                type="checkbox"
                checked={buildConfig.monitor_event_driven_enabled}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    monitor_event_driven_enabled: e.target.checked,
                  }))
                }
                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              <div>
                <label
                  htmlFor="build-monitor-event-driven-enabled"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Enable event-driven build monitor diagnostics
                </label>
                <div className="mt-1">
                  <RestartRequiredBadge />
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Uses build execution events to drive monitor backoff and diagnostics behavior.
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 px-4 py-3 md:col-span-2 bg-slate-50 dark:bg-slate-900/40">
              <input
                id="build-enable-temp-scan-stage"
                type="checkbox"
                checked={buildConfig.enable_temp_scan_stage}
                onChange={(e) =>
                  setBuildConfig((prev) => ({
                    ...prev,
                    enable_temp_scan_stage: e.target.checked,
                  }))
                }
                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              <div>
                <label
                  htmlFor="build-enable-temp-scan-stage"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Enable temporary internal scan stage
                </label>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Build tar is pushed to internal temporary registry for vulnerability scan and SBOM generation before final publish.
                </p>
              </div>
            </div>
          </div>
          <div className="mt-6">
            {canManageAdmin && (
              <button
                onClick={saveBuildConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save Build Configuration
              </button>
            )}
          </div>
        </div>
      )}

      {/* Tekton Configuration Tab */}
      {activeTab === "tekton" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
              Tekton Core Defaults
            </h2>
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
              Global configuration used when preparing Kubernetes infrastructure
              providers. Provider-level settings override these defaults.
            </p>
          </div>

          <div className="mb-5 flex flex-wrap gap-2">
            {[
              { key: "core", label: "Core Install" },
              { key: "images", label: "Task Images" },
              { key: "tenant", label: "Tenant Assets" },
              { key: "storage", label: "Storage Profiles" },
            ].map((tab) => (
              <button
                key={tab.key}
                type="button"
                onClick={() => setTektonSubTab(tab.key as TektonSubTab)}
                className={`px-3 py-1.5 rounded-md border text-sm font-medium transition-colors ${
                  tektonSubTab === tab.key
                    ? "bg-blue-600 border-blue-600 text-white dark:bg-blue-500 dark:border-blue-500"
                    : "bg-white border-slate-300 text-slate-700 hover:bg-slate-100 dark:bg-slate-700 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-600"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          <div className="space-y-6">
            {tektonSubTab === "core" && (
              <>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Install Source
              </label>
              <select
                value={tektonCoreConfig.install_source}
                onChange={(e) =>
                  setTektonCoreConfig((prev) => ({
                    ...prev,
                    install_source: e.target
                      .value as TektonCoreConfigFormData["install_source"],
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              >
                <option value="manifest">
                  Manifest (apply release YAML URLs)
                </option>
                <option value="helm">
                  Helm (admin installs outside Image Factory)
                </option>
                <option value="preinstalled">
                  Preinstalled (no install attempt)
                </option>
              </select>
              <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                For air-gapped clusters, use preinstalled or point manifest URLs
                to an internal mirror.
              </p>
            </div>

            {tektonCoreConfig.install_source === "manifest" && (
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Manifest URLs (one per line)
                </label>
                <textarea
                  rows={5}
                  value={(tektonCoreConfig.manifest_urls || []).join("\n")}
                  onChange={(e) => {
                    const urls = e.target.value
                      .split("\n")
                      .map((s) => s.trim())
                      .filter(Boolean);
                    setTektonCoreConfig((prev) => ({
                      ...prev,
                      manifest_urls: urls,
                    }));
                  }}
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  placeholder="https://.../release.yaml"
                />
              </div>
            )}

            {tektonCoreConfig.install_source === "helm" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Helm Repo URL (optional)
                  </label>
                  <input
                    type="text"
                    value={tektonCoreConfig.helm_repo_url}
                    onChange={(e) =>
                      setTektonCoreConfig((prev) => ({
                        ...prev,
                        helm_repo_url: e.target.value,
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Chart
                  </label>
                  <input
                    type="text"
                    value={tektonCoreConfig.helm_chart}
                    onChange={(e) =>
                      setTektonCoreConfig((prev) => ({
                        ...prev,
                        helm_chart: e.target.value,
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Release Name
                  </label>
                  <input
                    type="text"
                    value={tektonCoreConfig.helm_release_name}
                    onChange={(e) =>
                      setTektonCoreConfig((prev) => ({
                        ...prev,
                        helm_release_name: e.target.value,
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Namespace
                  </label>
                  <input
                    type="text"
                    value={tektonCoreConfig.helm_namespace}
                    onChange={(e) =>
                      setTektonCoreConfig((prev) => ({
                        ...prev,
                        helm_namespace: e.target.value,
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                </div>
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Assets Directory Override (optional)
              </label>
              <input
                type="text"
                value={tektonCoreConfig.assets_dir}
                onChange={(e) =>
                  setTektonCoreConfig((prev) => ({
                    ...prev,
                    assets_dir: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                placeholder="/opt/image-factory/tekton-assets"
              />
              <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                Path inside the API container where Tekton task/pipeline
                manifests are mounted (must include a kustomization.yaml).
              </p>
            </div>
            <div className="flex justify-end">
              {canManageAdmin && (
                <button
                  onClick={saveTektonCoreConfig}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
                >
                  Save Tekton Configuration
                </button>
              )}
            </div>
              </>
            )}

            {tektonSubTab === "images" && (
            <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-4">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Task Runtime Images
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
                Override task and pipeline step images for air-gapped registries.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {[
                  { key: "git_clone", label: "Git Clone" },
                  { key: "kaniko_executor", label: "Kaniko Executor" },
                  { key: "buildkit", label: "BuildKit" },
                  { key: "skopeo", label: "Skopeo" },
                  { key: "trivy", label: "Trivy" },
                  { key: "syft", label: "Syft" },
                  { key: "cosign", label: "Cosign" },
                  { key: "packer", label: "Packer" },
                  { key: "python_alpine", label: "Python Alpine" },
                  { key: "alpine", label: "Alpine" },
                  { key: "cleanup_kubectl", label: "Cleanup Kubectl" },
                ].map((item) => (
                  <div key={item.key}>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      {item.label}
                    </label>
                    <input
                      type="text"
                      value={
                        tektonTaskImages[
                          item.key as keyof TektonTaskImagesFormData
                        ]
                      }
                      onChange={(e) =>
                        setTektonTaskImages((prev) => ({
                          ...prev,
                          [item.key]: e.target.value,
                        }))
                      }
                      className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                  </div>
                ))}
              </div>
              <div className="mt-4 flex justify-end">
                {canManageAdmin && (
                  <button
                    onClick={saveTektonTaskImagesConfig}
                    className="px-4 py-2 bg-slate-700 hover:bg-slate-800 dark:bg-slate-600 dark:hover:bg-slate-500 text-white rounded-md font-medium transition-colors"
                  >
                    Save Task Images
                  </button>
                )}
              </div>
            </div>
            )}

            {tektonSubTab === "tenant" && (
            <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-4">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Tenant Namespace Asset Reconcile
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
                Controls how provider prepare updates Tekton assets across tenant namespaces.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Reconcile Policy
                  </label>
                  <select
                    value={runtimeServicesConfig.tenant_asset_reconcile_policy}
                    onChange={(e) =>
                      setRuntimeServicesConfig((prev) => ({
                        ...prev,
                        tenant_asset_reconcile_policy: e.target.value as
                          | "full_reconcile_on_prepare"
                          | "async_trigger_only"
                          | "manual_only",
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  >
                    <option value="full_reconcile_on_prepare">
                      Full Reconcile On Prepare
                    </option>
                    <option value="async_trigger_only">Async Trigger Only</option>
                    <option value="manual_only">Manual Only</option>
                  </select>
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Full reconcile applies inline; async queues background updates; manual only detects and reports.
                  </p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Drift Watch Interval (seconds)
                  </label>
                  <input
                    type="number"
                    min={30}
                    value={runtimeServicesConfig.tenant_asset_drift_watcher_interval_seconds}
                    onChange={(e) =>
                      setRuntimeServicesConfig((prev) => ({
                        ...prev,
                        tenant_asset_drift_watcher_interval_seconds:
                          parseInt(e.target.value) || 300,
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Minimum 30 seconds for drift detection cadence.
                  </p>
                </div>
                <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                  <input
                    type="checkbox"
                    checked={runtimeServicesConfig.tenant_asset_drift_watcher_enabled}
                    onChange={(e) =>
                      setRuntimeServicesConfig((prev) => ({
                        ...prev,
                        tenant_asset_drift_watcher_enabled: e.target.checked,
                      }))
                    }
                    className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                  />
                  Tenant asset drift watcher enabled
                </div>
              </div>
              <div className="mt-4">
                {canManageAdmin && (
                  <button
                    onClick={saveRuntimeServicesConfig}
                    className="px-4 py-2 bg-slate-700 hover:bg-slate-800 dark:bg-slate-600 dark:hover:bg-slate-500 text-white rounded-md font-medium transition-colors"
                  >
                    Save Tenant Asset Settings
                  </button>
                )}
              </div>
            </div>
            )}

            {tektonSubTab === "storage" && (
            <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-4">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Storage Profiles
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
                Configure storage backend used by Tekton managed bootstrap assets.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    Internal Registry Storage Type
                  </label>
                  <select
                    value={runtimeServicesConfig.storage_profiles.internal_registry.type}
                    onChange={(e) =>
                      setRuntimeServicesConfig((prev) => ({
                        ...prev,
                        storage_profiles: {
                          ...prev.storage_profiles,
                          internal_registry: {
                            ...prev.storage_profiles.internal_registry,
                            type: e.target.value as "hostPath" | "pvc" | "emptyDir",
                          },
                        },
                      }))
                    }
                    className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                      runtimeServicesFieldErrors.storage_profiles_internal_registry_type
                        ? "border-red-500 dark:border-red-500"
                        : "border-slate-300 dark:border-slate-600"
                    }`}
                  >
                    <option value="hostPath">hostPath</option>
                    <option value="pvc">PVC</option>
                    <option value="emptyDir">emptyDir</option>
                  </select>
                  {runtimeServicesFieldErrors.storage_profiles_internal_registry_type && (
                    <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                      {runtimeServicesFieldErrors.storage_profiles_internal_registry_type}
                    </p>
                  )}
                </div>
                {runtimeServicesConfig.storage_profiles.internal_registry.type ===
                  "hostPath" && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Host Path
                      </label>
                      <input
                        type="text"
                        value={runtimeServicesConfig.storage_profiles.internal_registry.host_path}
                        onChange={(e) =>
                          setRuntimeServicesConfig((prev) => ({
                            ...prev,
                            storage_profiles: {
                              ...prev.storage_profiles,
                              internal_registry: {
                                ...prev.storage_profiles.internal_registry,
                                host_path: e.target.value,
                              },
                            },
                          }))
                        }
                        className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                          runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path
                            ? "border-red-500 dark:border-red-500"
                            : "border-slate-300 dark:border-slate-600"
                        }`}
                      />
                      {runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path && (
                        <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                          {runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path}
                        </p>
                      )}
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Host Path Type
                      </label>
                      <select
                        value={runtimeServicesConfig.storage_profiles.internal_registry.host_path_type}
                        onChange={(e) =>
                          setRuntimeServicesConfig((prev) => ({
                            ...prev,
                            storage_profiles: {
                              ...prev.storage_profiles,
                              internal_registry: {
                                ...prev.storage_profiles.internal_registry,
                                host_path_type: e.target.value,
                              },
                            },
                          }))
                        }
                        className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                          runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path_type
                            ? "border-red-500 dark:border-red-500"
                            : "border-slate-300 dark:border-slate-600"
                        }`}
                      >
                        <option value="DirectoryOrCreate">DirectoryOrCreate</option>
                        <option value="Directory">Directory</option>
                        <option value="FileOrCreate">FileOrCreate</option>
                        <option value="File">File</option>
                      </select>
                      {runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path_type && (
                        <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                          {runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path_type}
                        </p>
                      )}
                    </div>
                  </>
                )}
                {runtimeServicesConfig.storage_profiles.internal_registry.type ===
                  "pvc" && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        PVC Name
                      </label>
                      <input
                        type="text"
                        value={runtimeServicesConfig.storage_profiles.internal_registry.pvc_name}
                        onChange={(e) =>
                          setRuntimeServicesConfig((prev) => ({
                            ...prev,
                            storage_profiles: {
                              ...prev.storage_profiles,
                              internal_registry: {
                                ...prev.storage_profiles.internal_registry,
                                pvc_name: e.target.value,
                              },
                            },
                          }))
                        }
                        className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                          runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_name
                            ? "border-red-500 dark:border-red-500"
                            : "border-slate-300 dark:border-slate-600"
                        }`}
                      />
                      {runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_name && (
                        <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                          {runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_name}
                        </p>
                      )}
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        PVC Size
                      </label>
                      <input
                        type="text"
                        value={runtimeServicesConfig.storage_profiles.internal_registry.pvc_size}
                        onChange={(e) =>
                          setRuntimeServicesConfig((prev) => ({
                            ...prev,
                            storage_profiles: {
                              ...prev.storage_profiles,
                              internal_registry: {
                                ...prev.storage_profiles.internal_registry,
                                pvc_size: e.target.value,
                              },
                            },
                          }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                        placeholder="20Gi"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Storage Class (optional)
                      </label>
                      <input
                        type="text"
                        value={runtimeServicesConfig.storage_profiles.internal_registry.pvc_storage_class}
                        onChange={(e) =>
                          setRuntimeServicesConfig((prev) => ({
                            ...prev,
                            storage_profiles: {
                              ...prev.storage_profiles,
                              internal_registry: {
                                ...prev.storage_profiles.internal_registry,
                                pvc_storage_class: e.target.value,
                              },
                            },
                          }))
                        }
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Access Modes (comma separated)
                      </label>
                      <input
                        type="text"
                        value={runtimeServicesConfig.storage_profiles.internal_registry.pvc_access_modes.join(",")}
                        onChange={(e) =>
                          setRuntimeServicesConfig((prev) => ({
                            ...prev,
                            storage_profiles: {
                              ...prev.storage_profiles,
                              internal_registry: {
                                ...prev.storage_profiles.internal_registry,
                                pvc_access_modes: e.target.value
                                  .split(",")
                                  .map((mode) => mode.trim())
                                  .filter((mode) => mode.length > 0),
                              },
                            },
                          }))
                        }
                        className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                          runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_access_modes_0
                            ? "border-red-500 dark:border-red-500"
                            : "border-slate-300 dark:border-slate-600"
                        }`}
                        placeholder="ReadWriteOnce"
                      />
                      {runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_access_modes_0 && (
                        <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                          {runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_access_modes_0}
                        </p>
                      )}
                    </div>
                  </>
                )}
                {runtimeServicesConfig.storage_profiles.internal_registry.type ===
                  "emptyDir" && (
                  <div className="md:col-span-2 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-200">
                    emptyDir is ephemeral. Internal registry data will be lost when the pod restarts or reschedules.
                  </div>
                )}
              </div>
              <div className="mt-4">
                {canManageAdmin && (
                  <button
                    onClick={saveRuntimeServicesConfig}
                    className="px-4 py-2 bg-slate-700 hover:bg-slate-800 dark:bg-slate-600 dark:hover:bg-slate-500 text-white rounded-md font-medium transition-colors"
                  >
                    Save Storage Profiles
                  </button>
                )}
              </div>
            </div>
            )}
          </div>
        </div>
      )}

      {activeTab === "quarantine_policy" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
              Quarantine Policy
            </h2>
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
              Configure scan gate thresholds and severity mapping used by quarantine evaluation.
            </p>
          </div>

          <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-4 mb-6">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
              Scope
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Policy Scope
                </label>
                <select
                  value={quarantinePolicyScope}
                  onChange={(e) =>
                    setQuarantinePolicyScope(e.target.value as "global" | "tenant")
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                >
                  <option value="global">Global Default</option>
                  <option value="tenant">Tenant Override</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Tenant ID
                </label>
                <input
                  type="text"
                  disabled={quarantinePolicyScope === "global"}
                  value={quarantinePolicyTenantID}
                  onChange={(e) => setQuarantinePolicyTenantID(e.target.value)}
                  placeholder="UUID (required for tenant override)"
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-60 disabled:cursor-not-allowed dark:bg-slate-700 dark:text-white"
                />
              </div>
              <div className="flex gap-2">
                <button
                  onClick={loadQuarantinePolicyConfig}
                  disabled={quarantinePolicyLoading}
                  className="px-4 py-2 bg-slate-700 hover:bg-slate-800 disabled:bg-slate-400 text-white rounded-md font-medium transition-colors"
                >
                  {quarantinePolicyLoading ? "Loading..." : "Load Policy"}
                </button>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 px-4 py-3">
              <input
                id="quarantine-policy-enabled"
                type="checkbox"
                checked={quarantinePolicyConfig.enabled}
                onChange={(e) =>
                  setQuarantinePolicyConfig((prev) => ({
                    ...prev,
                    enabled: e.target.checked,
                  }))
                }
                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              <label
                htmlFor="quarantine-policy-enabled"
                className="text-sm text-slate-700 dark:text-slate-300"
              >
                Enable policy evaluation
              </label>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Mode
              </label>
              <select
                value={quarantinePolicyConfig.mode}
                onChange={(e) =>
                  setQuarantinePolicyConfig((prev) => ({
                    ...prev,
                    mode: e.target.value as "enforce" | "dry_run",
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              >
                <option value="dry_run">dry_run</option>
                <option value="enforce">enforce</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Max Critical
              </label>
              <input
                type="number"
                min={0}
                value={quarantinePolicyConfig.max_critical}
                onChange={(e) =>
                  setQuarantinePolicyConfig((prev) => ({
                    ...prev,
                    max_critical: parseInt(e.target.value || "0", 10),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Max P2
              </label>
              <input
                type="number"
                min={0}
                value={quarantinePolicyConfig.max_p2}
                onChange={(e) =>
                  setQuarantinePolicyConfig((prev) => ({
                    ...prev,
                    max_p2: parseInt(e.target.value || "0", 10),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Max P3
              </label>
              <input
                type="number"
                min={0}
                value={quarantinePolicyConfig.max_p3}
                onChange={(e) =>
                  setQuarantinePolicyConfig((prev) => ({
                    ...prev,
                    max_p3: parseInt(e.target.value || "0", 10),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Max CVSS
              </label>
              <input
                type="number"
                min={0}
                max={10}
                step={0.1}
                value={quarantinePolicyConfig.max_cvss}
                onChange={(e) =>
                  setQuarantinePolicyConfig((prev) => ({
                    ...prev,
                    max_cvss: parseFloat(e.target.value || "0"),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
          </div>

          <div className="mt-6 rounded-md border border-slate-200 dark:border-slate-700 p-4">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
              Severity Mapping (comma-separated)
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {(["p1", "p2", "p3", "p4"] as const).map((priority) => (
                <div key={priority}>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    {priority.toUpperCase()}
                  </label>
                  <input
                    type="text"
                    value={quarantinePolicyConfig.severity_mapping[priority].join(", ")}
                    onChange={(e) =>
                      setQuarantinePolicyConfig((prev) => ({
                        ...prev,
                        severity_mapping: {
                          ...prev.severity_mapping,
                          [priority]: e.target.value
                            .split(",")
                            .map((value) => value.trim().toLowerCase())
                            .filter(Boolean),
                        },
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    placeholder="critical, high, medium, low, unknown"
                  />
                </div>
              ))}
            </div>
          </div>

          <div className="mt-6 rounded-md border border-slate-200 dark:border-slate-700 p-4">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
              Validate & Simulate
            </h3>
            <div className="flex flex-wrap gap-2 mb-4">
              <button
                onClick={validateQuarantinePolicyConfig}
                className="px-3 py-2 bg-slate-700 hover:bg-slate-800 text-white rounded-md text-sm font-medium transition-colors"
              >
                Validate Policy
              </button>
              <button
                onClick={simulateQuarantinePolicyConfig}
                className="px-3 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-md text-sm font-medium transition-colors"
              >
                Simulate Decision
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-4 gap-3 mb-4">
              <div>
                <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Critical Count
                </label>
                <input
                  type="number"
                  min={0}
                  value={quarantinePolicySimulationInput.critical}
                  onChange={(e) =>
                    setQuarantinePolicySimulationInput((prev) => ({
                      ...prev,
                      critical: parseInt(e.target.value || "0", 10),
                    }))
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  High Count
                </label>
                <input
                  type="number"
                  min={0}
                  value={quarantinePolicySimulationInput.high}
                  onChange={(e) =>
                    setQuarantinePolicySimulationInput((prev) => ({
                      ...prev,
                      high: parseInt(e.target.value || "0", 10),
                    }))
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Medium Count
                </label>
                <input
                  type="number"
                  min={0}
                  value={quarantinePolicySimulationInput.medium}
                  onChange={(e) =>
                    setQuarantinePolicySimulationInput((prev) => ({
                      ...prev,
                      medium: parseInt(e.target.value || "0", 10),
                    }))
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Max CVSS
                </label>
                <input
                  type="number"
                  min={0}
                  max={10}
                  step={0.1}
                  value={quarantinePolicySimulationInput.maxCVSS}
                  onChange={(e) =>
                    setQuarantinePolicySimulationInput((prev) => ({
                      ...prev,
                      maxCVSS: parseFloat(e.target.value || "0"),
                    }))
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                />
              </div>
            </div>

            {quarantinePolicyValidation && (
              <div className="mb-3 rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                <p className="text-xs font-medium text-slate-800 dark:text-slate-200">
                  Validation: {quarantinePolicyValidation.valid ? "valid" : "invalid"}
                </p>
                {quarantinePolicyValidation.errors?.length > 0 && (
                  <ul className="mt-1 text-xs text-rose-700 dark:text-rose-300 list-disc list-inside">
                    {quarantinePolicyValidation.errors.map((error, idx) => (
                      <li key={`${error}-${idx}`}>{error}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {quarantinePolicySimulationResult && (
              <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                <p className="text-xs font-medium text-slate-800 dark:text-slate-200">
                  Simulation Decision: {quarantinePolicySimulationResult.decision}
                </p>
                <p className="text-xs text-slate-600 dark:text-slate-300">
                  Mode: {quarantinePolicySimulationResult.mode}
                </p>
                {quarantinePolicySimulationResult.reasons?.length > 0 && (
                  <ul className="mt-1 text-xs text-slate-700 dark:text-slate-300 list-disc list-inside">
                    {quarantinePolicySimulationResult.reasons.map((reason, idx) => (
                      <li key={`${reason}-${idx}`}>{reason}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </div>

          <div className="mt-6 flex justify-end">
            {canManageAdmin && (
              <button
                onClick={saveQuarantinePolicyConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save Quarantine Policy
              </button>
            )}
          </div>
        </div>
      )}

      {activeTab === "sor_registration" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
              EPR Registration Policy
            </h2>
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
              Configure whether EPR registration is enforced and how runtime integration failures are handled.
            </p>
          </div>

          <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-4 mb-6">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
              Scope
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Policy Scope
                </label>
                <select
                  value={sorRegistrationScope}
                  onChange={(e) =>
                    setSorRegistrationScope(e.target.value as "global" | "tenant")
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                >
                  <option value="global">Global Default</option>
                  <option value="tenant">Tenant Override</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Tenant ID
                </label>
                <input
                  type="text"
                  disabled={sorRegistrationScope === "global"}
                  value={sorRegistrationTenantID}
                  onChange={(e) => setSorRegistrationTenantID(e.target.value)}
                  placeholder="UUID (required for tenant override)"
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-60 disabled:cursor-not-allowed dark:bg-slate-700 dark:text-white"
                />
              </div>
              <div className="flex gap-2">
                <button
                  onClick={loadSORRegistrationConfig}
                  disabled={sorRegistrationLoading}
                  className="px-4 py-2 bg-slate-700 hover:bg-slate-800 disabled:bg-slate-400 text-white rounded-md font-medium transition-colors"
                >
                  {sorRegistrationLoading ? "Loading..." : "Load Policy"}
                </button>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 px-4 py-3">
              <input
                id="sor-registration-enforce"
                type="checkbox"
                checked={sorRegistrationConfig.enforce}
                onChange={(e) =>
                  setSorRegistrationConfig((prev) => ({
                    ...prev,
                    enforce: e.target.checked,
                  }))
                }
                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              <label
                htmlFor="sor-registration-enforce"
                className="text-sm text-slate-700 dark:text-slate-300"
              >
                Enforce EPR registration prerequisite
              </label>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Runtime Error Mode
              </label>
              <select
                value={sorRegistrationConfig.runtime_error_mode}
                onChange={(e) =>
                  setSorRegistrationConfig((prev) => ({
                    ...prev,
                    runtime_error_mode: e.target.value as
                      | "error"
                      | "deny"
                      | "allow",
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              >
                <option value="error">error (surface runtime failures)</option>
                <option value="deny">deny (treat runtime failures as not registered)</option>
                <option value="allow">allow (bypass gate on runtime failures)</option>
              </select>
            </div>
          </div>

          <div className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-3 dark:border-amber-700 dark:bg-amber-900/20">
            <p className="text-xs text-amber-800 dark:text-amber-300">
              Recommended: keep <code>runtime_error_mode=error</code> to avoid masking integration outages.
            </p>
          </div>

          <div className="mt-6 flex justify-end">
            {canManageAdmin && (
              <button
                onClick={saveSORRegistrationConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save EPR Registration Policy
              </button>
            )}
          </div>
        </div>
      )}

      {/* Security Configuration Tab */}
      {activeTab === "security" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            Security Settings
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                JWT Expiration (hours)
              </label>
              <input
                type="number"
                value={securityConfig.jwt_expiration_hours}
                onChange={(e) =>
                  setSecurityConfig((prev) => ({
                    ...prev,
                    jwt_expiration_hours: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Refresh Token Expiration (hours)
              </label>
              <input
                type="number"
                value={securityConfig.refresh_token_hours}
                onChange={(e) =>
                  setSecurityConfig((prev) => ({
                    ...prev,
                    refresh_token_hours: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Max Login Attempts
              </label>
              <input
                type="number"
                value={securityConfig.max_login_attempts}
                onChange={(e) =>
                  setSecurityConfig((prev) => ({
                    ...prev,
                    max_login_attempts: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Account Lock Duration (minutes)
              </label>
              <input
                type="number"
                value={securityConfig.account_lock_duration_minutes}
                onChange={(e) =>
                  setSecurityConfig((prev) => ({
                    ...prev,
                    account_lock_duration_minutes: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Password Min Length
              </label>
              <input
                type="number"
                value={securityConfig.password_min_length}
                onChange={(e) =>
                  setSecurityConfig((prev) => ({
                    ...prev,
                    password_min_length: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Session Timeout (minutes)
              </label>
              <input
                type="number"
                value={securityConfig.session_timeout_minutes}
                onChange={(e) =>
                  setSecurityConfig((prev) => ({
                    ...prev,
                    session_timeout_minutes: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div className="md:col-span-2">
              <div className="space-y-3">
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="require_special_chars"
                    checked={securityConfig.require_special_chars}
                    onChange={(e) =>
                      setSecurityConfig((prev) => ({
                        ...prev,
                        require_special_chars: e.target.checked,
                      }))
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
                  />
                  <label
                    htmlFor="require_special_chars"
                    className="ml-2 text-sm text-slate-700 dark:text-slate-300"
                  >
                    Require special characters in passwords
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="require_numbers"
                    checked={securityConfig.require_numbers}
                    onChange={(e) =>
                      setSecurityConfig((prev) => ({
                        ...prev,
                        require_numbers: e.target.checked,
                      }))
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
                  />
                  <label
                    htmlFor="require_numbers"
                    className="ml-2 text-sm text-slate-700 dark:text-slate-300"
                  >
                    Require numbers in passwords
                  </label>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="require_uppercase"
                    checked={securityConfig.require_uppercase}
                    onChange={(e) =>
                      setSecurityConfig((prev) => ({
                        ...prev,
                        require_uppercase: e.target.checked,
                      }))
                    }
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
                  />
                  <label
                    htmlFor="require_uppercase"
                    className="ml-2 text-sm text-slate-700 dark:text-slate-300"
                  >
                    Require uppercase letters in passwords
                  </label>
                </div>
              </div>
            </div>
          </div>
          <div className="mt-6">
            {canManageAdmin && (
              <button
                onClick={saveSecurityConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save Security Configuration
              </button>
            )}
          </div>
        </div>
      )}

      {/* General Configuration Tab */}
      {activeTab === "general" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            General System Settings
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                System Name
              </label>
              <input
                type="text"
                value={generalConfig.system_name}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    system_name: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                System Description
              </label>
              <input
                type="text"
                value={generalConfig.system_description}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    system_description: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Admin Email
              </label>
              <input
                type="email"
                value={generalConfig.admin_email}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    admin_email: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Support Email
              </label>
              <input
                type="email"
                value={generalConfig.support_email}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    support_email: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Time Zone
              </label>
              <input
                type="text"
                value={generalConfig.time_zone}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    time_zone: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Date Format
              </label>
              <input
                type="text"
                value={generalConfig.date_format}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    date_format: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Default Language
              </label>
              <input
                type="text"
                value={generalConfig.default_language}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    default_language: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Project Deletion Retention (days)
              </label>
              <input
                type="number"
                min={0}
                value={generalConfig.project_retention_days}
                onChange={(e) =>
                  setGeneralConfig((prev) => ({
                    ...prev,
                    project_retention_days: parseInt(e.target.value, 10) || 0,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Number of days to retain soft-deleted projects before cleanup.
                Set to 0 to disable automatic purge.
              </p>
              <div className="mt-2 flex items-center gap-3">
                {canManageAdmin && (
                  <button
                    type="button"
                    onClick={handlePurgeDeletedProjects}
                    className="px-3 py-2 text-xs font-medium text-white bg-slate-900 dark:bg-slate-100 dark:text-slate-900 rounded-lg hover:bg-slate-800 dark:hover:bg-white"
                  >
                    Purge Now
                  </button>
                )}
                <span className="text-xs text-slate-500 dark:text-slate-400">
                  Cleanup job runs every 6 hours.
                </span>
              </div>
              <div className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                {generalConfig.project_last_purge_at
                  ? `Last purge: ${new Date(generalConfig.project_last_purge_at).toLocaleString()} (${generalConfig.project_last_purge_count ?? 0} purged)`
                  : "Last purge: never"}
              </div>
            </div>
            <div className="md:col-span-2">
              <div className="flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 px-4 py-3">
                <div>
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                    Maintenance Mode
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    When enabled, write actions are disabled for non-admin
                    users.
                  </p>
                </div>
                <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                  <input
                    type="checkbox"
                    checked={generalConfig.maintenance_mode}
                    onChange={(e) =>
                      setGeneralConfig((prev) => ({
                        ...prev,
                        maintenance_mode: e.target.checked,
                      }))
                    }
                    className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                  />
                  {generalConfig.maintenance_mode ? "Enabled" : "Disabled"}
                </label>
              </div>
            </div>
            <div className="md:col-span-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 px-4 py-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                    Workflow Orchestrator
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Controls internal workflow polling for queued workflow
                    steps.
                  </p>
                </div>
                <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                  <input
                    type="checkbox"
                    checked={generalConfig.workflow_enabled}
                    onChange={(e) =>
                      setGeneralConfig((prev) => ({
                        ...prev,
                        workflow_enabled: e.target.checked,
                      }))
                    }
                    className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                  />
                  {generalConfig.workflow_enabled ? "Enabled" : "Disabled"}
                </label>
              </div>
              <div className="mt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Poll Interval (duration)
                  </label>
                  <input
                    type="text"
                    value={generalConfig.workflow_poll_interval}
                    onChange={(e) =>
                      setGeneralConfig((prev) => ({
                        ...prev,
                        workflow_poll_interval: e.target.value,
                      }))
                    }
                    placeholder="3s"
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Max Steps Per Tick
                  </label>
                  <input
                    type="number"
                    min={1}
                    value={generalConfig.workflow_max_steps_per_tick}
                    onChange={(e) =>
                      setGeneralConfig((prev) => ({
                        ...prev,
                        workflow_max_steps_per_tick: Math.max(
                          1,
                          parseInt(e.target.value, 10) || 1,
                        ),
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Admin Dashboard Poll Interval (seconds)
                  </label>
                  <input
                    type="number"
                    min={5}
                    max={300}
                    value={generalConfig.admin_dashboard_poll_interval_seconds}
                    onChange={(e) =>
                      setGeneralConfig((prev) => ({
                        ...prev,
                        admin_dashboard_poll_interval_seconds: Math.max(
                          5,
                          parseInt(e.target.value, 10) || 15,
                        ),
                      }))
                    }
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  />
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Controls admin execution pipeline health auto-refresh
                    frequency.
                  </p>
                </div>
              </div>
              <div className="mt-3 text-xs text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-md px-3 py-2">
                Changes require backend restart to take effect.
              </div>
            </div>
          </div>
          <div className="mt-6 border border-slate-200 dark:border-slate-700 rounded-lg p-4 bg-slate-50 dark:bg-slate-900/40">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                  Project Purge Logs
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Latest 10 purge actions
                </p>
              </div>
              <button
                type="button"
                onClick={() => loadPurgeLogs(true)}
                disabled={purgeLogsLoading}
                className="px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-slate-600 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-60"
              >
                {purgeLogsLoading ? "Refreshing..." : "Refresh Logs"}
              </button>
            </div>
            <div className="mt-4 space-y-2 text-sm">
              {purgeLogsLoading && (
                <div className="text-slate-500 dark:text-slate-400">
                  Loading purge logs...
                </div>
              )}
              {!purgeLogsLoading && purgeLogsError && (
                <div className="text-red-600 dark:text-red-400">
                  {purgeLogsError}
                </div>
              )}
              {!purgeLogsLoading &&
                !purgeLogsError &&
                purgeLogs.length === 0 && (
                  <div className="text-slate-500 dark:text-slate-400">
                    No purge logs yet.
                  </div>
                )}
              {!purgeLogsLoading && !purgeLogsError && purgeLogs.length > 0 && (
                <div className="divide-y divide-slate-200 dark:divide-slate-700">
                  {purgeLogs.map((event) => (
                    <div
                      key={event.id}
                      className="py-2 flex flex-wrap items-center justify-between gap-3"
                    >
                      <div className="text-slate-700 dark:text-slate-200">
                        <div className="font-medium">
                          {new Date(event.timestamp).toLocaleString()}
                        </div>
                        <div className="text-xs text-slate-500 dark:text-slate-400">
                          {event.message}
                        </div>
                      </div>
                      <div className="text-xs text-slate-500 dark:text-slate-400 text-right">
                        <div>Purged: {event.details?.deleted_count ?? 0}</div>
                        <div>
                          Retention: {event.details?.retention_days ?? "-"} days
                        </div>
                        <div>By: {event.user_name || "System"}</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
          <div className="mt-6 border border-amber-200 dark:border-amber-700/60 rounded-lg p-4 bg-amber-50/70 dark:bg-amber-900/20">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-100">
                  System Actions
                </h3>
                <p className="text-xs text-amber-700 dark:text-amber-200">
                  Non-production only. Use with caution.
                </p>
              </div>
              {canManageAdmin && (
                <button
                  type="button"
                  onClick={handleRebootServer}
                  disabled={isProd || rebooting}
                  className="px-3 py-2 text-xs font-medium text-white bg-amber-600 hover:bg-amber-700 rounded-lg disabled:opacity-60 disabled:cursor-not-allowed"
                >
                  {rebooting ? "Rebooting..." : "Reboot Server"}
                </button>
              )}
            </div>
            {isProd && (
              <div className="mt-2 text-xs text-amber-700 dark:text-amber-200">
                Reboot is disabled in production.
              </div>
            )}
          </div>
          <div className="mt-6">
            {canManageAdmin && (
              <button
                onClick={saveGeneralConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save General Configuration
              </button>
            )}
          </div>
        </div>
      )}

      {/* Messaging Configuration Tab */}
      {activeTab === "messaging" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            Messaging Settings
          </h2>
          <div className="space-y-4">
            <div className="flex items-start gap-3 rounded-md border border-slate-200 dark:border-slate-700 p-3 bg-slate-50 dark:bg-slate-900/40">
              <input
                id="messaging_enable_nats"
                type="checkbox"
                checked={messagingConfig.enable_nats}
                onChange={(e) =>
                  setMessagingConfig((prev) => ({
                    ...prev,
                    enable_nats: e.target.checked,
                  }))
                }
                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
              />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <label
                    htmlFor="messaging_enable_nats"
                    className="text-sm font-medium text-slate-700 dark:text-slate-200"
                  >
                    Enable NATS for unified messaging
                  </label>
                  <RestartRequiredBadge />
                  <div className="relative">
                    <HelpCircle
                      className="h-4 w-4 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 cursor-pointer"
                      onClick={(e) => {
                        e.stopPropagation();
                        setShowMessagingTooltip((prev) => !prev);
                      }}
                    />
                    {showMessagingTooltip && (
                      <div className="absolute left-0 top-6 w-72 p-3 bg-slate-900 dark:bg-slate-700 text-white text-xs rounded-md shadow-lg border border-slate-700 z-10">
                        <div className="flex items-center justify-between mb-2">
                          <div className="font-semibold">NATS Messaging</div>
                          <button
                            type="button"
                            className="text-slate-400 hover:text-white"
                            onClick={(e) => {
                              e.stopPropagation();
                              setShowMessagingTooltip(false);
                            }}
                          >
                            <X className="h-3 w-3" />
                          </button>
                        </div>
                        <div className="space-y-1">
                          <div>Enables NATS-backed event delivery.</div>
                          <div>Requires NATS server + notification worker.</div>
                          <div>Changes take effect after restart.</div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Requires NATS server + notification worker. Changes take
                  effect on service restart.
                </p>
                <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
                  Restart required after saving this setting.
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3 rounded-md border border-slate-200 dark:border-slate-700 p-3 bg-slate-50 dark:bg-slate-900/40">
              <input
                id="messaging_nats_required"
                type="checkbox"
                checked={messagingConfig.nats_required}
                onChange={(e) =>
                  setMessagingConfig((prev) => ({
                    ...prev,
                    nats_required: e.target.checked,
                  }))
                }
                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
              />
              <div className="flex-1">
                <label
                  htmlFor="messaging_nats_required"
                  className="text-sm font-medium text-slate-700 dark:text-slate-200"
                >
                  Require NATS connectivity
                </label>
                <div className="mt-1">
                  <RestartRequiredBadge />
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Fail startup if NATS is unavailable instead of falling back to
                  local-only messaging.
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3 rounded-md border border-slate-200 dark:border-slate-700 p-3 bg-slate-50 dark:bg-slate-900/40">
              <input
                id="messaging_external_only"
                type="checkbox"
                checked={messagingConfig.external_only}
                onChange={(e) =>
                  setMessagingConfig((prev) => ({
                    ...prev,
                    external_only: e.target.checked,
                  }))
                }
                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
              />
              <div className="flex-1">
                <label
                  htmlFor="messaging_external_only"
                  className="text-sm font-medium text-slate-700 dark:text-slate-200"
                >
                  External-only event transport
                </label>
                <div className="mt-1">
                  <RestartRequiredBadge />
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Publish and subscribe through NATS only. Disables local
                  in-process event bus delivery.
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3 rounded-md border border-slate-200 dark:border-slate-700 p-3 bg-slate-50 dark:bg-slate-900/40">
              <input
                id="messaging_outbox_enabled"
                type="checkbox"
                checked={messagingConfig.outbox_enabled}
                onChange={(e) =>
                  setMessagingConfig((prev) => ({
                    ...prev,
                    outbox_enabled: e.target.checked,
                  }))
                }
                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
              />
              <div className="flex-1">
                <label
                  htmlFor="messaging_outbox_enabled"
                  className="text-sm font-medium text-slate-700 dark:text-slate-200"
                >
                  Enable outbox replay
                </label>
                <div className="mt-1">
                  <RestartRequiredBadge />
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Queue events in database if external publish fails and replay
                  them in background.
                </p>
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 rounded-md border border-slate-200 dark:border-slate-700 p-4 bg-slate-50 dark:bg-slate-900/40">
              <div>
                <label
                  htmlFor="messaging_outbox_relay_interval_seconds"
                  className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  Relay Interval (seconds)
                </label>
                <div className="mb-1">
                  <RestartRequiredBadge />
                </div>
                <input
                  id="messaging_outbox_relay_interval_seconds"
                  type="number"
                  min={1}
                  value={messagingConfig.outbox_relay_interval_seconds}
                  onChange={(e) =>
                    setMessagingConfig((prev) => ({
                      ...prev,
                      outbox_relay_interval_seconds:
                        Number.parseInt(e.target.value, 10) || 1,
                    }))
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div>
                <label
                  htmlFor="messaging_outbox_relay_batch_size"
                  className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  Relay Batch Size
                </label>
                <div className="mb-1">
                  <RestartRequiredBadge />
                </div>
                <input
                  id="messaging_outbox_relay_batch_size"
                  type="number"
                  min={1}
                  value={messagingConfig.outbox_relay_batch_size}
                  onChange={(e) =>
                    setMessagingConfig((prev) => ({
                      ...prev,
                      outbox_relay_batch_size:
                        Number.parseInt(e.target.value, 10) || 1,
                    }))
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div>
                <label
                  htmlFor="messaging_outbox_claim_lease_seconds"
                  className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  Claim Lease (seconds)
                </label>
                <div className="mb-1">
                  <RestartRequiredBadge />
                </div>
                <input
                  id="messaging_outbox_claim_lease_seconds"
                  type="number"
                  min={1}
                  value={messagingConfig.outbox_claim_lease_seconds}
                  onChange={(e) =>
                    setMessagingConfig((prev) => ({
                      ...prev,
                      outbox_claim_lease_seconds:
                        Number.parseInt(e.target.value, 10) || 1,
                    }))
                  }
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
            </div>
          </div>
          <div className="mt-6">
            {canManageAdmin && (
              <button
                onClick={saveMessagingConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save Messaging Configuration
              </button>
            )}
          </div>
        </div>
      )}

      {activeTab === "runtime_services" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            Runtime Services
          </h2>
          <p className="text-sm text-slate-600 dark:text-slate-300 mb-4">
            Configure connection details used by administrators to validate
            status and operations for independent runtime services.
          </p>
          <div className="mb-5 flex flex-wrap gap-2">
            {[
              { key: "services", label: "Service Endpoints" },
              { key: "registry_gc", label: "Registry GC" },
              { key: "watchers", label: "Watchers" },
              { key: "cleanup", label: "Cleanup Jobs" },
            ].map((tab) => (
              <button
                key={tab.key}
                type="button"
                onClick={() =>
                  setRuntimeServicesSubTab(tab.key as RuntimeServicesSubTab)
                }
                className={`px-3 py-1.5 rounded-md border text-sm font-medium transition-colors ${
                  runtimeServicesSubTab === tab.key
                    ? "bg-blue-600 border-blue-600 text-white dark:bg-blue-500 dark:border-blue-500"
                    : "bg-white border-slate-300 text-slate-700 hover:bg-slate-100 dark:bg-slate-700 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-600"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {runtimeServicesSubTab === "services" && (
              <>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Dispatcher URL
              </label>
              <input
                type="text"
                value={runtimeServicesConfig.dispatcher_url}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      dispatcher_url: e.target.value,
                    }));
                    if (runtimeServicesFieldErrors.dispatcher_url) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        dispatcher_url: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.dispatcher_url
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.dispatcher_url && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.dispatcher_url}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Dispatcher Port
              </label>
              <input
                type="number"
                min={1}
                max={65535}
                value={runtimeServicesConfig.dispatcher_port}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      dispatcher_port: parseInt(e.target.value) || 8084,
                    }));
                    if (runtimeServicesFieldErrors.dispatcher_port) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        dispatcher_port: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.dispatcher_port
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.dispatcher_port && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.dispatcher_port}
                </p>
              )}
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.dispatcher_mtls_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    dispatcher_mtls_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Dispatcher mTLS enabled
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.workflow_orchestrator_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    workflow_orchestrator_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Workflow orchestrator enabled
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Dispatcher CA Cert (PEM)
              </label>
              <textarea
                value={runtimeServicesConfig.dispatcher_ca_cert}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    dispatcher_ca_cert: e.target.value,
                  }))
                }
                rows={3}
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Email Worker URL
              </label>
              <input
                type="text"
                value={runtimeServicesConfig.email_worker_url}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      email_worker_url: e.target.value,
                    }));
                    if (runtimeServicesFieldErrors.email_worker_url) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        email_worker_url: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.email_worker_url
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.email_worker_url && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.email_worker_url}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Email Worker Port
              </label>
              <input
                type="number"
                min={1}
                max={65535}
                value={runtimeServicesConfig.email_worker_port}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      email_worker_port: parseInt(e.target.value) || 8081,
                    }));
                    if (runtimeServicesFieldErrors.email_worker_port) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        email_worker_port: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.email_worker_port
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.email_worker_port && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.email_worker_port}
                </p>
              )}
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.email_worker_tls_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    email_worker_tls_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Email worker TLS enabled
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Notification Worker URL
              </label>
              <input
                type="text"
                value={runtimeServicesConfig.notification_worker_url}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      notification_worker_url: e.target.value,
                    }));
                    if (runtimeServicesFieldErrors.notification_worker_url) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        notification_worker_url: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.notification_worker_url
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.notification_worker_url && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.notification_worker_url}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Notification Worker Port
              </label>
              <input
                type="number"
                min={1}
                max={65535}
                value={runtimeServicesConfig.notification_worker_port}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      notification_worker_port: parseInt(e.target.value) || 8083,
                    }));
                    if (runtimeServicesFieldErrors.notification_worker_port) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        notification_worker_port: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.notification_worker_port
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.notification_worker_port && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.notification_worker_port}
                </p>
              )}
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.notification_tls_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    notification_tls_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Notification worker TLS enabled
            </div>
              </>
            )}
            {runtimeServicesSubTab === "registry_gc" && (
              <>
            <div className="md:col-span-2 pt-2 border-t border-slate-200 dark:border-slate-700">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Internal Registry GC Worker Endpoint
              </h3>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                GC Worker URL
              </label>
              <input
                type="text"
                value={runtimeServicesConfig.internal_registry_gc_worker_url}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      internal_registry_gc_worker_url: e.target.value,
                    }));
                    if (runtimeServicesFieldErrors.internal_registry_gc_worker_url) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        internal_registry_gc_worker_url: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.internal_registry_gc_worker_url
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.internal_registry_gc_worker_url && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.internal_registry_gc_worker_url}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                GC Worker Port
              </label>
              <input
                type="number"
                min={1}
                max={65535}
                value={runtimeServicesConfig.internal_registry_gc_worker_port}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      internal_registry_gc_worker_port: parseInt(e.target.value) || 8085,
                    }));
                    if (runtimeServicesFieldErrors.internal_registry_gc_worker_port) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        internal_registry_gc_worker_port: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.internal_registry_gc_worker_port
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.internal_registry_gc_worker_port && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.internal_registry_gc_worker_port}
                </p>
              )}
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.internal_registry_gc_worker_tls_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    internal_registry_gc_worker_tls_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Internal Registry GC worker TLS enabled
            </div>
            <div className="md:col-span-2 pt-2 border-t border-slate-200 dark:border-slate-700">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Temp Image Cleanup Policy
              </h3>
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.internal_registry_temp_cleanup_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    internal_registry_temp_cleanup_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Internal registry temp cleanup enabled
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Retention (hours)
              </label>
              <input
                type="number"
                min={1}
                value={runtimeServicesConfig.internal_registry_temp_cleanup_retention_hours}
                onChange={(e) => {
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    internal_registry_temp_cleanup_retention_hours:
                      parseInt(e.target.value) || 72,
                  }));
                  if (
                    runtimeServicesFieldErrors.internal_registry_temp_cleanup_retention_hours
                  ) {
                    setRuntimeServicesFieldErrors((prev) => ({
                      ...prev,
                      internal_registry_temp_cleanup_retention_hours: undefined,
                    }));
                  }
                }}
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.internal_registry_temp_cleanup_retention_hours
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.internal_registry_temp_cleanup_retention_hours && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.internal_registry_temp_cleanup_retention_hours}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Interval (minutes)
              </label>
              <input
                type="number"
                min={1}
                value={runtimeServicesConfig.internal_registry_temp_cleanup_interval_minutes}
                onChange={(e) => {
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    internal_registry_temp_cleanup_interval_minutes:
                      parseInt(e.target.value) || 60,
                  }));
                  if (
                    runtimeServicesFieldErrors.internal_registry_temp_cleanup_interval_minutes
                  ) {
                    setRuntimeServicesFieldErrors((prev) => ({
                      ...prev,
                      internal_registry_temp_cleanup_interval_minutes: undefined,
                    }));
                  }
                }}
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.internal_registry_temp_cleanup_interval_minutes
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.internal_registry_temp_cleanup_interval_minutes && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.internal_registry_temp_cleanup_interval_minutes}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Batch Size
              </label>
              <input
                type="number"
                min={1}
                value={runtimeServicesConfig.internal_registry_temp_cleanup_batch_size}
                onChange={(e) => {
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    internal_registry_temp_cleanup_batch_size:
                      parseInt(e.target.value) || 100,
                  }));
                  if (
                    runtimeServicesFieldErrors.internal_registry_temp_cleanup_batch_size
                  ) {
                    setRuntimeServicesFieldErrors((prev) => ({
                      ...prev,
                      internal_registry_temp_cleanup_batch_size: undefined,
                    }));
                  }
                }}
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.internal_registry_temp_cleanup_batch_size
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.internal_registry_temp_cleanup_batch_size && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.internal_registry_temp_cleanup_batch_size}
                </p>
              )}
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.internal_registry_temp_cleanup_dry_run}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    internal_registry_temp_cleanup_dry_run: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Dry run (log candidates, do not delete)
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Health Timeout (seconds)
              </label>
              <input
                type="number"
                min={1}
                value={runtimeServicesConfig.health_check_timeout_seconds}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      health_check_timeout_seconds: parseInt(e.target.value) || 5,
                    }));
                    if (runtimeServicesFieldErrors.health_check_timeout_seconds) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        health_check_timeout_seconds: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.health_check_timeout_seconds
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.health_check_timeout_seconds && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.health_check_timeout_seconds}
                </p>
              )}
            </div>
              </>
            )}
            {runtimeServicesSubTab === "watchers" && (
              <>
            <div className="md:col-span-2 pt-2 border-t border-slate-200 dark:border-slate-700">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Provider Readiness Watcher
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                Periodically reconciles provider reachability/readiness/schedulability.
                Changes apply without restart. Recommended defaults: interval{" "}
                <span className="font-mono">180s</span>, timeout{" "}
                <span className="font-mono">90s</span>, batch{" "}
                <span className="font-mono">200</span>.
              </p>
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.provider_readiness_watcher_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    provider_readiness_watcher_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Provider readiness watcher enabled
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Watch Interval (seconds)
              </label>
              <input
                type="number"
                min={30}
                value={runtimeServicesConfig.provider_readiness_watcher_interval_seconds}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      provider_readiness_watcher_interval_seconds:
                        parseInt(e.target.value) || 180,
                    }));
                    if (
                      runtimeServicesFieldErrors.provider_readiness_watcher_interval_seconds
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        provider_readiness_watcher_interval_seconds: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.provider_readiness_watcher_interval_seconds
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Minimum 30. Lower values increase API traffic.
              </p>
              {runtimeServicesFieldErrors.provider_readiness_watcher_interval_seconds && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.provider_readiness_watcher_interval_seconds}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Watch Timeout (seconds)
              </label>
              <input
                type="number"
                min={10}
                value={runtimeServicesConfig.provider_readiness_watcher_timeout_seconds}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      provider_readiness_watcher_timeout_seconds:
                        parseInt(e.target.value) || 90,
                    }));
                    if (
                      runtimeServicesFieldErrors.provider_readiness_watcher_timeout_seconds
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        provider_readiness_watcher_timeout_seconds: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.provider_readiness_watcher_timeout_seconds
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Must be less than interval; recommended about half the interval.
              </p>
              {runtimeServicesFieldErrors.provider_readiness_watcher_timeout_seconds && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.provider_readiness_watcher_timeout_seconds}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Watch Batch Size
              </label>
              <input
                type="number"
                min={1}
                max={1000}
                value={runtimeServicesConfig.provider_readiness_watcher_batch_size}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      provider_readiness_watcher_batch_size:
                        parseInt(e.target.value) || 200,
                    }));
                    if (
                      runtimeServicesFieldErrors.provider_readiness_watcher_batch_size
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        provider_readiness_watcher_batch_size: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.provider_readiness_watcher_batch_size
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                1-1000 providers per tick. Use smaller values for very large clusters.
              </p>
              {runtimeServicesFieldErrors.provider_readiness_watcher_batch_size && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.provider_readiness_watcher_batch_size}
                </p>
              )}
            </div>
              </>
            )}
            {runtimeServicesSubTab === "cleanup" && (
              <>
            <div className="md:col-span-2 pt-2 border-t border-slate-200 dark:border-slate-700">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Tekton History Cleanup
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                Controls the per-tenant CronJob that prunes old PipelineRuns, TaskRuns, and Tekton pods.
                Changes apply on next namespace prepare/reconcile.
              </p>
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={runtimeServicesConfig.tekton_history_cleanup_enabled}
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    tekton_history_cleanup_enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Tekton history cleanup CronJob enabled
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Cleanup Schedule (cron)
              </label>
              <input
                type="text"
                value={runtimeServicesConfig.tekton_history_cleanup_schedule}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      tekton_history_cleanup_schedule: e.target.value,
                    }));
                    if (runtimeServicesFieldErrors.tekton_history_cleanup_schedule) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        tekton_history_cleanup_schedule: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.tekton_history_cleanup_schedule
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
                placeholder="30 2 * * *"
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Cron expression interpreted by Kubernetes CronJob scheduler.
              </p>
              {runtimeServicesFieldErrors.tekton_history_cleanup_schedule && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.tekton_history_cleanup_schedule}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Keep PipelineRuns
              </label>
              <input
                type="number"
                min={1}
                value={runtimeServicesConfig.tekton_history_cleanup_keep_pipelineruns}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      tekton_history_cleanup_keep_pipelineruns:
                        parseInt(e.target.value) || 120,
                    }));
                    if (
                      runtimeServicesFieldErrors.tekton_history_cleanup_keep_pipelineruns
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        tekton_history_cleanup_keep_pipelineruns: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.tekton_history_cleanup_keep_pipelineruns
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.tekton_history_cleanup_keep_pipelineruns && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.tekton_history_cleanup_keep_pipelineruns}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Keep TaskRuns
              </label>
              <input
                type="number"
                min={1}
                value={runtimeServicesConfig.tekton_history_cleanup_keep_taskruns}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      tekton_history_cleanup_keep_taskruns:
                        parseInt(e.target.value) || 240,
                    }));
                    if (
                      runtimeServicesFieldErrors.tekton_history_cleanup_keep_taskruns
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        tekton_history_cleanup_keep_taskruns: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.tekton_history_cleanup_keep_taskruns
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.tekton_history_cleanup_keep_taskruns && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.tekton_history_cleanup_keep_taskruns}
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Keep Tekton Pods
              </label>
              <input
                type="number"
                min={1}
                value={runtimeServicesConfig.tekton_history_cleanup_keep_pods}
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      tekton_history_cleanup_keep_pods:
                        parseInt(e.target.value) || 240,
                    }));
                    if (
                      runtimeServicesFieldErrors.tekton_history_cleanup_keep_pods
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        tekton_history_cleanup_keep_pods: undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.tekton_history_cleanup_keep_pods
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              {runtimeServicesFieldErrors.tekton_history_cleanup_keep_pods && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {runtimeServicesFieldErrors.tekton_history_cleanup_keep_pods}
                </p>
              )}
            </div>
            <div className="md:col-span-2 pt-2 border-t border-slate-200 dark:border-slate-700">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                Image Import Notification Receipts
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                Controls retention for persisted idempotency receipts used to
                prevent duplicate quarantine import notifications during event
                replay.
              </p>
            </div>
            <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={
                  runtimeServicesConfig.image_import_notification_receipt_cleanup_enabled
                }
                onChange={(e) =>
                  setRuntimeServicesConfig((prev) => ({
                    ...prev,
                    image_import_notification_receipt_cleanup_enabled:
                      e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              Cleanup worker enabled
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Retention (days)
              </label>
              <input
                type="number"
                min={1}
                max={3650}
                value={
                  runtimeServicesConfig.image_import_notification_receipt_retention_days
                }
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      image_import_notification_receipt_retention_days:
                        parseInt(e.target.value) || 30,
                    }));
                    if (
                      runtimeServicesFieldErrors.image_import_notification_receipt_retention_days
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        image_import_notification_receipt_retention_days:
                          undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.image_import_notification_receipt_retention_days
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Range: 1-3650 days.
              </p>
              {runtimeServicesFieldErrors.image_import_notification_receipt_retention_days && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {
                    runtimeServicesFieldErrors.image_import_notification_receipt_retention_days
                  }
                </p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Cleanup Interval (hours)
              </label>
              <input
                type="number"
                min={1}
                max={168}
                value={
                  runtimeServicesConfig.image_import_notification_receipt_cleanup_interval_hours
                }
                onChange={(e) =>
                  {
                    setRuntimeServicesConfig((prev) => ({
                      ...prev,
                      image_import_notification_receipt_cleanup_interval_hours:
                        parseInt(e.target.value) || 24,
                    }));
                    if (
                      runtimeServicesFieldErrors.image_import_notification_receipt_cleanup_interval_hours
                    ) {
                      setRuntimeServicesFieldErrors((prev) => ({
                        ...prev,
                        image_import_notification_receipt_cleanup_interval_hours:
                          undefined,
                      }));
                    }
                  }
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${
                  runtimeServicesFieldErrors.image_import_notification_receipt_cleanup_interval_hours
                    ? "border-red-500 dark:border-red-500"
                    : "border-slate-300 dark:border-slate-600"
                }`}
              />
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Range: 1-168 hours.
              </p>
              {runtimeServicesFieldErrors.image_import_notification_receipt_cleanup_interval_hours && (
                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                  {
                    runtimeServicesFieldErrors.image_import_notification_receipt_cleanup_interval_hours
                  }
                </p>
              )}
            </div>
              </>
            )}
          </div>
          <div className="mt-6">
            {canManageAdmin && (
              <button
                onClick={saveRuntimeServicesConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save Runtime Services Configuration
              </button>
            )}
          </div>
        </div>
      )}

      {/* LDAP Configuration Tab */}
      {activeTab === "ldap" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            LDAP Authentication Settings
          </h2>
          <div className="mb-6 flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 px-4 py-3">
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                LDAP Authentication
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Disable to turn off LDAP login and directory lookups.
              </p>
            </div>
            <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={ldapConfig.enabled}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              {ldapConfig.enabled ? "Enabled" : "Disabled"}
            </label>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                LDAP Host
              </label>
              <input
                type="text"
                value={ldapConfig.host}
                onChange={(e) =>
                  setLdapConfig((prev) => ({ ...prev, host: e.target.value }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                LDAP Port
              </label>
              <input
                type="number"
                value={ldapConfig.port}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    port: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Base DN
              </label>
              <input
                type="text"
                value={ldapConfig.base_dn}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    base_dn: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Bind DN
              </label>
              <input
                type="text"
                value={ldapConfig.bind_dn}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    bind_dn: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                User Search Base
              </label>
              <input
                type="text"
                value={ldapConfig.user_search_base}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    user_search_base: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Group Search Base
              </label>
              <input
                type="text"
                value={ldapConfig.group_search_base}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    group_search_base: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Bind Password
              </label>
              <input
                type="password"
                value={ldapConfig.bind_password}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    bind_password: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                User Filter
              </label>
              <input
                type="text"
                value={ldapConfig.user_filter}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    user_filter: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Group Filter
              </label>
              <input
                type="text"
                value={ldapConfig.group_filter}
                onChange={(e) =>
                  setLdapConfig((prev) => ({
                    ...prev,
                    group_filter: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div className="flex items-center space-x-4">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={ldapConfig.start_tls}
                  onChange={(e) =>
                    setLdapConfig((prev) => ({
                      ...prev,
                      start_tls: e.target.checked,
                    }))
                  }
                  className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                />
                <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                  Start TLS
                </span>
              </label>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={ldapConfig.ssl}
                  onChange={(e) =>
                    setLdapConfig((prev) => ({
                      ...prev,
                      ssl: e.target.checked,
                    }))
                  }
                  className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                />
                <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                  SSL
                </span>
              </label>
            </div>
          </div>

          {/* Allowed Domains Section */}
          <div className="mt-6">
            <h3 className="text-md font-medium text-slate-900 dark:text-white mb-4">
              Allowed Email Domains
            </h3>
            <div className="space-y-4">
              <div className="flex gap-2">
                <input
                  type="text"
                  value={newDomain}
                  onChange={(e) => setNewDomain(e.target.value)}
                  placeholder="@example.com"
                  className="flex-1 px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                  onKeyPress={(e) =>
                    e.key === "Enter" && canManageAdmin && addAllowedDomain()
                  }
                />
                {canManageAdmin && (
                  <button
                    onClick={addAllowedDomain}
                    className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-md font-medium transition-colors"
                  >
                    Add Domain
                  </button>
                )}
              </div>
              {ldapConfig.allowed_domains.length > 0 && (
                <div className="space-y-2">
                  <p className="text-sm text-slate-600 dark:text-slate-400">
                    Current allowed domains:
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {ldapConfig.allowed_domains.map((domain, index) => (
                      <div
                        key={index}
                        className="flex items-center bg-slate-100 dark:bg-slate-700 rounded-md px-3 py-1"
                      >
                        <span className="text-sm text-slate-900 dark:text-white">
                          {domain}
                        </span>
                        {canManageAdmin && (
                          <button
                            onClick={() => removeAllowedDomain(domain)}
                            className="ml-2 text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                          >
                            ×
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="mt-6">
            {canManageAdmin && (
              <button
                onClick={saveLdapConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save LDAP Configuration
              </button>
            )}
          </div>
        </div>
      )}

      {/* SMTP Configuration Tab */}
      {activeTab === "smtp" && (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            SMTP Email Settings
          </h2>
          <div className="mb-6 flex items-center justify-between rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 px-4 py-3">
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                SMTP Delivery
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Disable to stop sending email notifications.
              </p>
            </div>
            <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
              <input
                type="checkbox"
                checked={smtpConfig.enabled}
                onChange={(e) =>
                  setSmtpConfig((prev) => ({
                    ...prev,
                    enabled: e.target.checked,
                  }))
                }
                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
              />
              {smtpConfig.enabled ? "Enabled" : "Disabled"}
            </label>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                SMTP Host
              </label>
              <input
                type="text"
                value={smtpConfig.host}
                onChange={(e) =>
                  setSmtpConfig((prev) => ({ ...prev, host: e.target.value }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                SMTP Port
              </label>
              <input
                type="number"
                value={smtpConfig.port}
                onChange={(e) =>
                  setSmtpConfig((prev) => ({
                    ...prev,
                    port: parseInt(e.target.value),
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Username
              </label>
              <input
                type="text"
                value={smtpConfig.username}
                onChange={(e) =>
                  setSmtpConfig((prev) => ({
                    ...prev,
                    username: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Password
              </label>
              <input
                type="password"
                value={smtpConfig.password}
                onChange={(e) =>
                  setSmtpConfig((prev) => ({
                    ...prev,
                    password: e.target.value,
                  }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                From Email
              </label>
              <input
                type="email"
                value={smtpConfig.from}
                onChange={(e) =>
                  setSmtpConfig((prev) => ({ ...prev, from: e.target.value }))
                }
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
              />
            </div>
            <div className="flex items-center space-x-4">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={smtpConfig.start_tls}
                  onChange={(e) =>
                    setSmtpConfig((prev) => ({
                      ...prev,
                      start_tls: e.target.checked,
                    }))
                  }
                  className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                />
                <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                  Start TLS
                </span>
              </label>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={smtpConfig.ssl}
                  onChange={(e) =>
                    setSmtpConfig((prev) => ({
                      ...prev,
                      ssl: e.target.checked,
                    }))
                  }
                  className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                />
                <span className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                  SSL
                </span>
              </label>
            </div>
          </div>
          <div className="mt-6">
            {canManageAdmin && (
              <button
                onClick={saveSmtpConfig}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
              >
                Save SMTP Configuration
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default SystemConfigurationPage;
