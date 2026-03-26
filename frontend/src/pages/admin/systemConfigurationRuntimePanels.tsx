import React from "react";

import type {
    RuntimeServicesConfigFormData,
    RuntimeServicesFieldErrors,
    RuntimeServicesSubTab,
} from "./systemConfigurationShared";

const RUNTIME_SERVICE_TABS: Array<{
    key: RuntimeServicesSubTab;
    label: string;
}> = [
        { key: "services", label: "Service Endpoints" },
        { key: "registry_gc", label: "Registry GC" },
        { key: "watchers", label: "Watchers" },
        { key: "cleanup", label: "Cleanup Jobs" },
        { key: "tenant_service", label: "Tenant Service (AppHQ)" },
    ];

export const RuntimeServicesConfigurationPanel: React.FC<{
    runtimeServicesConfig: RuntimeServicesConfigFormData;
    setRuntimeServicesConfig: React.Dispatch<
        React.SetStateAction<RuntimeServicesConfigFormData>
    >;
    runtimeServicesFieldErrors: RuntimeServicesFieldErrors;
    setRuntimeServicesFieldErrors: React.Dispatch<
        React.SetStateAction<RuntimeServicesFieldErrors>
    >;
    runtimeServicesSubTab: RuntimeServicesSubTab;
    setRuntimeServicesSubTab: React.Dispatch<
        React.SetStateAction<RuntimeServicesSubTab>
    >;
    canManageAdmin: boolean;
    saveRuntimeServicesConfig: () => void;
}> = ({
    runtimeServicesConfig,
    setRuntimeServicesConfig,
    runtimeServicesFieldErrors,
    setRuntimeServicesFieldErrors,
    runtimeServicesSubTab,
    setRuntimeServicesSubTab,
    canManageAdmin,
    saveRuntimeServicesConfig,
}) => (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                Runtime Services
            </h2>
            <p className="text-sm text-slate-600 dark:text-slate-300 mb-4">
                Configure connection details used by administrators to validate status and
                operations for independent runtime services.
            </p>
            <div className="mb-5 flex flex-wrap gap-2">
                {RUNTIME_SERVICE_TABS.map((tab) => (
                    <button
                        key={tab.key}
                        type="button"
                        onClick={() => setRuntimeServicesSubTab(tab.key)}
                        className={`px-3 py-1.5 rounded-md border text-sm font-medium transition-colors ${runtimeServicesSubTab === tab.key
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.dispatcher_url
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.dispatcher_port
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.email_worker_url
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.email_worker_port
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.notification_worker_url
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.notification_worker_port
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.internal_registry_gc_worker_url
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.internal_registry_gc_worker_port
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
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.internal_registry_temp_cleanup_retention_hours
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
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.internal_registry_temp_cleanup_interval_minutes
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
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.internal_registry_temp_cleanup_batch_size
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.health_check_timeout_seconds
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
                                Changes apply without restart. Recommended defaults: interval <span className="font-mono">180s</span>, timeout <span className="font-mono">90s</span>, batch <span className="font-mono">200</span>.
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.provider_readiness_watcher_interval_seconds
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.provider_readiness_watcher_timeout_seconds
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.provider_readiness_watcher_batch_size
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.tekton_history_cleanup_schedule
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.tekton_history_cleanup_keep_pipelineruns
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
                                onChange={(e) => {
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
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.tekton_history_cleanup_keep_taskruns
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
                                onChange={(e) => {
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        tekton_history_cleanup_keep_pods:
                                            parseInt(e.target.value) || 240,
                                    }));
                                    if (runtimeServicesFieldErrors.tekton_history_cleanup_keep_pods) {
                                        setRuntimeServicesFieldErrors((prev) => ({
                                            ...prev,
                                            tekton_history_cleanup_keep_pods: undefined,
                                        }));
                                    }
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.tekton_history_cleanup_keep_pods
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
                                Controls retention for persisted idempotency receipts used to prevent duplicate quarantine import notifications during event replay.
                            </p>
                        </div>
                        <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                            <input
                                type="checkbox"
                                checked={runtimeServicesConfig.image_import_notification_receipt_cleanup_enabled}
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
                                value={runtimeServicesConfig.image_import_notification_receipt_retention_days}
                                onChange={(e) => {
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
                                            image_import_notification_receipt_retention_days: undefined,
                                        }));
                                    }
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.image_import_notification_receipt_retention_days
                                    ? "border-red-500 dark:border-red-500"
                                    : "border-slate-300 dark:border-slate-600"
                                    }`}
                            />
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                Range: 1-3650 days.
                            </p>
                            {runtimeServicesFieldErrors.image_import_notification_receipt_retention_days && (
                                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                                    {runtimeServicesFieldErrors.image_import_notification_receipt_retention_days}
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
                                value={runtimeServicesConfig.image_import_notification_receipt_cleanup_interval_hours}
                                onChange={(e) => {
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
                                            image_import_notification_receipt_cleanup_interval_hours: undefined,
                                        }));
                                    }
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.image_import_notification_receipt_cleanup_interval_hours
                                    ? "border-red-500 dark:border-red-500"
                                    : "border-slate-300 dark:border-slate-600"
                                    }`}
                            />
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                Range: 1-168 hours.
                            </p>
                            {runtimeServicesFieldErrors.image_import_notification_receipt_cleanup_interval_hours && (
                                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                                    {runtimeServicesFieldErrors.image_import_notification_receipt_cleanup_interval_hours}
                                </p>
                            )}
                        </div>
                    </>
                )}
                {runtimeServicesSubTab === "tenant_service" && (
                    <>
                        <div className="md:col-span-2">
                            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-1">
                                AppHQ Tenant Lookup
                            </h3>
                            <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
                                Configure the corporate AppHQ service used to look up tenant details during onboarding.
                            </p>
                        </div>
                        <div className="md:col-span-2 flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                            <input
                                type="checkbox"
                                checked={runtimeServicesConfig.apphq_enabled}
                                onChange={(e) =>
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_enabled: e.target.checked,
                                    }))
                                }
                                className="rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                            />
                            Enable AppHQ tenant lookup
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                OAuth Token URL
                            </label>
                            <input
                                type="text"
                                value={runtimeServicesConfig.apphq_oauth_token_url}
                                onChange={(e) => {
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_oauth_token_url: e.target.value,
                                    }));
                                    if (runtimeServicesFieldErrors.apphq_oauth_token_url) {
                                        setRuntimeServicesFieldErrors((prev) => ({
                                            ...prev,
                                            apphq_oauth_token_url: undefined,
                                        }));
                                    }
                                }}
                                placeholder="https://example.com/as/token.oauth2"
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.apphq_oauth_token_url
                                    ? "border-red-500 dark:border-red-500"
                                    : "border-slate-300 dark:border-slate-600"
                                    }`}
                            />
                            {runtimeServicesFieldErrors.apphq_oauth_token_url && (
                                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                                    {runtimeServicesFieldErrors.apphq_oauth_token_url}
                                </p>
                            )}
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                API URL
                            </label>
                            <input
                                type="text"
                                value={runtimeServicesConfig.apphq_api_url}
                                onChange={(e) => {
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_api_url: e.target.value,
                                    }));
                                    if (runtimeServicesFieldErrors.apphq_api_url) {
                                        setRuntimeServicesFieldErrors((prev) => ({
                                            ...prev,
                                            apphq_api_url: undefined,
                                        }));
                                    }
                                }}
                                placeholder="https://example.com/TechAdsExecService/exec/run"
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.apphq_api_url
                                    ? "border-red-500 dark:border-red-500"
                                    : "border-slate-300 dark:border-slate-600"
                                    }`}
                            />
                            {runtimeServicesFieldErrors.apphq_api_url && (
                                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                                    {runtimeServicesFieldErrors.apphq_api_url}
                                </p>
                            )}
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Client ID
                            </label>
                            <input
                                type="text"
                                value={runtimeServicesConfig.apphq_client_id}
                                onChange={(e) => {
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_client_id: e.target.value,
                                    }));
                                    if (runtimeServicesFieldErrors.apphq_client_id) {
                                        setRuntimeServicesFieldErrors((prev) => ({
                                            ...prev,
                                            apphq_client_id: undefined,
                                        }));
                                    }
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.apphq_client_id
                                    ? "border-red-500 dark:border-red-500"
                                    : "border-slate-300 dark:border-slate-600"
                                    }`}
                            />
                            {runtimeServicesFieldErrors.apphq_client_id && (
                                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                                    {runtimeServicesFieldErrors.apphq_client_id}
                                </p>
                            )}
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Client Secret
                            </label>
                            <input
                                type="password"
                                value={runtimeServicesConfig.apphq_client_secret}
                                onChange={(e) => {
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_client_secret: e.target.value,
                                    }));
                                    if (runtimeServicesFieldErrors.apphq_client_secret) {
                                        setRuntimeServicesFieldErrors((prev) => ({
                                            ...prev,
                                            apphq_client_secret: undefined,
                                        }));
                                    }
                                }}
                                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white ${runtimeServicesFieldErrors.apphq_client_secret
                                    ? "border-red-500 dark:border-red-500"
                                    : "border-slate-300 dark:border-slate-600"
                                    }`}
                            />
                            {runtimeServicesFieldErrors.apphq_client_secret && (
                                <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                                    {runtimeServicesFieldErrors.apphq_client_secret}
                                </p>
                            )}
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                System
                            </label>
                            <input
                                type="text"
                                value={runtimeServicesConfig.apphq_system}
                                onChange={(e) =>
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_system: e.target.value,
                                    }))
                                }
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                System Name
                            </label>
                            <input
                                type="text"
                                value={runtimeServicesConfig.apphq_system_name}
                                onChange={(e) =>
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_system_name: e.target.value,
                                    }))
                                }
                                placeholder="Landlord"
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Run
                            </label>
                            <input
                                type="text"
                                value={runtimeServicesConfig.apphq_run}
                                onChange={(e) =>
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_run: e.target.value,
                                    }))
                                }
                                placeholder="TECHADS_GENERIC_REST_API"
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Object Code
                            </label>
                            <input
                                type="text"
                                value={runtimeServicesConfig.apphq_obj_cd}
                                onChange={(e) =>
                                    setRuntimeServicesConfig((prev) => ({
                                        ...prev,
                                        apphq_obj_cd: e.target.value,
                                    }))
                                }
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                            />
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
    );