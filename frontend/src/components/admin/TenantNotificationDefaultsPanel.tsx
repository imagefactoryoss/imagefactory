import React, { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";

import { adminService } from "@/services/adminService";
import { api } from "@/services/api";

type TriggerID =
  | "BN-001"
  | "BN-002"
  | "BN-003"
  | "BN-004"
  | "BN-005"
  | "BN-006"
  | "BN-007"
  | "BN-008"
  | "BN-009"
  | "BN-010";

type Channel = "in_app" | "email";
type RecipientPolicy =
  | "initiator"
  | "project_members"
  | "tenant_admins"
  | "custom_users";
type Severity = "low" | "normal" | "high";
type PreferenceSource = "system" | "tenant" | "project";

interface TenantNotificationPreference {
  trigger_id: TriggerID;
  source?: PreferenceSource;
  enabled: boolean;
  channels: Channel[];
  recipient_policy: RecipientPolicy;
  custom_recipient_user_ids?: string[];
  severity_override?: Severity;
}

interface TenantOption {
  id: string;
  name: string;
}

const TRIGGER_CATALOG: Array<{ id: TriggerID; name: string }> = [
  { id: "BN-001", name: "Build queued" },
  { id: "BN-002", name: "Build started" },
  { id: "BN-003", name: "Build completed" },
  { id: "BN-004", name: "Build failed" },
  { id: "BN-005", name: "Build cancelled" },
  { id: "BN-006", name: "Retry started" },
  { id: "BN-007", name: "Retry failed" },
  { id: "BN-008", name: "Retry succeeded" },
  { id: "BN-009", name: "Recovered from stuck/orphaned" },
  { id: "BN-010", name: "Preflight blocked" },
];

const TenantNotificationDefaultsPanel: React.FC = () => {
  const [tenants, setTenants] = useState<TenantOption[]>([]);
  const [selectedTenantId, setSelectedTenantId] = useState<string>("");
  const [preferences, setPreferences] = useState<TenantNotificationPreference[]>(
    [],
  );
  const [loadingTenants, setLoadingTenants] = useState(false);
  const [loadingPrefs, setLoadingPrefs] = useState(false);
  const [saving, setSaving] = useState(false);

  const byTrigger = useMemo(() => {
    const map = new Map<TriggerID, TenantNotificationPreference>();
    preferences.forEach((pref) => map.set(pref.trigger_id, pref));
    return map;
  }, [preferences]);

  useEffect(() => {
    const loadTenants = async () => {
      try {
        setLoadingTenants(true);
        const response = await adminService.getTenants({ page: 1, limit: 200 });
        const rows = (response.data || []).map((tenant) => ({
          id: tenant.id,
          name: tenant.name,
        }));
        setTenants(rows);
        if (rows.length > 0) {
          setSelectedTenantId(rows[0].id);
        }
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : "Failed to load tenants for notification defaults";
        toast.error(message);
      } finally {
        setLoadingTenants(false);
      }
    };
    loadTenants();
  }, []);

  useEffect(() => {
    const loadPrefs = async () => {
      if (!selectedTenantId) {
        setPreferences([]);
        return;
      }
      try {
        setLoadingPrefs(true);
        const response = await api.get(
          `/admin/tenants/${selectedTenantId}/notification-triggers`,
        );
        setPreferences(response.data?.preferences || []);
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : "Failed to load tenant notification defaults";
        toast.error(message);
      } finally {
        setLoadingPrefs(false);
      }
    };
    loadPrefs();
  }, [selectedTenantId]);

  const updatePref = (
    triggerId: TriggerID,
    patch: Partial<TenantNotificationPreference>,
  ) => {
    setPreferences((prev) =>
      prev.map((pref) =>
        pref.trigger_id === triggerId
          ? {
              ...pref,
              ...patch,
            }
          : pref,
      ),
    );
  };

  const toggleChannel = (triggerId: TriggerID, channel: Channel, checked: boolean) => {
    const pref = byTrigger.get(triggerId);
    if (!pref) return;
    const channels = checked
      ? Array.from(new Set([...(pref.channels || []), channel]))
      : (pref.channels || []).filter((c) => c !== channel);
    updatePref(triggerId, { channels });
  };

  const save = async () => {
    if (!selectedTenantId) return;
    try {
      setSaving(true);
      await api.put(`/admin/tenants/${selectedTenantId}/notification-triggers`, {
        preferences: preferences.map((pref) => ({
          trigger_id: pref.trigger_id,
          enabled: pref.enabled,
          channels: pref.channels,
          recipient_policy: pref.recipient_policy,
          custom_recipient_user_ids: pref.custom_recipient_user_ids || [],
          severity_override: pref.severity_override,
        })),
      });
      toast.success("Tenant notification defaults saved");
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "Failed to save tenant notification defaults";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-slate-800">
        <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100">
          Tenant Notification Defaults
        </h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
          Configure tenant-level defaults for build notification triggers from the
          system admin surface.
        </p>
      </div>

      <div className="rounded-lg border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-slate-800">
        <label className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">
          Tenant
        </label>
        <select
          value={selectedTenantId}
          disabled={loadingTenants || tenants.length === 0}
          onChange={(e) => setSelectedTenantId(e.target.value)}
          className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100"
        >
          {tenants.length === 0 ? (
            <option value="">No tenants found</option>
          ) : null}
          {tenants.map((tenant) => (
            <option key={tenant.id} value={tenant.id}>
              {tenant.name}
            </option>
          ))}
        </select>
      </div>

      <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800">
        {loadingPrefs ? (
          <div className="p-6 text-sm text-slate-500 dark:text-slate-400">
            Loading tenant notification defaults...
          </div>
        ) : (
          <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
            <thead className="bg-slate-50 dark:bg-slate-900/40">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Trigger
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Enabled
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Channels
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Recipients
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  Severity
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
              {TRIGGER_CATALOG.map((trigger) => {
                const pref = byTrigger.get(trigger.id);
                if (!pref) return null;
                return (
                  <tr key={trigger.id}>
                    <td className="px-4 py-3 align-top">
                      <div className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        {trigger.id} - {trigger.name}
                      </div>
                      <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        Source: {pref.source || "tenant"}
                      </div>
                    </td>
                    <td className="px-4 py-3 align-top">
                      <input
                        type="checkbox"
                        checked={pref.enabled}
                        onChange={(e) =>
                          updatePref(trigger.id, { enabled: e.target.checked })
                        }
                        className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-700"
                      />
                    </td>
                    <td className="px-4 py-3 align-top">
                      <div className="space-y-2">
                        <label className="inline-flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                          <input
                            type="checkbox"
                            checked={pref.channels.includes("in_app")}
                            disabled={!pref.enabled}
                            onChange={(e) =>
                              toggleChannel(
                                trigger.id,
                                "in_app",
                                e.target.checked,
                              )
                            }
                            className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-700"
                          />
                          In-app
                        </label>
                        <label className="inline-flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                          <input
                            type="checkbox"
                            checked={pref.channels.includes("email")}
                            disabled={!pref.enabled}
                            onChange={(e) =>
                              toggleChannel(trigger.id, "email", e.target.checked)
                            }
                            className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-700"
                          />
                          Email
                        </label>
                      </div>
                    </td>
                    <td className="px-4 py-3 align-top">
                      <select
                        value={pref.recipient_policy}
                        disabled={!pref.enabled}
                        onChange={(e) =>
                          updatePref(trigger.id, {
                            recipient_policy: e.target
                              .value as TenantNotificationPreference["recipient_policy"],
                          })
                        }
                        className="w-full rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm text-slate-900 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100"
                      >
                        <option value="initiator">Initiator</option>
                        <option value="project_members">Project Members</option>
                        <option value="tenant_admins">Tenant Admins</option>
                        <option value="custom_users">Custom Users</option>
                      </select>
                    </td>
                    <td className="px-4 py-3 align-top">
                      <select
                        value={pref.severity_override || ""}
                        disabled={!pref.enabled}
                        onChange={(e) =>
                          updatePref(trigger.id, {
                            severity_override: (e.target.value ||
                              undefined) as Severity | undefined,
                          })
                        }
                        className="w-full rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm text-slate-900 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100"
                      >
                        <option value="">Default</option>
                        <option value="low">Low</option>
                        <option value="normal">Normal</option>
                        <option value="high">High</option>
                      </select>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      <div className="flex justify-end">
        <button
          type="button"
          disabled={saving || !selectedTenantId}
          onClick={save}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-blue-500 dark:hover:bg-blue-600"
        >
          {saving ? "Saving..." : "Save Tenant Defaults"}
        </button>
      </div>
    </div>
  );
};

export default TenantNotificationDefaultsPanel;
