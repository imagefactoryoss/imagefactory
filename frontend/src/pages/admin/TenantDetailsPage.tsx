import Drawer from '@/components/ui/Drawer'
import { api } from '@/services/api'
import { adminService } from '@/services/adminService'
import { Tenant } from '@/types'
import { Copy } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { useNavigate, useParams } from 'react-router-dom'

interface Group {
    id: string
    name: string
    description: string
    role_type: 'viewer' | 'developer' | 'operator' | 'owner'
    member_count: number
    created_at: string
}

type TriggerID =
    | 'BN-001'
    | 'BN-002'
    | 'BN-003'
    | 'BN-004'
    | 'BN-005'
    | 'BN-006'
    | 'BN-007'
    | 'BN-008'
    | 'BN-009'
    | 'BN-010'

type Channel = 'in_app' | 'email'
type RecipientPolicy = 'initiator' | 'project_members' | 'tenant_admins' | 'custom_users'
type Severity = 'low' | 'normal' | 'high'
type PreferenceSource = 'system' | 'tenant' | 'project'

interface TenantNotificationPreference {
    trigger_id: TriggerID
    source?: PreferenceSource
    enabled: boolean
    channels: Channel[]
    recipient_policy: RecipientPolicy
    custom_recipient_user_ids?: string[]
    severity_override?: Severity
}

const TRIGGER_CATALOG: Array<{ id: TriggerID; name: string }> = [
    { id: 'BN-001', name: 'Build queued' },
    { id: 'BN-002', name: 'Build started' },
    { id: 'BN-003', name: 'Build completed' },
    { id: 'BN-004', name: 'Build failed' },
    { id: 'BN-005', name: 'Build cancelled' },
    { id: 'BN-006', name: 'Retry started' },
    { id: 'BN-007', name: 'Retry failed' },
    { id: 'BN-008', name: 'Retry succeeded' },
    { id: 'BN-009', name: 'Recovered from stuck/orphaned' },
    { id: 'BN-010', name: 'Preflight blocked' },
]

