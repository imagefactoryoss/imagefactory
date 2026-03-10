import Drawer from '@/components/ui/Drawer'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { buildService } from '@/services/buildService'
import { buildTriggerService, BuildTrigger } from '@/services/buildTriggerService'
import { projectService, ProjectWebhookReceipt } from '@/services/projectService'
import { Build } from '@/types'
import React, { useMemo, useState } from 'react'
import toast from 'react-hot-toast'

interface ProjectWebhooksPanelProps {
    projectId: string
    canManage: boolean
}

type ProviderKey = 'github' | 'gitlab'
type DrawerMode = 'create' | 'edit'

const providerOptions: Array<{ value: ProviderKey; label: string }> = [
    { value: 'github', label: 'GitHub' },
    { value: 'gitlab', label: 'GitLab' },
]

const providerHints: Record<ProviderKey, string[]> = {
    github: [
        'Webhook URL: use the provider endpoint below.',
        'Content type: application/json.',
        'Secret: set the same value here and in GitHub webhook settings.',
    ],
    gitlab: [
        'Webhook URL: use the provider endpoint below.',
        'Enable push and merge request events to match selected trigger events.',
        'Secret token: set the same value here and in GitLab webhook settings.',
    ],
}

const eventOptions = ['push', 'pull_request', 'release', 'tag_push']

