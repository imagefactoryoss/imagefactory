import { HelpCircle, X } from "lucide-react";
import React from "react";

import type { AuditEvent } from "@/types/audit";

import type {
    GeneralConfigFormData,
    MessagingConfigFormData,
    SecurityConfigFormData,
} from "./systemConfigurationShared";
import { RestartRequiredBadge } from "./systemConfigurationShared";

export const SecurityConfigurationPanel: React.FC<{
    securityConfig: SecurityConfigFormData;
    setSecurityConfig: React.Dispatch<React.SetStateAction<SecurityConfigFormData>>;
    canManageAdmin: boolean;
    saveSecurityConfig: () => void;
}> = ({
    securityConfig,
    setSecurityConfig,
    canManageAdmin,
    saveSecurityConfig,
}) => (
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
    );

export const GeneralSystemConfigurationPanel: React.FC<{
    generalConfig: GeneralConfigFormData;
    setGeneralConfig: React.Dispatch<React.SetStateAction<GeneralConfigFormData>>;
    purgeLogs: AuditEvent[];
    purgeLogsLoading: boolean;
    purgeLogsError: string | null;
    canManageAdmin: boolean;
    isProd: boolean;
    rebooting: boolean;
    handlePurgeDeletedProjects: () => void;
    loadPurgeLogs: (force?: boolean) => void;
    handleRebootServer: () => void;
    saveGeneralConfig: () => void;
}> = ({
    generalConfig,
    setGeneralConfig,
    purgeLogs,
    purgeLogsLoading,
    purgeLogsError,
    canManageAdmin,
    isProd,
    rebooting,
    handlePurgeDeletedProjects,
    loadPurgeLogs,
    handleRebootServer,
    saveGeneralConfig,
}) => (
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
                        Number of days to retain soft-deleted projects before cleanup. Set to 0 to disable automatic purge.
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
                                When enabled, write actions are disabled for non-admin users.
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
                                Controls internal workflow polling for queued workflow steps.
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
                                Controls admin execution pipeline health auto-refresh frequency.
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
                        <div className="text-red-600 dark:text-red-400">{purgeLogsError}</div>
                    )}
                    {!purgeLogsLoading && !purgeLogsError && purgeLogs.length === 0 && (
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
    );

export const MessagingConfigurationPanel: React.FC<{
    messagingConfig: MessagingConfigFormData;
    setMessagingConfig: React.Dispatch<React.SetStateAction<MessagingConfigFormData>>;
    showMessagingTooltip: boolean;
    setShowMessagingTooltip: React.Dispatch<React.SetStateAction<boolean>>;
    canManageAdmin: boolean;
    saveMessagingConfig: () => void;
}> = ({
    messagingConfig,
    setMessagingConfig,
    showMessagingTooltip,
    setShowMessagingTooltip,
    canManageAdmin,
    saveMessagingConfig,
}) => (
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
                            Requires NATS server + notification worker. Changes take effect on service restart.
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
                            Fail startup if NATS is unavailable instead of falling back to local-only messaging.
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
                            Publish and subscribe through NATS only. Disables local in-process event bus delivery.
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
                            Queue events in database if external publish fails and replay them in background.
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
    );