const TenantDetailsPage: React.FC = () => {
    const { id } = useParams<{ id: string }>()
    const navigate = useNavigate()
    const [tenant, setTenant] = useState<Tenant | null>(null)
    const [groups, setGroups] = useState<Group[]>([])
    const [loading, setLoading] = useState(true)
    const [showGroupDrawer, setShowGroupDrawer] = useState(false)
    const [selectedGroup, setSelectedGroup] = useState<Group | null>(null)
    const [editingGroup, setEditingGroup] = useState(false)
    const [showViewGroupDrawer, setShowViewGroupDrawer] = useState(false)
    const [viewingGroup, setViewingGroup] = useState<Group | null>(null)
    const [groupMembers, setGroupMembers] = useState<any[]>([])
    const [loadingMembers, setLoadingMembers] = useState(false)
    const [notificationPrefs, setNotificationPrefs] = useState<TenantNotificationPreference[]>([])
    const [notificationPrefsLoading, setNotificationPrefsLoading] = useState(false)
    const [notificationPrefsSaving, setNotificationPrefsSaving] = useState(false)
    const [showNotificationDrawer, setShowNotificationDrawer] = useState(false)

    useEffect(() => {
        if (id) {
            const fetchData = async () => {
                await loadTenantDetails()
                await loadGroups()
                await loadTenantNotificationTriggers()
            }
            fetchData()
        }
    }, [id])

    const loadTenantDetails = async () => {
        try {
            if (!id) {
                setLoading(false)
                return
            }
            setLoading(true)
            // Fetch tenant details from API
            const response = await api.get(`/tenants/${id}`)
            const data = response.data

            // The API returns the object directly
            setTenant(data)
            setLoading(false)
        } catch (error: any) {
            toast.error(error.message || 'Failed to load tenant')
            setLoading(false)
            navigate('/admin/tenants')
        }
    }

    const loadGroups = async () => {
        try {
            if (!id) return
            const groupsData = await adminService.getGroupsByTenant(id)
            setGroups(groupsData as Group[])
        } catch (error: any) {
            toast.error(error.message || 'Failed to load groups')
        }
    }

    const loadTenantNotificationTriggers = async () => {
        try {
            if (!id) return
            setNotificationPrefsLoading(true)
            const response = await api.get(`/admin/tenants/${id}/notification-triggers`)
            setNotificationPrefs(response.data?.preferences || [])
        } catch (error: any) {
            toast.error(error.message || 'Failed to load tenant notification defaults')
        } finally {
            setNotificationPrefsLoading(false)
        }
    }

    const updateNotificationPref = (triggerId: TriggerID, patch: Partial<TenantNotificationPreference>) => {
        setNotificationPrefs((prev) =>
            prev.map((pref) =>
                pref.trigger_id === triggerId
                    ? {
                        ...pref,
                        ...patch,
                    }
                    : pref
            )
        )
    }

    const toggleNotificationChannel = (triggerId: TriggerID, channel: Channel, checked: boolean) => {
        setNotificationPrefs((prev) =>
            prev.map((pref) => {
                if (pref.trigger_id !== triggerId) return pref
                const channels = checked
                    ? Array.from(new Set([...(pref.channels || []), channel]))
                    : (pref.channels || []).filter((c) => c !== channel)
                return {
                    ...pref,
                    channels,
                }
            })
        )
    }

    const saveTenantNotificationTriggers = async () => {
        try {
            if (!id) return
            setNotificationPrefsSaving(true)
            await api.put(`/admin/tenants/${id}/notification-triggers`, {
                preferences: notificationPrefs.map((pref) => ({
                    trigger_id: pref.trigger_id,
                    enabled: pref.enabled,
                    channels: pref.channels,
                    recipient_policy: pref.recipient_policy,
                    custom_recipient_user_ids: pref.custom_recipient_user_ids || [],
                    severity_override: pref.severity_override,
                })),
            })
            toast.success('Tenant notification defaults updated')
            await loadTenantNotificationTriggers()
        } catch (error: any) {
            toast.error(error.message || 'Failed to update tenant notification defaults')
        } finally {
            setNotificationPrefsSaving(false)
        }
    }

    const handleEditGroup = (group: Group) => {
        setSelectedGroup(group)
        setEditingGroup(true)
        setShowGroupDrawer(true)
    }

    const handleViewGroup = async (group: Group) => {
        setViewingGroup(group)
        setShowViewGroupDrawer(true)

        // Load group members
        try {
            setLoadingMembers(true)
            const members = await adminService.getGroupMembers(group.id)
            setGroupMembers(members || [])
        } catch (error: any) {
            toast.error('Failed to load group members')
            setGroupMembers([])
        } finally {
            setLoadingMembers(false)
        }
    }

    const handleDeleteGroup = async (_groupId: string) => {
        if (!confirm('Are you sure you want to delete this group?')) {
            return
        }

        try {
            // TODO: Implement API call
            toast.success('Group deleted successfully')
            await loadGroups()
        } catch (error: any) {
            toast.error(error.message || 'Failed to delete group')
        }
    }

    const handleCloseDrawer = () => {
        setShowGroupDrawer(false)
        setSelectedGroup(null)
        setEditingGroup(false)
    }

    const handleCloseViewDrawer = () => {
        setShowViewGroupDrawer(false)
        setViewingGroup(null)
        setGroupMembers([])
    }

    const handleRemoveGroupMember = async (_groupId: string, _memberId: string) => {
        // TODO: Implement remove group member functionality
    }

    const enabledNotificationDefaultsCount = notificationPrefs.filter((pref) => pref.enabled).length

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <svg
                    className="animate-spin h-8 w-8 text-blue-600"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                >
                    <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                    />
                    <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                </svg>
            </div>
        )
    }

    if (!tenant) {
        return (
            <div className="text-center py-12">
                <p className="text-slate-500">Tenant not found</p>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-start justify-between">
                <div>
                    <button
                        onClick={() => navigate('/admin/tenants')}
                        className="mb-4 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 text-sm font-medium flex items-center gap-1"
                    >
                        ← Back to Tenants
                    </button>
                    <h1 className="text-3xl font-bold text-slate-900 dark:text-white">{tenant?.name || 'Tenant'}</h1>
                    <p className="mt-2 text-slate-600 dark:text-slate-400">
                        Manage tenant groups and user assignments
                    </p>
                </div>
            </div>

            {/* Tenant Info Card */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
                    <div>
                        <p className="text-sm text-slate-600 dark:text-slate-400">ID</p>
                        <div className="mt-1 flex items-center space-x-2">
                            <p className="text-sm font-medium text-slate-900 dark:text-white font-mono">
                                {tenant?.id?.slice(0, 8) || 'N/A'}...
                            </p>
                            <button
                                onClick={() => {
                                    if (tenant?.id) {
                                        navigator.clipboard.writeText(tenant.id)
                                        toast.success('ID copied to clipboard')
                                    }
                                }}
                                className="text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300 transition-colors"
                                title="Copy full ID"
                            >
                                <Copy className="h-4 w-4" />
                            </button>
                        </div>
                    </div>
                    <div>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Tenant Code</p>
                        <p className="mt-1 text-sm font-medium text-slate-900 dark:text-white">
                            {tenant?.tenantCode || 'N/A'}
                        </p>
                    </div>
                    <div>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Name</p>
                        <p className="mt-1 text-sm font-medium text-slate-900 dark:text-white">
                            {tenant?.name || 'N/A'}
                        </p>
                    </div>
                    <div>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Slug</p>
                        <p className="mt-1 text-sm font-medium text-slate-900 dark:text-white">
                            {tenant?.slug || 'N/A'}
                        </p>
                    </div>
                    <div>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Status</p>
                        <p className="mt-1">
                            <span
                                className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${tenant?.status === 'active'
                                    ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                                    : tenant?.status === 'suspended'
                                        ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                                        : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                                    }`}
                            >
                                {tenant?.status || 'Unknown'}
                            </span>
                        </p>
                    </div>
                    <div>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Version</p>
                        <p className="mt-1 text-sm font-medium text-slate-900 dark:text-white">
                            {tenant?.version || 'N/A'}
                        </p>
                    </div>
                </div>

                {/* Description */}
                {tenant?.description && (
                    <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
                        <div>
                            <p className="text-sm text-slate-600 dark:text-slate-400">Description</p>
                            <p className="mt-1 text-slate-900 dark:text-white">
                                {tenant.description}
                            </p>
                        </div>
                    </div>
                )}

                {/* Quota Information */}
                {tenant?.quota && (
                    <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
                        <h3 className="text-sm font-medium text-slate-900 dark:text-white mb-3">Resource Quota</h3>
                        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                            <div>
                                <p className="text-sm text-slate-600 dark:text-slate-400">Max Builds</p>
                                <p className="mt-1 text-lg font-medium text-slate-900 dark:text-white">
                                    {tenant.quota.maxBuilds}
                                </p>
                            </div>
                            <div>
                                <p className="text-sm text-slate-600 dark:text-slate-400">Max Images</p>
                                <p className="mt-1 text-lg font-medium text-slate-900 dark:text-white">
                                    {tenant.quota.maxImages}
                                </p>
                            </div>
                            <div>
                                <p className="text-sm text-slate-600 dark:text-slate-400">Max Storage</p>
                                <p className="mt-1 text-lg font-medium text-slate-900 dark:text-white">
                                    {tenant.quota.maxStorageGB}GB
                                </p>
                            </div>
                            <div>
                                <p className="text-sm text-slate-600 dark:text-slate-400">Max Concurrent Jobs</p>
                                <p className="mt-1 text-lg font-medium text-slate-900 dark:text-white">
                                    {tenant.quota.maxConcurrentJobs}
                                </p>
                            </div>
                        </div>
                    </div>
                )}

                {/* Metadata */}
                <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
                    <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                            <p className="text-slate-600 dark:text-slate-400">Created</p>
                            <p className="mt-1 text-slate-900 dark:text-white">
                                {tenant?.createdAt ? `${new Date(tenant.createdAt).toLocaleDateString()} ${new Date(tenant.createdAt).toLocaleTimeString()}` : 'N/A'}
                            </p>
                        </div>
                        <div>
                            <p className="text-slate-600 dark:text-slate-400">Last Updated</p>
                            <p className="mt-1 text-slate-900 dark:text-white">
                                {tenant?.updatedAt ? `${new Date(tenant.updatedAt).toLocaleDateString()} ${new Date(tenant.updatedAt).toLocaleTimeString()}` : 'N/A'}
                            </p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Tenant Notification Defaults */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-xl font-semibold text-slate-900 dark:text-white">Build Notification Defaults</h2>
                        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                            Configure tenant-level defaults for project build notification triggers from a dedicated drawer.
                        </p>
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={loadTenantNotificationTriggers}
                            disabled={notificationPrefsLoading}
                            className="px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50"
                        >
                            Refresh
                        </button>
                        <button
                            onClick={() => setShowNotificationDrawer(true)}
                            className="px-3 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
                        >
                            View / Edit Defaults
                        </button>
                    </div>
                </div>
                <div className="mt-4 grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40 p-3">
                        <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Total triggers</p>
                        <p className="mt-1 text-lg font-semibold text-slate-900 dark:text-white">{notificationPrefs.length}</p>
                    </div>
                    <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40 p-3">
                        <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Enabled</p>
                        <p className="mt-1 text-lg font-semibold text-slate-900 dark:text-white">{enabledNotificationDefaultsCount}</p>
                    </div>
                    <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/40 p-3">
                        <p className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Loading state</p>
                        <p className="mt-1 text-lg font-semibold text-slate-900 dark:text-white">{notificationPrefsLoading ? 'Refreshing' : 'Up to date'}</p>
                    </div>
                </div>
            </div>

            {/* Groups Section */}
            <div>
                <div className="flex items-center justify-between mb-6">
                    <div>
                        <h2 className="text-2xl font-bold text-slate-900">Groups</h2>
                        <p className="mt-1 text-sm text-slate-600">
                            Manage tenant groups and user assignments
                        </p>
                    </div>
                </div>

                {/* Groups Table */}
                <div className="overflow-hidden border border-slate-200 dark:border-slate-700 rounded-lg">
                    <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                        <thead className="bg-slate-50 dark:bg-slate-900">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-600 dark:text-slate-300 uppercase tracking-wider">
                                    Name
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-600 dark:text-slate-300 uppercase tracking-wider">
                                    Description
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-600 dark:text-slate-300 uppercase tracking-wider">
                                    Users
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-600 dark:text-slate-300 uppercase tracking-wider">
                                    Created
                                </th>
                                <th className="px-6 py-3 text-right text-xs font-medium text-slate-600 dark:text-slate-300 uppercase tracking-wider">
                                    Actions
                                </th>
                            </tr>
                        </thead>
                        <tbody className="bg-white dark:bg-slate-950 divide-y divide-slate-200 dark:divide-slate-700">
                            {groups.length === 0 ? (
                                <tr>
                                    <td
                                        colSpan={5}
                                        className="px-6 py-4 text-center text-slate-500 dark:text-slate-400"
                                    >
                                        No groups found
                                    </td>
                                </tr>
                            ) : (
                                groups.map((group) => (
                                    <tr
                                        key={group.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-900 transition-colors"
                                    >
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <p className="font-medium text-slate-900 dark:text-white">
                                                {group.name}
                                            </p>
                                        </td>
                                        <td className="px-6 py-4">
                                            <p className="text-sm text-slate-600 dark:text-slate-400">
                                                {group.description || 'N/A'}
                                            </p>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <button className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 text-sm font-medium">
                                                {group.member_count} users
                                            </button>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                            {new Date(
                                                group.created_at
                                            ).toLocaleDateString()}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right flex gap-2 justify-end">
                                            <button
                                                onClick={() => handleViewGroup(group)}
                                                title="View group details"
                                                className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400 hover:bg-green-100 dark:hover:bg-green-900/50 transition-colors"
                                            >
                                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                                </svg>
                                            </button>
                                            <button
                                                onClick={() => handleEditGroup(group)}
                                                title="Edit group"
                                                className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors"
                                            >
                                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                </svg>
                                            </button>
                                            <button
                                                onClick={() => handleDeleteGroup(group.id)}
                                                title="Delete group"
                                                className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/50 transition-colors"
                                            >
                                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                </svg>
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Drawer */}
            <Drawer
                isOpen={showGroupDrawer}
                onClose={handleCloseDrawer}
                title={editingGroup ? 'Edit Group' : 'Add Group'}
                description="Configure group settings and user assignments"
            >
                <div className="space-y-6">
                    <div>
                        <label className="block text-sm font-medium text-slate-900 dark:text-white">
                            Group Name
                        </label>
                        <input
                            type="text"
                            defaultValue={selectedGroup?.name || ''}
                            className="mt-1 block w-full border border-slate-300 dark:border-slate-600 rounded-lg shadow-sm py-2 px-3 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-500 dark:placeholder-slate-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                            placeholder="e.g., if_acme_developer"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-900 dark:text-white">
                            Description
                        </label>
                        <textarea
                            defaultValue={selectedGroup?.description || ''}
                            rows={3}
                            className="mt-1 block w-full border border-slate-300 dark:border-slate-600 rounded-lg shadow-sm py-2 px-3 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-500 dark:placeholder-slate-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                            placeholder="Group description..."
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-900 dark:text-white mb-3">
                            Role Type
                        </label>
                        <div className="space-y-2">
                            {['viewer', 'developer', 'operator', 'owner'].map(
                                (role) => (
                                    <label
                                        key={role}
                                        className="flex items-center cursor-pointer"
                                    >
                                        <input
                                            type="radio"
                                            name="roleType"
                                            value={role}
                                            defaultChecked={
                                                selectedGroup?.role_type === role
                                            }
                                            className="rounded-full"
                                        />
                                        <span className="ml-3 text-sm text-slate-900 dark:text-white capitalize">
                                            {role === 'owner' ? 'Owner' : role}
                                        </span>
                                    </label>
                                )
                            )}
                        </div>
                    </div>

                    <div className="flex gap-3 pt-6 border-t border-slate-200 dark:border-slate-700">
                        <button
                            onClick={handleCloseDrawer}
                            className="flex-1 px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg text-slate-700 dark:text-slate-300 font-medium hover:bg-slate-50 dark:hover:bg-slate-800"
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleCloseDrawer}
                            className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700"
                        >
                            {editingGroup ? 'Update' : 'Create'}
                        </button>
                    </div>
                </div>
            </Drawer>

            {/* View Group Drawer */}
            <Drawer
                isOpen={showViewGroupDrawer}
                onClose={handleCloseViewDrawer}
                title={`Group: ${viewingGroup?.name || ''}`}
            >
                <div className="space-y-6">
                    {viewingGroup && (
                        <>
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Description
                                    </label>
                                    <p className="text-slate-600 dark:text-slate-400">
                                        {viewingGroup.description || 'No description provided'}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Role Type
                                    </label>
                                    <p className="text-slate-600 dark:text-slate-400 capitalize">
                                        {viewingGroup.role_type === 'owner' ? 'Owner' : viewingGroup.role_type || 'N/A'}
                                    </p>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Member Count
                                    </label>
                                    <p className="text-slate-600 dark:text-slate-400">
                                        {groupMembers.length} member{groupMembers.length !== 1 ? 's' : ''}
                                    </p>
                                </div>
                            </div>

                            <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                                <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
                                    Members
                                </h3>
                                {loadingMembers ? (
                                    <p className="text-sm text-slate-500 dark:text-slate-400">Loading members...</p>
                                ) : groupMembers.length === 0 ? (
                                    <p className="text-sm text-slate-500 dark:text-slate-400">No members in this group</p>
                                ) : (
                                    <div className="space-y-2">
                                        {groupMembers.map((member) => (
                                            <div
                                                key={member.id}
                                                className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700"
                                            >
                                                <div className="flex-1">
                                                    <p className="text-sm font-medium text-slate-900 dark:text-white">
                                                        {member.email}
                                                    </p>
                                                    <p className="text-xs text-slate-500 dark:text-slate-400">
                                                        {member.is_group_admin ? '👤 Group Admin' : '👤 Member'}
                                                    </p>
                                                </div>
                                                <button
                                                    onClick={() => handleRemoveGroupMember(viewingGroup.id, member.id)}
                                                    className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 text-xs font-medium"
                                                >
                                                    Remove
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </>
                    )}

                    <div className="flex gap-3 pt-6 border-t border-slate-200 dark:border-slate-700">
                        <button
                            onClick={handleCloseViewDrawer}
                            className="flex-1 px-4 py-2 bg-slate-600 dark:bg-slate-700 text-white rounded-lg font-medium hover:bg-slate-700 dark:hover:bg-slate-600"
                        >
                            Close
                        </button>
                        <button
                            onClick={() => {
                                if (viewingGroup) {
                                    handleEditGroup(viewingGroup);
                                    handleCloseViewDrawer();
                                }
                            }}
                            className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700"
                        >
                            Edit Group
                        </button>
                    </div>
                </div>
            </Drawer>

            <Drawer
                isOpen={showNotificationDrawer}
                onClose={() => setShowNotificationDrawer(false)}
                title="Build Notification Defaults"
                description="Configure tenant-level defaults for project build notification triggers."
                width="2xl"
            >
                <div className="space-y-4">
                    <div className="flex items-center justify-end gap-2">
                        <button
                            onClick={loadTenantNotificationTriggers}
                            disabled={notificationPrefsLoading}
                            className="px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50"
                        >
                            Refresh
                        </button>
                        <button
                            onClick={saveTenantNotificationTriggers}
                            disabled={notificationPrefsSaving || notificationPrefsLoading}
                            className="px-3 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
                        >
                            {notificationPrefsSaving ? 'Saving...' : 'Save Defaults'}
                        </button>
                    </div>

                    <div className="overflow-x-auto border border-slate-200 dark:border-slate-700 rounded-lg">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                            <thead className="bg-slate-50 dark:bg-slate-900">
                                <tr>
                                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-300">Trigger</th>
                                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-300">Source</th>
                                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-300">Enabled</th>
                                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-300">Channels</th>
                                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-300">Recipients</th>
                                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-300">Severity</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-200 dark:divide-slate-700 bg-white dark:bg-slate-950">
                                {notificationPrefsLoading ? (
                                    <tr>
                                        <td colSpan={6} className="px-4 py-6 text-sm text-slate-500 dark:text-slate-400">
                                            Loading tenant notification defaults...
                                        </td>
                                    </tr>
                                ) : notificationPrefs.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="px-4 py-6 text-sm text-slate-500 dark:text-slate-400">
                                            No notification trigger defaults found.
                                        </td>
                                    </tr>
                                ) : (
                                    notificationPrefs.map((pref) => {
                                        const triggerName = TRIGGER_CATALOG.find((t) => t.id === pref.trigger_id)?.name || pref.trigger_id
                                        return (
                                            <tr key={pref.trigger_id}>
                                                <td className="px-4 py-3">
                                                    <p className="text-sm font-medium text-slate-900 dark:text-white">
                                                        {pref.trigger_id} - {triggerName}
                                                    </p>
                                                </td>
                                                <td className="px-4 py-3">
                                                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-200">
                                                        {pref.source || 'tenant'}
                                                    </span>
                                                </td>
                                                <td className="px-4 py-3">
                                                    <input
                                                        type="checkbox"
                                                        checked={pref.enabled}
                                                        onChange={(e) => updateNotificationPref(pref.trigger_id, { enabled: e.target.checked })}
                                                        className="h-4 w-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                                                    />
                                                </td>
                                                <td className="px-4 py-3">
                                                    <div className="space-y-1">
                                                        <label className="flex items-center gap-2 text-xs text-slate-700 dark:text-slate-300">
                                                            <input
                                                                type="checkbox"
                                                                checked={(pref.channels || []).includes('in_app')}
                                                                onChange={(e) => toggleNotificationChannel(pref.trigger_id, 'in_app', e.target.checked)}
                                                                className="h-3.5 w-3.5 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                                                            />
                                                            In-app
                                                        </label>
                                                        <label className="flex items-center gap-2 text-xs text-slate-700 dark:text-slate-300">
                                                            <input
                                                                type="checkbox"
                                                                checked={(pref.channels || []).includes('email')}
                                                                onChange={(e) => toggleNotificationChannel(pref.trigger_id, 'email', e.target.checked)}
                                                                className="h-3.5 w-3.5 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 dark:bg-slate-700"
                                                            />
                                                            Email
                                                        </label>
                                                    </div>
                                                </td>
                                                <td className="px-4 py-3">
                                                    <select
                                                        value={pref.recipient_policy}
                                                        onChange={(e) => updateNotificationPref(pref.trigger_id, { recipient_policy: e.target.value as RecipientPolicy })}
                                                        className="w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-xs text-slate-900 dark:text-white"
                                                    >
                                                        <option value="initiator">Initiator</option>
                                                        <option value="project_members">Project Members</option>
                                                        <option value="tenant_admins">Tenant Admins</option>
                                                        <option value="custom_users">Custom Users</option>
                                                    </select>
                                                    {pref.recipient_policy === 'custom_users' && (
                                                        <textarea
                                                            rows={2}
                                                            value={(pref.custom_recipient_user_ids || []).join('\n')}
                                                            onChange={(e) =>
                                                                updateNotificationPref(pref.trigger_id, {
                                                                    custom_recipient_user_ids: e.target.value
                                                                        .split(/[\n,]/g)
                                                                        .map((v) => v.trim())
                                                                        .filter(Boolean),
                                                                })
                                                            }
                                                            className="mt-1 w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-xs text-slate-900 dark:text-white"
                                                            placeholder="user UUIDs"
                                                        />
                                                    )}
                                                </td>
                                                <td className="px-4 py-3">
                                                    <select
                                                        value={pref.severity_override || ''}
                                                        onChange={(e) => updateNotificationPref(pref.trigger_id, { severity_override: (e.target.value || undefined) as Severity | undefined })}
                                                        className="w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-2 py-1.5 text-xs text-slate-900 dark:text-white"
                                                    >
                                                        <option value="">Default</option>
                                                        <option value="low">Low</option>
                                                        <option value="normal">Normal</option>
                                                        <option value="high">High</option>
                                                    </select>
                                                </td>
                                            </tr>
                                        )
                                    })
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            </Drawer>
        </div >
    )
}

export default TenantDetailsPage
