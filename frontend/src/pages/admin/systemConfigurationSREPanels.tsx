import React, { useState } from "react";

import type {
    RobotSREPolicyFormData,
    RobotSRERemediationPackFormData,
} from "./systemConfigurationShared";

const inputClass =
    "w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white";

const RISK_TIERS: Array<{ value: "low" | "medium" | "high"; label: string }> = [
    { value: "low", label: "Low" },
    { value: "medium", label: "Medium" },
    { value: "high", label: "High" },
];

const emptyPack = (): RobotSRERemediationPackFormData => ({
    key: "",
    version: "1.0.0",
    name: "",
    summary: "",
    risk_tier: "low",
    action_class: "",
    requires_approval: false,
    incident_types: [],
});

type EditState =
    | { mode: "idle" }
    | { mode: "add"; draft: RobotSRERemediationPackFormData }
    | { mode: "edit"; index: number; draft: RobotSRERemediationPackFormData };

interface PackFormProps {
    draft: RobotSRERemediationPackFormData;
    isNew: boolean;
    onChange: (updated: RobotSRERemediationPackFormData) => void;
    onSave: () => void;
    onCancel: () => void;
}

const PackForm: React.FC<PackFormProps> = ({ draft, isNew, onChange, onSave, onCancel }) => {
    const [incidentTypesRaw, setIncidentTypesRaw] = useState<string>(
        draft.incident_types.join(", "),
    );

    const handleIncidentTypesChange = (raw: string) => {
        setIncidentTypesRaw(raw);
        onChange({
            ...draft,
            incident_types: raw
                .split(",")
                .map((s) => s.trim())
                .filter(Boolean),
        });
    };

    return (
        <div className="rounded-md border border-blue-300 dark:border-blue-600 bg-blue-50 dark:bg-blue-900/20 p-4 space-y-4">
            <h4 className="text-sm font-semibold text-slate-900 dark:text-white">
                {isNew ? "New Remediation Pack" : "Edit Remediation Pack"}
            </h4>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Key <span className="text-rose-500">*</span>
                    </label>
                    <input
                        type="text"
                        value={draft.key}
                        disabled={!isNew}
                        onChange={(e) => onChange({ ...draft, key: e.target.value })}
                        placeholder="e.g. async_backlog_pressure_pack"
                        className={`${inputClass} ${!isNew ? "opacity-60 cursor-not-allowed" : ""}`}
                    />
                    {!isNew && (
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                            Key cannot be changed after creation.
                        </p>
                    )}
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Version
                    </label>
                    <input
                        type="text"
                        value={draft.version}
                        onChange={(e) => onChange({ ...draft, version: e.target.value })}
                        placeholder="1.0.0"
                        className={inputClass}
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Name <span className="text-rose-500">*</span>
                    </label>
                    <input
                        type="text"
                        value={draft.name}
                        onChange={(e) => onChange({ ...draft, name: e.target.value })}
                        placeholder="Human-readable name"
                        className={inputClass}
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Action Class
                    </label>
                    <input
                        type="text"
                        value={draft.action_class}
                        onChange={(e) => onChange({ ...draft, action_class: e.target.value })}
                        placeholder="e.g. async_pressure_relief"
                        className={inputClass}
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Risk Tier
                    </label>
                    <select
                        value={draft.risk_tier}
                        onChange={(e) =>
                            onChange({ ...draft, risk_tier: e.target.value as "low" | "medium" | "high" })
                        }
                        className={inputClass}
                    >
                        {RISK_TIERS.map((t) => (
                            <option key={t.value} value={t.value}>
                                {t.label}
                            </option>
                        ))}
                    </select>
                </div>

                <div className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 px-4 py-3">
                    <input
                        id={`pack-requires-approval-${draft.key || "new"}`}
                        type="checkbox"
                        checked={draft.requires_approval}
                        onChange={(e) => onChange({ ...draft, requires_approval: e.target.checked })}
                        className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                    />
                    <label
                        htmlFor={`pack-requires-approval-${draft.key || "new"}`}
                        className="text-sm text-slate-700 dark:text-slate-300"
                    >
                        Requires approval before execution
                    </label>
                </div>
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Summary
                </label>
                <textarea
                    value={draft.summary}
                    onChange={(e) => onChange({ ...draft, summary: e.target.value })}
                    rows={2}
                    placeholder="Brief description of what this pack does"
                    className={inputClass}
                />
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Incident Types (comma-separated)
                </label>
                <input
                    type="text"
                    value={incidentTypesRaw}
                    onChange={(e) => handleIncidentTypesChange(e.target.value)}
                    placeholder="e.g. email_queue_backlog_pressure, dispatcher_backlog_pressure"
                    className={inputClass}
                />
                {draft.incident_types.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                        {draft.incident_types.map((t) => (
                            <span
                                key={t}
                                className="inline-block rounded-full bg-slate-200 dark:bg-slate-600 px-2 py-0.5 text-xs text-slate-800 dark:text-slate-200"
                            >
                                {t}
                            </span>
                        ))}
                    </div>
                )}
            </div>

            <div className="flex gap-2 pt-1">
                <button
                    type="button"
                    onClick={onSave}
                    disabled={!draft.key.trim() || !draft.name.trim()}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-400 text-white rounded-md text-sm font-medium transition-colors"
                >
                    {isNew ? "Add Pack" : "Save Changes"}
                </button>
                <button
                    type="button"
                    onClick={onCancel}
                    className="px-4 py-2 bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-600 rounded-md text-sm font-medium transition-colors"
                >
                    Cancel
                </button>
            </div>
        </div>
    );
};

