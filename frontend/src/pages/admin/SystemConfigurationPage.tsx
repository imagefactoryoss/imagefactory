import { useCanManageAdmin } from "@/hooks/useAccess";
import { adminService } from "@/services/adminService";
import { api } from "@/services/api";
import { auditService } from "@/services/auditService";
import { AuditEvent } from "@/types/audit";
import React, { Suspense, lazy, useEffect, useState } from "react";
import toast from "react-hot-toast";
import {
  BuildConfigFormData,
  GeneralConfigFormData,
  getInitialSystemConfigurationTab,
  LDAPConfigFormData,
  MessagingConfigFormData,
  QuarantinePolicyFormData,
  QuarantinePolicySimulationResult,
  QuarantinePolicyValidationResult,
  RobotSREPolicyFormData,
  RuntimeServicesConfigFormData,
  RuntimeServicesFieldErrors,
  RuntimeServicesSubTab,
  SecurityConfigFormData,
  SMTPConfigFormData,
  SORRegistrationFormData,
  SystemConfig,
  SystemConfigurationTabId,
  TektonCoreConfigFormData,
  TektonSubTab,
  TektonTaskImagesFormData,
} from "./systemConfigurationShared";

const BuildConfigurationPanel = lazy(() =>
  import("./systemConfigurationBuildTektonPanels").then((module) => ({
    default: module.BuildConfigurationPanel,
  })),
);
const SystemConfigurationTabs = lazy(() =>
  import("./systemConfigurationBuildTektonPanels").then((module) => ({
    default: module.SystemConfigurationTabs,
  })),
);
const TektonConfigurationPanel = lazy(() =>
  import("./systemConfigurationBuildTektonPanels").then((module) => ({
    default: module.TektonConfigurationPanel,
  })),
);
const GeneralSystemConfigurationPanel = lazy(() =>
  import("./systemConfigurationCorePanels").then((module) => ({
    default: module.GeneralSystemConfigurationPanel,
  })),
);
const MessagingConfigurationPanel = lazy(() =>
  import("./systemConfigurationCorePanels").then((module) => ({
    default: module.MessagingConfigurationPanel,
  })),
);
const SecurityConfigurationPanel = lazy(() =>
  import("./systemConfigurationCorePanels").then((module) => ({
    default: module.SecurityConfigurationPanel,
  })),
);
const LDAPConfigurationPanel = lazy(() =>
  import("./systemConfigurationDirectoryPanels").then((module) => ({
    default: module.LDAPConfigurationPanel,
  })),
);
const SMTPConfigurationPanel = lazy(() =>
  import("./systemConfigurationDirectoryPanels").then((module) => ({
    default: module.SMTPConfigurationPanel,
  })),
);
const QuarantinePolicyConfigurationPanel = lazy(() =>
  import("./systemConfigurationPolicyPanels").then((module) => ({
    default: module.QuarantinePolicyConfigurationPanel,
  })),
);
const SORRegistrationConfigurationPanel = lazy(() =>
  import("./systemConfigurationPolicyPanels").then((module) => ({
    default: module.SORRegistrationConfigurationPanel,
  })),
);
const RuntimeServicesConfigurationPanel = lazy(() =>
  import("./systemConfigurationRuntimePanels").then((module) => ({
    default: module.RuntimeServicesConfigurationPanel,
  })),
);
const RobotSREPolicyPanel = lazy(() =>
  import("./systemConfigurationSREPanels").then((module) => ({
    default: module.RobotSREPolicyPanel,
  })),
);

const tabPanelLoadingFallback = (
  <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300">
    Loading configuration panel...
  </div>
);

