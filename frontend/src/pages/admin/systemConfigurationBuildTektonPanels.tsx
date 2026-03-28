import React from "react";
import type {
    BuildConfigFormData,
    RuntimeServicesConfigFormData,
    RuntimeServicesFieldErrors,
    SystemConfigurationTabId,
    TektonCoreConfigFormData,
    TektonSubTab,
    TektonTaskImagesFormData,
} from "./systemConfigurationShared";
import { RestartRequiredBadge, SYSTEM_CONFIGURATION_TABS } from "./systemConfigurationShared";

export const SystemConfigurationTabs: React.FC<{
    activeTab: SystemConfigurationTabId;
    setActiveTab: (tab: SystemConfigurationTabId) => void;
}> = ({ activeTab, setActiveTab }) => (
    <div className="border-b border-slate-200 dark:border-slate-700">
        <nav className="-mb-px flex space-x-8 overflow-x-auto">
            {SYSTEM_CONFIGURATION_TABS.map((tab) => (
                <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${activeTab === tab.id
                        ? "border-blue-500 text-blue-600 dark:text-blue-400"
                        : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300"
                        }`}
                >
                    {tab.label}
                </button>
            ))}
        </nav>
    </div>
);

export const BuildConfigurationPanel: React.FC<{
    buildConfig: BuildConfigFormData;
    setBuildConfig: React.Dispatch<React.SetStateAction<BuildConfigFormData>>;
    canManageAdmin: boolean;
    saveBuildConfig: () => void;
}> = ({ buildConfig, setBuildConfig, canManageAdmin, saveBuildConfig }) => (
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
                <label htmlFor="tekton-enabled" className="text-sm text-slate-700 dark:text-slate-300">
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
                    <label htmlFor="build-monitor-event-driven-enabled" className="text-sm font-medium text-slate-700 dark:text-slate-300">
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
                    <label htmlFor="build-enable-temp-scan-stage" className="text-sm font-medium text-slate-700 dark:text-slate-300">
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
);

export const TektonConfigurationPanel: React.FC<{
    tektonSubTab: TektonSubTab;
    setTektonSubTab: React.Dispatch<React.SetStateAction<TektonSubTab>>;
    tektonCoreConfig: TektonCoreConfigFormData;
    setTektonCoreConfig: React.Dispatch<React.SetStateAction<TektonCoreConfigFormData>>;
    tektonTaskImages: TektonTaskImagesFormData;
    setTektonTaskImages: React.Dispatch<React.SetStateAction<TektonTaskImagesFormData>>;
    runtimeServicesConfig: RuntimeServicesConfigFormData;
    setRuntimeServicesConfig: React.Dispatch<React.SetStateAction<RuntimeServicesConfigFormData>>;
    runtimeServicesFieldErrors: RuntimeServicesFieldErrors;
    canManageAdmin: boolean;
    saveTektonCoreConfig: () => void;
    saveTektonTaskImagesConfig: () => void;
    saveRuntimeServicesConfig: () => void;
}> = ({
    tektonSubTab,
    setTektonSubTab,
    tektonCoreConfig,
    setTektonCoreConfig,
    tektonTaskImages,
    setTektonTaskImages,
    runtimeServicesConfig,
    setRuntimeServicesConfig,
    runtimeServicesFieldErrors,
    canManageAdmin,
    saveTektonCoreConfig,
    saveTektonTaskImagesConfig,
    saveRuntimeServicesConfig,
}) => (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
            <div className="mb-6">
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                    Tekton Core Defaults
                </h2>
                <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
                    Global configuration used when preparing Kubernetes infrastructure providers. Provider-level settings override these defaults.
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
                        className={`px-3 py-1.5 rounded-md border text-sm font-medium transition-colors ${tektonSubTab === tab.key
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
                                        install_source: e.target.value as TektonCoreConfigFormData["install_source"],
                                    }))
                                }
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            >
                                <option value="manifest">Manifest (apply release YAML URLs)</option>
                                <option value="helm">Helm (admin installs outside Image Factory)</option>
                                <option value="preinstalled">Preinstalled (no install attempt)</option>
                            </select>
                            <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                                For air-gapped clusters, use preinstalled or point manifest URLs to an internal mirror.
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
                                Path inside the API container where Tekton task/pipeline manifests are mounted (must include a kustomization.yaml).
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
                                        value={tektonTaskImages[item.key as keyof TektonTaskImagesFormData]}
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
                                    <option value="full_reconcile_on_prepare">Full Reconcile On Prepare</option>
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
                                            tenant_asset_drift_watcher_interval_seconds: parseInt(e.target.value) || 300,
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
                                    className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.storage_profiles_internal_registry_type
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
                            {runtimeServicesConfig.storage_profiles.internal_registry.type === "hostPath" && (
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
                                            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path
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
                                            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.storage_profiles_internal_registry_host_path_type
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
                            {runtimeServicesConfig.storage_profiles.internal_registry.type === "pvc" && (
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
                                            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_name
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
                                            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.storage_profiles_internal_registry_pvc_access_modes_0
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
                            {runtimeServicesConfig.storage_profiles.internal_registry.type === "emptyDir" && (
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
    );
