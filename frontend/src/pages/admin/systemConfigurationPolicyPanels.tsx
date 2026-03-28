import React from "react";

import type {
    QuarantinePolicyFormData,
    QuarantinePolicySimulationResult,
    QuarantinePolicyValidationResult,
    SORRegistrationFormData,
} from "./systemConfigurationShared";

export const QuarantinePolicyConfigurationPanel: React.FC<{
    quarantinePolicyScope: "global" | "tenant";
    setQuarantinePolicyScope: React.Dispatch<
        React.SetStateAction<"global" | "tenant">
    >;
    quarantinePolicyTenantID: string;
    setQuarantinePolicyTenantID: React.Dispatch<React.SetStateAction<string>>;
    quarantinePolicyLoading: boolean;
    quarantinePolicyConfig: QuarantinePolicyFormData;
    setQuarantinePolicyConfig: React.Dispatch<
        React.SetStateAction<QuarantinePolicyFormData>
    >;
    quarantinePolicySimulationInput: {
        critical: number;
        high: number;
        medium: number;
        maxCVSS: number;
    };
    setQuarantinePolicySimulationInput: React.Dispatch<
        React.SetStateAction<{
            critical: number;
            high: number;
            medium: number;
            maxCVSS: number;
        }>
    >;
    quarantinePolicyValidation: QuarantinePolicyValidationResult | null;
    quarantinePolicySimulationResult: QuarantinePolicySimulationResult | null;
    canManageAdmin: boolean;
    loadQuarantinePolicyConfig: () => void;
    validateQuarantinePolicyConfig: () => void;
    simulateQuarantinePolicyConfig: () => void;
    saveQuarantinePolicyConfig: () => void;
}> = ({
    quarantinePolicyScope,
    setQuarantinePolicyScope,
    quarantinePolicyTenantID,
    setQuarantinePolicyTenantID,
    quarantinePolicyLoading,
    quarantinePolicyConfig,
    setQuarantinePolicyConfig,
    quarantinePolicySimulationInput,
    setQuarantinePolicySimulationInput,
    quarantinePolicyValidation,
    quarantinePolicySimulationResult,
    canManageAdmin,
    loadQuarantinePolicyConfig,
    validateQuarantinePolicyConfig,
    simulateQuarantinePolicyConfig,
    saveQuarantinePolicyConfig,
}) => (
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
    );

export const SORRegistrationConfigurationPanel: React.FC<{
    sorRegistrationScope: "global" | "tenant";
    setSorRegistrationScope: React.Dispatch<
        React.SetStateAction<"global" | "tenant">
    >;
    sorRegistrationTenantID: string;
    setSorRegistrationTenantID: React.Dispatch<React.SetStateAction<string>>;
    sorRegistrationLoading: boolean;
    sorRegistrationConfig: SORRegistrationFormData;
    setSorRegistrationConfig: React.Dispatch<
        React.SetStateAction<SORRegistrationFormData>
    >;
    canManageAdmin: boolean;
    loadSORRegistrationConfig: () => void;
    saveSORRegistrationConfig: () => void;
}> = ({
    sorRegistrationScope,
    setSorRegistrationScope,
    sorRegistrationTenantID,
    setSorRegistrationTenantID,
    sorRegistrationLoading,
    sorRegistrationConfig,
    setSorRegistrationConfig,
    canManageAdmin,
    loadSORRegistrationConfig,
    saveSORRegistrationConfig,
}) => (
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
                                runtime_error_mode: e.target.value as "error" | "deny" | "allow",
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
    );