const SystemConfigurationPage: React.FC = () => {
  const canManageAdmin = useCanManageAdmin();
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<SystemConfigurationTabId>(
    getInitialSystemConfigurationTab,
  );

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
      apphq_enabled: false,
      apphq_oauth_token_url: "",
      apphq_client_id: "",
      apphq_client_secret: "",
      apphq_api_url: "",
      apphq_system: "",
      apphq_system_name: "",
      apphq_run: "",
      apphq_obj_cd: "",
    });
  const [runtimeServicesFieldErrors, setRuntimeServicesFieldErrors] = useState<RuntimeServicesFieldErrors>({});
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

  const [robotSREPolicyLoading, setRobotSREPolicyLoading] = useState(false);
  const [robotSREPolicy, setRobotSREPolicy] = useState<RobotSREPolicyFormData>({
    display_name: "",
    enabled: false,
    environment_mode: "observe_only",
    auto_observe_enabled: true,
    auto_notify_enabled: false,
    auto_contain_enabled: false,
    auto_recover_enabled: false,
    require_approval_for_recover: true,
    require_approval_for_disruptive: true,
    duplicate_alert_suppression_seconds: 300,
    action_cooldown_seconds: 60,
    enabled_domains: [],
    remediation_packs: [],
  });

  const [purgeLogs, setPurgeLogs] = useState<AuditEvent[]>([]);
  const [purgeLogsLoading, setPurgeLogsLoading] = useState(false);
  const [purgeLogsLoaded, setPurgeLogsLoaded] = useState(false);
  const [purgeLogsError, setPurgeLogsError] = useState<string | null>(null);
  const [showMessagingTooltip, setShowMessagingTooltip] = useState(false);
  const [rebooting, setRebooting] = useState(false);

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

  useEffect(() => {
    if (activeTab !== "robot_sre") {
      return;
    }
    loadRobotSREPolicy();
  }, [activeTab]);

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

  const loadRobotSREPolicy = async () => {
    try {
      setRobotSREPolicyLoading(true);
      const response = await api.get("/admin/settings/robot-sre");
      setRobotSREPolicy((prev) => ({
        ...prev,
        ...response.data,
        remediation_packs: response.data?.remediation_packs ?? prev.remediation_packs,
        enabled_domains: response.data?.enabled_domains ?? prev.enabled_domains,
      }));
    } catch {
      toast.error("Failed to load Robot SRE policy");
    } finally {
      setRobotSREPolicyLoading(false);
    }
  };

  const loadRobotSREPolicyDefaults = async () => {
    try {
      setRobotSREPolicyLoading(true);
      const response = await api.get("/admin/settings/robot-sre/defaults");
      setRobotSREPolicy((prev) => ({
        ...prev,
        ...response.data,
        remediation_packs: response.data?.remediation_packs ?? prev.remediation_packs,
        enabled_domains: response.data?.enabled_domains ?? prev.enabled_domains,
      }));
      toast.success("Defaults loaded — click Save to persist");
    } catch {
      toast.error("Failed to load Robot SRE policy defaults");
    } finally {
      setRobotSREPolicyLoading(false);
    }
  };

  const saveRobotSREPolicy = async () => {
    try {
      await api.put("/admin/settings/robot-sre", robotSREPolicy);
      toast.success("Robot SRE policy saved successfully");
      await loadRobotSREPolicy();
    } catch (error: any) {
      const message =
        error?.response?.data?.message ||
        error?.response?.data?.error ||
        "Failed to save Robot SRE policy";
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

      <Suspense fallback={tabPanelLoadingFallback}>
        <SystemConfigurationTabs activeTab={activeTab} setActiveTab={setActiveTab} />
      </Suspense>

      <Suspense fallback={tabPanelLoadingFallback}>
        {activeTab === "build" && (
          <BuildConfigurationPanel
            buildConfig={buildConfig}
            setBuildConfig={setBuildConfig}
            canManageAdmin={canManageAdmin}
            saveBuildConfig={saveBuildConfig}
          />
        )}

        {activeTab === "tekton" && (
          <TektonConfigurationPanel
            tektonSubTab={tektonSubTab}
            setTektonSubTab={setTektonSubTab}
            tektonCoreConfig={tektonCoreConfig}
            setTektonCoreConfig={setTektonCoreConfig}
            tektonTaskImages={tektonTaskImages}
            setTektonTaskImages={setTektonTaskImages}
            runtimeServicesConfig={runtimeServicesConfig}
            setRuntimeServicesConfig={setRuntimeServicesConfig}
            runtimeServicesFieldErrors={runtimeServicesFieldErrors}
            canManageAdmin={canManageAdmin}
            saveTektonCoreConfig={saveTektonCoreConfig}
            saveTektonTaskImagesConfig={saveTektonTaskImagesConfig}
            saveRuntimeServicesConfig={saveRuntimeServicesConfig}
          />
        )}

        {activeTab === "quarantine_policy" && (
          <QuarantinePolicyConfigurationPanel
            quarantinePolicyScope={quarantinePolicyScope}
            setQuarantinePolicyScope={setQuarantinePolicyScope}
            quarantinePolicyTenantID={quarantinePolicyTenantID}
            setQuarantinePolicyTenantID={setQuarantinePolicyTenantID}
            quarantinePolicyLoading={quarantinePolicyLoading}
            quarantinePolicyConfig={quarantinePolicyConfig}
            setQuarantinePolicyConfig={setQuarantinePolicyConfig}
            quarantinePolicySimulationInput={quarantinePolicySimulationInput}
            setQuarantinePolicySimulationInput={setQuarantinePolicySimulationInput}
            quarantinePolicyValidation={quarantinePolicyValidation}
            quarantinePolicySimulationResult={quarantinePolicySimulationResult}
            canManageAdmin={canManageAdmin}
            loadQuarantinePolicyConfig={loadQuarantinePolicyConfig}
            validateQuarantinePolicyConfig={validateQuarantinePolicyConfig}
            simulateQuarantinePolicyConfig={simulateQuarantinePolicyConfig}
            saveQuarantinePolicyConfig={saveQuarantinePolicyConfig}
          />
        )}

        {activeTab === "sor_registration" && (
          <SORRegistrationConfigurationPanel
            sorRegistrationScope={sorRegistrationScope}
            setSorRegistrationScope={setSorRegistrationScope}
            sorRegistrationTenantID={sorRegistrationTenantID}
            setSorRegistrationTenantID={setSorRegistrationTenantID}
            sorRegistrationLoading={sorRegistrationLoading}
            sorRegistrationConfig={sorRegistrationConfig}
            setSorRegistrationConfig={setSorRegistrationConfig}
            canManageAdmin={canManageAdmin}
            loadSORRegistrationConfig={loadSORRegistrationConfig}
            saveSORRegistrationConfig={saveSORRegistrationConfig}
          />
        )}

        {activeTab === "security" && (
          <SecurityConfigurationPanel
            securityConfig={securityConfig}
            setSecurityConfig={setSecurityConfig}
            canManageAdmin={canManageAdmin}
            saveSecurityConfig={saveSecurityConfig}
          />
        )}

        {activeTab === "general" && (
          <GeneralSystemConfigurationPanel
            generalConfig={generalConfig}
            setGeneralConfig={setGeneralConfig}
            purgeLogs={purgeLogs}
            purgeLogsLoading={purgeLogsLoading}
            purgeLogsError={purgeLogsError}
            canManageAdmin={canManageAdmin}
            isProd={isProd}
            rebooting={rebooting}
            handlePurgeDeletedProjects={handlePurgeDeletedProjects}
            loadPurgeLogs={loadPurgeLogs}
            handleRebootServer={handleRebootServer}
            saveGeneralConfig={saveGeneralConfig}
          />
        )}

        {activeTab === "messaging" && (
          <MessagingConfigurationPanel
            messagingConfig={messagingConfig}
            setMessagingConfig={setMessagingConfig}
            showMessagingTooltip={showMessagingTooltip}
            setShowMessagingTooltip={setShowMessagingTooltip}
            canManageAdmin={canManageAdmin}
            saveMessagingConfig={saveMessagingConfig}
          />
        )}

        {activeTab === "ldap" && (
          <LDAPConfigurationPanel
            ldapConfig={ldapConfig}
            setLdapConfig={setLdapConfig}
            newDomain={newDomain}
            setNewDomain={setNewDomain}
            canManageAdmin={canManageAdmin}
            addAllowedDomain={addAllowedDomain}
            removeAllowedDomain={removeAllowedDomain}
            saveLdapConfig={saveLdapConfig}
          />
        )}

        {activeTab === "smtp" && (
          <SMTPConfigurationPanel
            smtpConfig={smtpConfig}
            setSmtpConfig={setSmtpConfig}
            canManageAdmin={canManageAdmin}
            saveSmtpConfig={saveSmtpConfig}
          />
        )}

        {activeTab === "runtime_services" && (
          <RuntimeServicesConfigurationPanel
            runtimeServicesConfig={runtimeServicesConfig}
            setRuntimeServicesConfig={setRuntimeServicesConfig}
            runtimeServicesFieldErrors={runtimeServicesFieldErrors}
            setRuntimeServicesFieldErrors={setRuntimeServicesFieldErrors}
            runtimeServicesSubTab={runtimeServicesSubTab}
            setRuntimeServicesSubTab={setRuntimeServicesSubTab}
            canManageAdmin={canManageAdmin}
            saveRuntimeServicesConfig={saveRuntimeServicesConfig}
          />
        )}

        {activeTab === "robot_sre" && (
          <RobotSREPolicyPanel
            robotSREPolicy={robotSREPolicy}
            setRobotSREPolicy={setRobotSREPolicy}
            robotSREPolicyLoading={robotSREPolicyLoading}
            canManageAdmin={canManageAdmin}
            loadRobotSREPolicy={loadRobotSREPolicy}
            saveRobotSREPolicy={saveRobotSREPolicy}
            loadRobotSREPolicyDefaults={loadRobotSREPolicyDefaults}
          />
        )}
      </Suspense>
    </div>
  );
};

export default SystemConfigurationPage;