const riskBadge = (tier: string) => {
    const map: Record<string, string> = {
        low: "bg-green-100 text-green-800 border-green-300 dark:bg-green-900/30 dark:text-green-300 dark:border-green-700",
        medium: "bg-amber-100 text-amber-800 border-amber-300 dark:bg-amber-900/30 dark:text-amber-300 dark:border-amber-700",
        high: "bg-rose-100 text-rose-800 border-rose-300 dark:bg-rose-900/30 dark:text-rose-300 dark:border-rose-700",
    };
    return map[tier] ?? "bg-slate-100 text-slate-700 border-slate-300 dark:bg-slate-700 dark:text-slate-300 dark:border-slate-600";
};

export const RobotSREPolicyPanel: React.FC<{
    robotSREPolicy: RobotSREPolicyFormData;
    setRobotSREPolicy: React.Dispatch<React.SetStateAction<RobotSREPolicyFormData>>;
    robotSREPolicyLoading: boolean;
    canManageAdmin: boolean;
    loadRobotSREPolicy: () => void;
    saveRobotSREPolicy: () => void;
    loadRobotSREPolicyDefaults: () => void;
}> = ({
    robotSREPolicy,
    setRobotSREPolicy,
    robotSREPolicyLoading,
    canManageAdmin,
    loadRobotSREPolicy,
    saveRobotSREPolicy,
    loadRobotSREPolicyDefaults,
}) => {
        const [editState, setEditState] = useState<EditState>({ mode: "idle" });
        const [deleteConfirmIndex, setDeleteConfirmIndex] = useState<number | null>(null);

        const startAdd = () =>
            setEditState({ mode: "add", draft: emptyPack() });

        const startEdit = (index: number) =>
            setEditState({
                mode: "edit",
                index,
                draft: { ...robotSREPolicy.remediation_packs[index] },
            });

        const commitAdd = () => {
            if (editState.mode !== "add") return;
            setRobotSREPolicy((prev) => ({
                ...prev,
                remediation_packs: [...prev.remediation_packs, editState.draft],
            }));
            setEditState({ mode: "idle" });
        };

        const commitEdit = () => {
            if (editState.mode !== "edit") return;
            const packs = [...robotSREPolicy.remediation_packs];
            packs[editState.index] = editState.draft;
            setRobotSREPolicy((prev) => ({ ...prev, remediation_packs: packs }));
            setEditState({ mode: "idle" });
        };

        const removePack = (index: number) => {
            setRobotSREPolicy((prev) => ({
                ...prev,
                remediation_packs: prev.remediation_packs.filter((_, i) => i !== index),
            }));
            setDeleteConfirmIndex(null);
        };

        const updateDraft = (updated: RobotSRERemediationPackFormData) => {
            if (editState.mode === "add") {
                setEditState({ mode: "add", draft: updated });
            } else if (editState.mode === "edit") {
                setEditState({ mode: "edit", index: editState.index, draft: updated });
            }
        };

        return (
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6 space-y-6">
                {/* Header */}
                <div className="flex items-start justify-between">
                    <div>
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                            Robot SRE Policy
                        </h2>
                        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                            Configure posture settings and manage guided remediation packs for the
                            SRE Smart Bot subsystem.
                        </p>
                    </div>
                    <div className="flex gap-2 shrink-0">
                        <button
                            type="button"
                            onClick={loadRobotSREPolicy}
                            disabled={robotSREPolicyLoading}
                            className="px-3 py-1.5 bg-slate-700 hover:bg-slate-800 disabled:bg-slate-400 text-white rounded-md text-sm font-medium transition-colors"
                        >
                            {robotSREPolicyLoading ? "Loading…" : "Reload"}
                        </button>
                        <button
                            type="button"
                            onClick={loadRobotSREPolicyDefaults}
                            className="px-3 py-1.5 bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-600 rounded-md text-sm font-medium transition-colors"
                        >
                            Load Defaults
                        </button>
                    </div>
                </div>

                {/* Posture settings */}
                <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-4">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
                        Policy Posture
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-4 py-3">
                            <input
                                id="sre-enabled"
                                type="checkbox"
                                checked={robotSREPolicy.enabled}
                                onChange={(e) =>
                                    setRobotSREPolicy((prev) => ({ ...prev, enabled: e.target.checked }))
                                }
                                className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                            />
                            <label htmlFor="sre-enabled" className="text-sm text-slate-700 dark:text-slate-300">
                                Enable Robot SRE subsystem
                            </label>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                Environment Mode
                            </label>
                            <select
                                value={robotSREPolicy.environment_mode}
                                onChange={(e) =>
                                    setRobotSREPolicy((prev) => ({
                                        ...prev,
                                        environment_mode: e.target.value,
                                    }))
                                }
                                className={inputClass}
                            >
                                <option value="observe_only">observe_only</option>
                                <option value="guided">guided</option>
                                <option value="auto">auto</option>
                            </select>
                        </div>

                        {(
                            [
                                ["auto_observe_enabled", "Auto-Observe"],
                                ["auto_notify_enabled", "Auto-Notify"],
                                ["auto_contain_enabled", "Auto-Contain"],
                                ["auto_recover_enabled", "Auto-Recover"],
                                ["require_approval_for_recover", "Require Approval for Recover"],
                                ["require_approval_for_disruptive", "Require Approval for Disruptive"],
                            ] as [keyof RobotSREPolicyFormData, string][]
                        ).map(([field, label]) => (
                            <div
                                key={field}
                                className="flex items-center gap-3 rounded-md border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-4 py-3"
                            >
                                <input
                                    id={`sre-${field}`}
                                    type="checkbox"
                                    checked={robotSREPolicy[field] as boolean}
                                    onChange={(e) =>
                                        setRobotSREPolicy((prev) => ({
                                            ...prev,
                                            [field]: e.target.checked,
                                        }))
                                    }
                                    className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                                />
                                <label
                                    htmlFor={`sre-${field}`}
                                    className="text-sm text-slate-700 dark:text-slate-300"
                                >
                                    {label}
                                </label>
                            </div>
                        ))}

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                Duplicate Alert Suppression (seconds)
                            </label>
                            <input
                                type="number"
                                min={0}
                                value={robotSREPolicy.duplicate_alert_suppression_seconds}
                                onChange={(e) =>
                                    setRobotSREPolicy((prev) => ({
                                        ...prev,
                                        duplicate_alert_suppression_seconds: parseInt(e.target.value || "0", 10),
                                    }))
                                }
                                className={inputClass}
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                Action Cooldown (seconds)
                            </label>
                            <input
                                type="number"
                                min={0}
                                value={robotSREPolicy.action_cooldown_seconds}
                                onChange={(e) =>
                                    setRobotSREPolicy((prev) => ({
                                        ...prev,
                                        action_cooldown_seconds: parseInt(e.target.value || "0", 10),
                                    }))
                                }
                                className={inputClass}
                            />
                        </div>
                    </div>
                </div>

                {/* Remediation Packs */}
                <div>
                    <div className="flex items-center justify-between mb-3">
                        <div>
                            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                                Guided Remediation Packs
                            </h3>
                            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                Packs define automated remediation playbooks available in SRE incident workspaces.
                            </p>
                        </div>
                        {canManageAdmin && editState.mode === "idle" && (
                            <button
                                type="button"
                                onClick={startAdd}
                                className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium transition-colors"
                            >
                                + Add Pack
                            </button>
                        )}
                    </div>

                    {editState.mode === "add" && (
                        <div className="mb-4">
                            <PackForm
                                draft={editState.draft}
                                isNew
                                onChange={updateDraft}
                                onSave={commitAdd}
                                onCancel={() => setEditState({ mode: "idle" })}
                            />
                        </div>
                    )}

                    {robotSREPolicy.remediation_packs.length === 0 && editState.mode !== "add" && (
                        <div className="rounded-md border border-dashed border-slate-300 dark:border-slate-600 bg-slate-50 dark:bg-slate-900/40 px-4 py-6 text-center">
                            <p className="text-sm text-slate-500 dark:text-slate-400">
                                No remediation packs configured.{" "}
                                {canManageAdmin && (
                                    <button
                                        type="button"
                                        onClick={startAdd}
                                        className="text-blue-600 dark:text-blue-400 hover:underline"
                                    >
                                        Add the first pack
                                    </button>
                                )}
                                {!canManageAdmin && "Contact an admin to add packs."}
                            </p>
                        </div>
                    )}

                    <div className="space-y-3">
                        {robotSREPolicy.remediation_packs.map((pack, index) => {
                            if (editState.mode === "edit" && editState.index === index) {
                                return (
                                    <PackForm
                                        key={pack.key}
                                        draft={editState.draft}
                                        isNew={false}
                                        onChange={updateDraft}
                                        onSave={commitEdit}
                                        onCancel={() => setEditState({ mode: "idle" })}
                                    />
                                );
                            }

                            return (
                                <div
                                    key={pack.key}
                                    className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-4"
                                >
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="min-w-0 flex-1">
                                            <div className="flex flex-wrap items-center gap-2 mb-1">
                                                <span className="font-mono text-sm font-medium text-slate-900 dark:text-white">
                                                    {pack.key}
                                                </span>
                                                <span className="text-xs text-slate-500 dark:text-slate-400">
                                                    v{pack.version}
                                                </span>
                                                <span
                                                    className={`inline-block rounded-full border px-2 py-0.5 text-xs font-medium ${riskBadge(pack.risk_tier)}`}
                                                >
                                                    {pack.risk_tier}
                                                </span>
                                                {pack.requires_approval && (
                                                    <span className="inline-block rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                                                        approval required
                                                    </span>
                                                )}
                                            </div>
                                            <p className="text-sm font-medium text-slate-800 dark:text-slate-200">
                                                {pack.name}
                                            </p>
                                            {pack.summary && (
                                                <p className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                                                    {pack.summary}
                                                </p>
                                            )}
                                            {pack.incident_types.length > 0 && (
                                                <div className="mt-2 flex flex-wrap gap-1">
                                                    {pack.incident_types.map((t) => (
                                                        <span
                                                            key={t}
                                                            className="inline-block rounded-full bg-slate-200 dark:bg-slate-600 px-2 py-0.5 text-xs text-slate-700 dark:text-slate-300"
                                                        >
                                                            {t}
                                                        </span>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                        {canManageAdmin && (
                                            <div className="flex gap-1 shrink-0">
                                                <button
                                                    type="button"
                                                    onClick={() => startEdit(index)}
                                                    disabled={editState.mode !== "idle"}
                                                    className="px-2.5 py-1.5 text-xs font-medium bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-600 disabled:opacity-40 rounded-md transition-colors"
                                                >
                                                    Edit
                                                </button>
                                                {deleteConfirmIndex === index ? (
                                                    <>
                                                        <button
                                                            type="button"
                                                            onClick={() => removePack(index)}
                                                            className="px-2.5 py-1.5 text-xs font-medium bg-rose-600 hover:bg-rose-700 text-white rounded-md transition-colors"
                                                        >
                                                            Confirm
                                                        </button>
                                                        <button
                                                            type="button"
                                                            onClick={() => setDeleteConfirmIndex(null)}
                                                            className="px-2.5 py-1.5 text-xs font-medium bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-600 rounded-md transition-colors"
                                                        >
                                                            Cancel
                                                        </button>
                                                    </>
                                                ) : (
                                                    <button
                                                        type="button"
                                                        onClick={() => setDeleteConfirmIndex(index)}
                                                        disabled={editState.mode !== "idle"}
                                                        className="px-2.5 py-1.5 text-xs font-medium bg-white dark:bg-slate-700 border border-rose-300 dark:border-rose-700 text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-900/20 disabled:opacity-40 rounded-md transition-colors"
                                                    >
                                                        Delete
                                                    </button>
                                                )}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </div>

                {/* Save */}
                {canManageAdmin && (
                    <div className="flex justify-end pt-2 border-t border-slate-200 dark:border-slate-700">
                        <button
                            type="button"
                            onClick={saveRobotSREPolicy}
                            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors"
                        >
                            Save Robot SRE Policy
                        </button>
                    </div>
                )}
            </div>
        );
    };