const extractProviderFromWebhookURL = (url?: string): ProviderKey | null => {
    if (!url) return null
    const match = url.match(/\/webhooks\/([^/]+)\//i)
    const provider = (match?.[1] || '').toLowerCase()
    if (provider === 'github' || provider === 'gitlab') {
        return provider
    }
    return null
}

const formatDateTime = (value?: string): string => {
    if (!value) return 'n/a'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return 'n/a'
    return date.toLocaleString()
}

const copyText = async (value: string) => {
    try {
        await navigator.clipboard.writeText(value)
        toast.success('Copied')
    } catch {
        toast.error('Failed to copy')
    }
}

const buildWebhookReceiverURL = (provider: ProviderKey, projectId: string): string => {
    const origin = window.location.origin
    return `${origin}/api/v1/webhooks/${provider}/${projectId}`
}

const ProjectWebhooksPanel: React.FC<ProjectWebhooksPanelProps> = ({ projectId, canManage }) => {
    const [builds, setBuilds] = useState<Build[]>([])
    const [triggers, setTriggers] = useState<BuildTrigger[]>([])
    const [receipts, setReceipts] = useState<ProjectWebhookReceipt[]>([])
    const [loading, setLoading] = useState(false)
    const [drawerOpen, setDrawerOpen] = useState(false)
    const [drawerMode, setDrawerMode] = useState<DrawerMode>('create')
    const [editingTriggerID, setEditingTriggerID] = useState<string | null>(null)
    const [saving, setSaving] = useState(false)
    const [updatingIDs, setUpdatingIDs] = useState<Record<string, boolean>>({})
    const confirmDialog = useConfirmDialog()

    const [selectedBuildId, setSelectedBuildId] = useState('')
    const [provider, setProvider] = useState<ProviderKey>('github')
    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [webhookSecret, setWebhookSecret] = useState('')
    const [webhookEvents, setWebhookEvents] = useState<string[]>(['push'])

    const loadData = React.useCallback(async () => {
        try {
            setLoading(true)
            const [buildsResp, triggerList, receiptList] = await Promise.all([
                buildService.getBuilds({
                    projectId,
                    page: 1,
                    limit: 100,
                }),
                buildTriggerService.getProjectTriggers(projectId),
                projectService.listProjectWebhookReceipts(projectId, { limit: 20, offset: 0 }),
            ])
            setBuilds(buildsResp.data || [])
            setTriggers(triggerList.filter((trigger) => trigger.triggerType === 'webhook'))
            setReceipts(receiptList)
        } catch (error: any) {
            toast.error(error?.message || 'Failed to load webhook triggers')
        } finally {
            setLoading(false)
        }
    }, [projectId])

    React.useEffect(() => {
        void loadData()
    }, [loadData])

    const buildsByID = useMemo(() => {
        const lookup = new Map<string, Build>()
        builds.forEach((build) => lookup.set(build.id, build))
        return lookup
    }, [builds])

    const receiverURL = useMemo(() => buildWebhookReceiverURL(provider, projectId), [provider, projectId])
    const resetForm = () => {
        setSelectedBuildId('')
        setProvider('github')
        setName('')
        setDescription('')
        setWebhookSecret('')
        setWebhookEvents(['push'])
        setEditingTriggerID(null)
        setDrawerMode('create')
    }

    const closeDrawer = () => {
        setDrawerOpen(false)
        resetForm()
    }

    const openCreateDrawer = () => {
        resetForm()
        setDrawerMode('create')
        setDrawerOpen(true)
    }

    const openEditDrawer = (trigger: BuildTrigger) => {
        setDrawerMode('edit')
        setEditingTriggerID(trigger.id)
        setSelectedBuildId(trigger.buildId)
        setProvider(extractProviderFromWebhookURL(trigger.webhookUrl) || 'github')
        setName(trigger.name || '')
        setDescription(trigger.description || '')
        setWebhookSecret(trigger.webhookSecret || '')
        setWebhookEvents(trigger.webhookEvents && trigger.webhookEvents.length > 0 ? trigger.webhookEvents : ['push'])
        setDrawerOpen(true)
    }

    const toggleEvent = (eventName: string) => {
        setWebhookEvents((prev) => {
            if (prev.includes(eventName)) {
                return prev.filter((value) => value !== eventName)
            }
            return [...prev, eventName]
        })
    }

    const handleSave = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!canManage) return

        const trimmedName = name.trim()
        if (!trimmedName) {
            toast.error('Trigger name is required')
            return
        }
        if (webhookEvents.length === 0) {
            toast.error('Select at least one webhook event')
            return
        }
        if (drawerMode === 'create' && !selectedBuildId && builds.length === 0) {
            toast.error('No project builds found. Create at least one build before adding webhook triggers.')
            return
        }

        try {
            setSaving(true)
            if (drawerMode === 'create') {
                await buildTriggerService.createProjectWebhookTrigger(projectId, {
                    buildId: selectedBuildId || undefined,
                    name: trimmedName,
                    description: description.trim() || undefined,
                    webhookUrl: receiverURL,
                    webhookSecret: webhookSecret.trim() || undefined,
                    webhookEvents,
                })
                toast.success('Webhook trigger created')
            } else if (editingTriggerID) {
                await buildTriggerService.updateProjectWebhookTrigger(projectId, editingTriggerID, {
                    name: trimmedName,
                    description: description.trim(),
                    webhookUrl: receiverURL,
                    webhookSecret: webhookSecret.trim(),
                    webhookEvents,
                })
                toast.success('Webhook trigger updated')
            }
            closeDrawer()
            await loadData()
        } catch (error: any) {
            const message = error?.response?.data?.message || error?.message || 'Failed to save trigger'
            toast.error(message)
        } finally {
            setSaving(false)
        }
    }

    const handleToggleActive = async (trigger: BuildTrigger) => {
        if (!canManage) return
        setUpdatingIDs((prev) => ({ ...prev, [trigger.id]: true }))
        try {
            await buildTriggerService.updateProjectWebhookTrigger(projectId, trigger.id, {
                isActive: !trigger.isActive,
            })
            toast.success(`Trigger ${trigger.isActive ? 'disabled' : 'enabled'}`)
            await loadData()
        } catch (error: any) {
            const message = error?.response?.data?.message || error?.message || 'Failed to update trigger'
            toast.error(message)
        } finally {
            setUpdatingIDs((prev) => {
                const next = { ...prev }
                delete next[trigger.id]
                return next
            })
        }
    }

    const handleDelete = async (trigger: BuildTrigger) => {
        if (!canManage) return
        const confirmed = await confirmDialog({
            title: 'Delete Webhook Trigger',
            message: `Delete webhook trigger "${trigger.name}"?`,
            confirmLabel: 'Delete Trigger',
            destructive: true,
        })
        if (!confirmed) return

        try {
            await buildTriggerService.deleteTrigger(projectId, trigger.id)
            toast.success('Webhook trigger deleted')
            await loadData()
        } catch (error: any) {
            const message = error?.response?.data?.message || error?.message || 'Failed to delete trigger'
            toast.error(message)
        }
    }

    return (
        <div className="space-y-6">
            <div className="rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800">
                <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
                    <div>
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">Webhook Triggers</h3>
                        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                            Configure inbound provider webhooks that trigger builds for this project.
                        </p>
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            type="button"
                            onClick={() => void loadData()}
                            className="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
                        >
                            Refresh
                        </button>
                        {canManage && (
                            <button
                                type="button"
                                onClick={openCreateDrawer}
                                className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-400"
                            >
                                Add Webhook Trigger
                            </button>
                        )}
                    </div>
                </div>

                <div className="mb-6 rounded-lg border border-indigo-200 bg-indigo-50 p-4 dark:border-indigo-700 dark:bg-indigo-900/20">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                        <p className="text-xs font-medium uppercase tracking-wide text-indigo-800 dark:text-indigo-300">Provider Endpoint</p>
                        <select
                            value={provider}
                            onChange={(e) => setProvider(e.target.value as ProviderKey)}
                            className="rounded border border-indigo-300 bg-white px-2 py-1 text-xs text-indigo-900 dark:border-indigo-600 dark:bg-slate-900 dark:text-indigo-200"
                        >
                            {providerOptions.map((option) => (
                                <option key={option.value} value={option.value}>{option.label}</option>
                            ))}
                        </select>
                    </div>
                    <p className="mt-1 break-all font-mono text-xs text-indigo-900 dark:text-indigo-200">{receiverURL}</p>
                    <div className="mt-2 space-y-1 text-xs text-indigo-800 dark:text-indigo-300">
                        {providerHints[provider].map((hint) => (
                            <p key={hint}>{hint}</p>
                        ))}
                    </div>
                    <button
                        type="button"
                        onClick={() => void copyText(receiverURL)}
                        className="mt-2 rounded border border-indigo-300 bg-white px-2 py-1 text-xs font-medium text-indigo-800 hover:bg-indigo-100 dark:border-indigo-600 dark:bg-slate-900 dark:text-indigo-200 dark:hover:bg-indigo-900/30"
                    >
                        Copy Endpoint
                    </button>
                </div>

                {loading ? (
                    <p className="text-sm text-slate-600 dark:text-slate-300">Loading triggers...</p>
                ) : triggers.length === 0 ? (
                    <p className="text-sm text-slate-600 dark:text-slate-300">No webhook triggers configured yet.</p>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                            <thead className="bg-slate-50 dark:bg-slate-900/50">
                                <tr>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Name</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Build</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Provider</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Events</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Status</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Created</th>
                                    {canManage && (
                                        <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Actions</th>
                                    )}
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                                {triggers.map((trigger) => {
                                    const build = buildsByID.get(trigger.buildId)
                                    const buildName = build?.manifest?.name || trigger.buildId.slice(0, 8)
                                    const providerName = extractProviderFromWebhookURL(trigger.webhookUrl) || 'github'
                                    const disabled = Boolean(updatingIDs[trigger.id])
                                    return (
                                        <tr key={trigger.id}>
                                            <td className="px-3 py-3 text-sm font-medium text-slate-900 dark:text-slate-100">{trigger.name}</td>
                                            <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{buildName}</td>
                                            <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{providerName}</td>
                                            <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">
                                                {(trigger.webhookEvents || []).join(', ') || 'all'}
                                            </td>
                                            <td className="px-3 py-3 text-sm">
                                                <span
                                                    className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                                                        trigger.isActive
                                                            ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-200'
                                                            : 'bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200'
                                                    }`}
                                                >
                                                    {trigger.isActive ? 'active' : 'inactive'}
                                                </span>
                                            </td>
                                            <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{formatDateTime(trigger.createdAt)}</td>
                                            {canManage && (
                                                <td className="px-3 py-3 text-right">
                                                    <div className="flex items-center justify-end gap-2">
                                                        <button
                                                            type="button"
                                                            disabled={disabled}
                                                            onClick={() => void handleToggleActive(trigger)}
                                                            className="rounded border border-slate-300 px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-60 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
                                                        >
                                                            {trigger.isActive ? 'Disable' : 'Enable'}
                                                        </button>
                                                        <button
                                                            type="button"
                                                            onClick={() => openEditDrawer(trigger)}
                                                            className="rounded border border-blue-300 px-2 py-1 text-xs font-medium text-blue-700 hover:bg-blue-50 dark:border-blue-700 dark:text-blue-300 dark:hover:bg-blue-900/20"
                                                        >
                                                            Edit
                                                        </button>
                                                        <button
                                                            type="button"
                                                            onClick={() => void handleDelete(trigger)}
                                                            className="rounded border border-red-300 px-2 py-1 text-xs font-medium text-red-700 hover:bg-red-50 dark:border-red-700 dark:text-red-300 dark:hover:bg-red-900/20"
                                                        >
                                                            Delete
                                                        </button>
                                                    </div>
                                                </td>
                                            )}
                                        </tr>
                                    )
                                })}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            <div className="rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800">
                <div className="mb-4 flex items-center justify-between gap-3">
                    <div>
                        <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">Webhook Receipts</h3>
                        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                            Latest deliveries for this project endpoint, including match diagnostics.
                        </p>
                    </div>
                    <button
                        type="button"
                        onClick={() => void loadData()}
                        className="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
                    >
                        Refresh Diagnostics
                    </button>
                </div>
                {receipts.length === 0 ? (
                    <p className="text-sm text-slate-600 dark:text-slate-300">No webhook receipts yet.</p>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                            <thead className="bg-slate-50 dark:bg-slate-900/50">
                                <tr>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Received</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Provider</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Event</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Status</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Matched</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Reason</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                                {receipts.map((receipt) => (
                                    <tr key={receipt.id}>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{formatDateTime(receipt.receivedAt)}</td>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{receipt.provider}</td>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">
                                            {receipt.eventType}
                                            {receipt.eventBranch && (
                                                <span className="ml-2 rounded bg-slate-100 px-1.5 py-0.5 text-xs dark:bg-slate-700">{receipt.eventBranch}</span>
                                            )}
                                        </td>
                                        <td className="px-3 py-3 text-sm">
                                            <span
                                                className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                                                    receipt.status === 'accepted' || receipt.status === 'processed'
                                                        ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-200'
                                                        : 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
                                                }`}
                                            >
                                                {receipt.status}
                                            </span>
                                        </td>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{receipt.matchedTriggerCount}</td>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{receipt.reason || '-'}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            <Drawer
                isOpen={drawerOpen}
                onClose={closeDrawer}
                title={drawerMode === 'create' ? 'Add Webhook Trigger' : 'Edit Webhook Trigger'}
                description="Bind provider webhook events to a specific project build."
                width="lg"
            >
                <form onSubmit={handleSave} className="space-y-5">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Build Template</label>
                        <select
                            value={selectedBuildId}
                            onChange={(e) => setSelectedBuildId(e.target.value)}
                            disabled={drawerMode === 'edit'}
                            className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400"
                        >
                            <option value="">Auto-select latest project build</option>
                            {builds.map((build) => (
                                <option key={build.id} value={build.id}>
                                    {build.manifest?.name || build.id}
                                </option>
                            ))}
                        </select>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                            {drawerMode === 'create'
                                ? 'Leave empty to let backend bind this trigger to the latest build in the project.'
                                : 'Build template cannot be changed for an existing trigger.'}
                        </p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Provider *</label>
                        <select
                            value={provider}
                            onChange={(e) => setProvider(e.target.value as ProviderKey)}
                            className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400"
                        >
                            {providerOptions.map((option) => (
                                <option key={option.value} value={option.value}>{option.label}</option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label htmlFor="project-webhook-trigger-name" className="block text-sm font-medium text-slate-700 dark:text-slate-300">Trigger Name *</label>
                        <input
                            id="project-webhook-trigger-name"
                            type="text"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            placeholder="e.g., GitHub push trigger"
                            className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Description</label>
                        <textarea
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                            rows={2}
                            placeholder="Optional description"
                            className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Webhook Endpoint</label>
                        <div className="mt-1 flex gap-2">
                            <input
                                type="text"
                                value={receiverURL}
                                readOnly
                                className="block w-full rounded-md border border-slate-300 bg-slate-50 px-3 py-2 text-xs font-mono text-slate-700 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200"
                            />
                            <button
                                type="button"
                                onClick={() => void copyText(receiverURL)}
                                className="rounded border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
                            >
                                Copy
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Webhook Secret</label>
                        <input
                            type="password"
                            autoComplete="new-password"
                            value={webhookSecret}
                            onChange={(e) => setWebhookSecret(e.target.value)}
                            placeholder="Optional secret/token to validate signatures"
                            className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Events *</label>
                        <div className="mt-2 grid grid-cols-2 gap-2">
                            {eventOptions.map((eventName) => (
                                <label key={eventName} className="inline-flex items-center gap-2 rounded border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200">
                                    <input
                                        type="checkbox"
                                        checked={webhookEvents.includes(eventName)}
                                        onChange={() => toggleEvent(eventName)}
                                        className="h-4 w-4 rounded border-slate-400 text-blue-600 focus:ring-blue-500"
                                    />
                                    {eventName}
                                </label>
                            ))}
                        </div>
                    </div>

                    <div className="flex justify-end gap-3 pt-2">
                        <button
                            type="button"
                            onClick={closeDrawer}
                            className="rounded border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={saving}
                            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-60 dark:bg-blue-500 dark:hover:bg-blue-400"
                        >
                            {saving ? 'Saving...' : drawerMode === 'create' ? 'Create Trigger' : 'Save Changes'}
                        </button>
                    </div>
                </form>
            </Drawer>
        </div>
    )
}

export default ProjectWebhooksPanel
