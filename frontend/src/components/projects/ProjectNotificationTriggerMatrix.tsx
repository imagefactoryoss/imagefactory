import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import {
  BuildNotificationChannel,
  BuildNotificationRecipientPolicy,
  BuildNotificationSeverity,
  BuildNotificationTriggerID,
  ProjectNotificationTriggerPreference,
  projectNotificationTriggerService,
} from '@/services/projectNotificationTriggerService'
import { userService } from '@/services/userService'
import { useTenantStore } from '@/store/tenant'

interface Props {
  projectId: string
  canEdit: boolean
}

type TriggerCatalogItem = {
  id: BuildNotificationTriggerID
  name: string
  description: string
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

const TRIGGER_CATALOG: TriggerCatalogItem[] = [
  { id: 'BN-001', name: 'Build queued', description: 'Build enters queued state.' },
  { id: 'BN-002', name: 'Build started', description: 'First execution starts.' },
  { id: 'BN-003', name: 'Build completed', description: 'Execution succeeds.' },
  { id: 'BN-004', name: 'Build failed', description: 'Execution fails for any reason.' },
  { id: 'BN-005', name: 'Build cancelled', description: 'Cancellation is finalized.' },
  { id: 'BN-006', name: 'Retry started', description: 'Retry execution begins (attempt > 1).' },
  { id: 'BN-007', name: 'Retry failed', description: 'Retry execution fails.' },
  { id: 'BN-008', name: 'Retry succeeded', description: 'Retry execution succeeds.' },
  { id: 'BN-009', name: 'Recovered from stuck/orphaned', description: 'Sweeper/subscriber recovered stale state.' },
  { id: 'BN-010', name: 'Preflight blocked', description: 'Build blocked by preflight validation.' },
]

const RECIPIENT_OPTIONS: { value: BuildNotificationRecipientPolicy; label: string }[] = [
  { value: 'initiator', label: 'Initiator' },
  { value: 'project_members', label: 'Project Members' },
  { value: 'tenant_admins', label: 'Tenant Admins' },
  { value: 'custom_users', label: 'Custom Users' },
]

const SEVERITY_OPTIONS: { value: BuildNotificationSeverity; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'normal', label: 'Normal' },
  { value: 'high', label: 'High' },
]

type UserOption = {
  id: string
  label: string
}

const ProjectNotificationTriggerMatrix: React.FC<Props> = ({ projectId, canEdit }) => {
  const selectedTenantId = useTenantStore((state) => state.selectedTenantId)
  const [preferences, setPreferences] = useState<ProjectNotificationTriggerPreference[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [tenantUserOptions, setTenantUserOptions] = useState<UserOption[]>([])
  const [customUserSearch, setCustomUserSearch] = useState<Record<string, string>>({})

  const byTrigger = useMemo(() => {
    const map = new Map<BuildNotificationTriggerID, ProjectNotificationTriggerPreference>()
    preferences.forEach((pref) => map.set(pref.trigger_id, pref))
    return map
  }, [preferences])

  const load = async () => {
    try {
      setLoading(true)
      const response = await projectNotificationTriggerService.getProjectNotificationTriggers(projectId)
      setPreferences(response.preferences || [])
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to load notification trigger preferences'
      toast.error(message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [projectId])

  useEffect(() => {
    const loadTenantUsers = async () => {
      if (!selectedTenantId) {
        setTenantUserOptions([])
        return
      }
      try {
        const response = await userService.listUsers(selectedTenantId, 1, 200)
        const users = (response?.users || []).map((user: any) => {
          const first = user.first_name || ''
          const last = user.last_name || ''
          const fullName = `${first} ${last}`.trim()
          return {
            id: user.id,
            label: fullName ? `${fullName} (${user.email})` : user.email,
          }
        })
        setTenantUserOptions(users)
      } catch (_err) {
        setTenantUserOptions([])
      }
    }
    loadTenantUsers()
  }, [selectedTenantId])

  const tenantUserLabelByID = useMemo(() => {
    const map = new Map<string, string>()
    tenantUserOptions.forEach((user) => {
      map.set(user.id, user.label)
    })
    return map
  }, [tenantUserOptions])

  const updatePref = (triggerId: BuildNotificationTriggerID, patch: Partial<ProjectNotificationTriggerPreference>) => {
    setPreferences((prev) =>
      prev.map((pref) =>
        pref.trigger_id === triggerId
          ? {
              ...pref,
              ...patch,
            }
          : pref,
      ),
    )
  }

  const toggleChannel = (
    triggerId: BuildNotificationTriggerID,
    channel: BuildNotificationChannel,
    checked: boolean,
  ) => {
    const pref = byTrigger.get(triggerId)
    if (!pref) return
    const nextChannels = checked ? Array.from(new Set([...pref.channels, channel])) : pref.channels.filter((c) => c !== channel)
    updatePref(triggerId, { channels: nextChannels })
  }

  const save = async () => {
    try {
      setSaving(true)
      const invalidEnabledTrigger = preferences.find((pref) => pref.enabled && pref.channels.length === 0)
      if (invalidEnabledTrigger) {
        toast.error(`Select at least one channel for ${invalidEnabledTrigger.trigger_id}`)
        return
      }

      const invalidCustomUser = preferences.find(
        (pref) =>
          pref.recipient_policy === 'custom_users' &&
          (pref.custom_recipient_user_ids || []).some((id) => !UUID_RE.test(id)),
      )
      if (invalidCustomUser) {
        toast.error(`Invalid custom user UUID in ${invalidCustomUser.trigger_id}`)
        return
      }

      const payload = preferences.map((pref) => ({
        ...pref,
      }))
      const response = await projectNotificationTriggerService.updateProjectNotificationTriggers(projectId, payload)
      setPreferences(response.preferences || [])
      toast.success('Notification trigger preferences saved')
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to save notification trigger preferences'
      toast.error(message)
    } finally {
      setSaving(false)
    }
  }

  const resetToInherited = async (triggerId: BuildNotificationTriggerID) => {
    try {
      const response = await projectNotificationTriggerService.deleteProjectNotificationTrigger(projectId, triggerId)
      setPreferences(response.preferences || [])
      toast.success(`${triggerId} reset to inherited defaults`)
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to reset trigger preference'
      toast.error(message)
    }
  }

  if (loading) {
    return (
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-6">
        <div className="animate-pulse text-sm text-slate-500 dark:text-slate-400">Loading notification trigger matrix...</div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-6">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Build Notification Trigger Matrix</h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
          Configure which build events notify users for this project, over which channels, and to whom.
        </p>
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          Current source: project-level preferences. Inheritance markers will be added in a follow-up phase.
        </p>
      </div>

      <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
          <thead className="bg-slate-50 dark:bg-slate-900/40">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Trigger</th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Enabled</th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Channels</th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Recipients</th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Severity Override</th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
            {TRIGGER_CATALOG.map((trigger) => {
              const pref = byTrigger.get(trigger.id)
              if (!pref) return null

              return (
                <tr key={trigger.id}>
                  <td className="px-4 py-3 align-top">
                    <div className="text-sm font-medium text-slate-900 dark:text-white">
                      {trigger.id} - {trigger.name}
                    </div>
                    <div className="mt-1">
                      <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                        {pref.source || 'project'}
                      </span>
                    </div>
                    <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">{trigger.description}</div>
                  </td>
                  <td className="px-4 py-3 align-top">
                    <label className="inline-flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                      <input
                        type="checkbox"
                        checked={pref.enabled}
                        disabled={!canEdit}
                        onChange={(e) => updatePref(trigger.id, { enabled: e.target.checked })}
                        className="h-4 w-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                      />
                      Enabled
                    </label>
                  </td>
                  <td className="px-4 py-3 align-top">
                    <div className="space-y-2">
                      <label className="inline-flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                        <input
                          type="checkbox"
                          checked={pref.channels.includes('in_app')}
                          disabled={!canEdit || !pref.enabled}
                          onChange={(e) => toggleChannel(trigger.id, 'in_app', e.target.checked)}
                          className="h-4 w-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                        />
                        In-app
                      </label>
                      <label className="inline-flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                        <input
                          type="checkbox"
                          checked={pref.channels.includes('email')}
                          disabled={!canEdit || !pref.enabled}
                          onChange={(e) => toggleChannel(trigger.id, 'email', e.target.checked)}
                          className="h-4 w-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                        />
                        Email
                      </label>
                    </div>
                  </td>
                  <td className="px-4 py-3 align-top">
                    <select
                      value={pref.recipient_policy}
                      disabled={!canEdit || !pref.enabled}
                      onChange={(e) =>
                        updatePref(trigger.id, {
                          recipient_policy: e.target.value as BuildNotificationRecipientPolicy,
                          custom_recipient_user_ids: e.target.value === 'custom_users' ? pref.custom_recipient_user_ids : [],
                        })
                      }
                      className="w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-2 py-1.5 text-sm text-slate-900 dark:text-white"
                    >
                      {RECIPIENT_OPTIONS.map((opt) => (
                        <option key={opt.value} value={opt.value}>
                          {opt.label}
                        </option>
                      ))}
                    </select>
                    {pref.recipient_policy === 'custom_users' ? (
                      <div className="mt-2">
                        <label className="mb-1 block text-xs font-medium text-slate-600 dark:text-slate-300">
                          Tenant users
                        </label>
                        {(pref.custom_recipient_user_ids || []).length > 0 ? (
                          <div className="mb-2 flex flex-wrap gap-1.5">
                            {(pref.custom_recipient_user_ids || []).map((id) => (
                              <span
                                key={`${trigger.id}-${id}`}
                                className="inline-flex items-center gap-1 rounded-full border border-slate-300 dark:border-slate-600 bg-slate-100 dark:bg-slate-700 px-2 py-0.5 text-[11px] text-slate-700 dark:text-slate-200"
                              >
                                <span className="max-w-[220px] truncate">{tenantUserLabelByID.get(id) || id}</span>
                                {canEdit && pref.enabled ? (
                                  <button
                                    type="button"
                                    onClick={() =>
                                      updatePref(trigger.id, {
                                        custom_recipient_user_ids: (pref.custom_recipient_user_ids || []).filter((userID) => userID !== id),
                                      })
                                    }
                                    className="text-slate-500 transition-colors hover:text-red-600 dark:text-slate-300 dark:hover:text-red-400"
                                  >
                                    ×
                                  </button>
                                ) : null}
                              </span>
                            ))}
                          </div>
                        ) : null}

                        <input
                          type="text"
                          disabled={!canEdit || !pref.enabled}
                          value={customUserSearch[trigger.id] || ''}
                          onChange={(e) =>
                            setCustomUserSearch((prev) => ({
                              ...prev,
                              [trigger.id]: e.target.value,
                            }))
                          }
                          placeholder="Search tenant users by name or email..."
                          className="w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-2 py-1.5 text-xs text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-400"
                        />

                        <div className="mt-2 max-h-28 space-y-1 overflow-y-auto rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 p-2">
                          {tenantUserOptions
                            .filter((user) => {
                              const selected = (pref.custom_recipient_user_ids || []).includes(user.id)
                              if (selected) return false
                              const q = (customUserSearch[trigger.id] || '').trim().toLowerCase()
                              if (!q) return true
                              return user.label.toLowerCase().includes(q) || user.id.toLowerCase().includes(q)
                            })
                            .slice(0, 8)
                            .map((user) => (
                              <button
                                key={user.id}
                                type="button"
                                disabled={!canEdit || !pref.enabled}
                                onClick={() => {
                                  const existing = pref.custom_recipient_user_ids || []
                                  updatePref(trigger.id, {
                                    custom_recipient_user_ids: Array.from(new Set([...existing, user.id])),
                                  })
                                }}
                                className="flex w-full items-center justify-between rounded px-2 py-1 text-left text-xs text-slate-700 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-slate-200 dark:hover:bg-slate-600"
                              >
                                <span className="truncate">{user.label}</span>
                                <span className="ml-2 text-[10px] text-slate-400 dark:text-slate-400">Add</span>
                              </button>
                            ))}
                          {tenantUserOptions.length === 0 ? (
                            <div className="text-xs text-slate-500 dark:text-slate-400">No tenant users available.</div>
                          ) : null}
                        </div>
                      </div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 align-top">
                    <select
                      value={pref.severity_override || ''}
                      disabled={!canEdit || !pref.enabled}
                      onChange={(e) =>
                        updatePref(trigger.id, {
                          severity_override: (e.target.value || undefined) as BuildNotificationSeverity | undefined,
                        })
                      }
                      className="w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 px-2 py-1.5 text-sm text-slate-900 dark:text-white"
                    >
                      <option value="">Default</option>
                      {SEVERITY_OPTIONS.map((opt) => (
                        <option key={opt.value} value={opt.value}>
                          {opt.label}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="px-4 py-3 align-top">
                    {pref.source === 'project' ? (
                      <button
                        type="button"
                        disabled={!canEdit}
                        onClick={() => resetToInherited(trigger.id)}
                        className="rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        Reset
                      </button>
                    ) : (
                      <span className="text-xs text-slate-400 dark:text-slate-500">Inherited</span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-end gap-3">
        <button
          type="button"
          onClick={load}
          className="rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-700"
        >
          Refresh
        </button>
        <button
          type="button"
          onClick={save}
          disabled={!canEdit || saving}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {saving ? 'Saving...' : 'Save Trigger Preferences'}
        </button>
      </div>
    </div>
  )
}

export default ProjectNotificationTriggerMatrix
