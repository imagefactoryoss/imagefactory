import Drawer from '@/components/ui/Drawer'
import { useCanManageAdmin } from '@/hooks/useAccess'
import { adminService } from '@/services/adminService'
import { operationCapabilityService } from '@/services/operationCapabilityService'
import type { OperationCapabilitiesConfig, Tenant, TenantStatus } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

type CapabilityCategory = 'Build' | 'Quarantine' | 'Image Scanning'

const capabilityMetaByKey: Partial<Record<keyof OperationCapabilitiesConfig, { label: string; description: string; category: CapabilityCategory }>> = {
    build: { label: 'Image Build', description: 'Create and manage build workflows.', category: 'Build' },
    quarantine_request: { label: 'Quarantine Request', description: 'Tenant trigger for quarantine request and import/scan pipeline.', category: 'Quarantine' },
    quarantine_release: { label: 'Quarantine Release (Admin)', description: 'Admin-governed release action for eligible quarantined images.', category: 'Quarantine' },
    ondemand_image_scanning: { label: 'On-Demand Image Scanning', description: 'Trigger manual vulnerability scans.', category: 'Image Scanning' },
}

const DEFAULT_REASON = 'Operational capability update'
const MAX_BULK_SELECTION = 50

const OperationalCapabilitiesPage: React.FC = () => {
    const canManageAdmin = useCanManageAdmin()
    const [tenants, setTenants] = useState<Tenant[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    const [search, setSearch] = useState('')
    const [statusFilter, setStatusFilter] = useState<'all' | TenantStatus>('all')
    const [page, setPage] = useState(1)
    const [limit] = useState(20)
    const [pagination, setPagination] = useState({ page: 1, limit: 20, total: 0, totalPages: 0 })
    const [capabilityKeys, setCapabilityKeys] = useState<Array<keyof OperationCapabilitiesConfig>>([])
    const [selectedTenantIds, setSelectedTenantIds] = useState<Set<string>>(new Set())
    const [bulkCapabilityKey, setBulkCapabilityKey] = useState<keyof OperationCapabilitiesConfig>('quarantine_request')
    const [bulkCapabilityValue, setBulkCapabilityValue] = useState<boolean>(true)
    const [bulkChangeReason, setBulkChangeReason] = useState('Bulk operational capability update')
    const [bulkReasonError, setBulkReasonError] = useState<string | null>(null)
    const [bulkSaving, setBulkSaving] = useState(false)

    const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null)
    const [drawerOpen, setDrawerOpen] = useState(false)
    const [capabilities, setCapabilities] = useState<OperationCapabilitiesConfig>(operationCapabilityService.defaultConfig())
    const [initialCapabilities, setInitialCapabilities] = useState<OperationCapabilitiesConfig>(operationCapabilityService.defaultConfig())
    const [changeReason, setChangeReason] = useState(DEFAULT_REASON)
    const [capabilitiesLoading, setCapabilitiesLoading] = useState(false)
    const [capabilityError, setCapabilityError] = useState<string | null>(null)
    const [reasonError, setReasonError] = useState<string | null>(null)
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        let active = true
        const loadCapabilityKeys = async () => {
            try {
                const cfg = await operationCapabilityService.getAdminCapabilities({ globalDefault: true })
                if (!active) {
                    return
                }
                setCapabilityKeys(Object.keys(cfg) as Array<keyof OperationCapabilitiesConfig>)
            } catch {
                if (active) {
                    setCapabilityKeys(Object.keys(operationCapabilityService.defaultConfig()) as Array<keyof OperationCapabilitiesConfig>)
                }
            }
        }
        void loadCapabilityKeys()
        return () => {
            active = false
        }
    }, [])

    useEffect(() => {
        let active = true
        const loadTenants = async () => {
            try {
                setLoading(true)
                setError(null)
                const response = await adminService.getTenants({
                    page,
                    limit,
                    search: search.trim() || undefined,
                    status: statusFilter === 'all' ? undefined : [statusFilter],
                })
                if (!active) {
                    return
                }
                setTenants(response.data || [])
                setPagination(response.pagination || { page, limit, total: 0, totalPages: 0 })
            } catch (err) {
                if (active) {
                    setError(err instanceof Error ? err.message : 'Failed to load tenants')
                    setTenants([])
                    setPagination({ page, limit, total: 0, totalPages: 0 })
                }
            } finally {
                if (active) {
                    setLoading(false)
                }
            }
        }

        void loadTenants()
        return () => {
            active = false
        }
    }, [page, limit, search, statusFilter])

    const capabilitiesChanged = useMemo(
        () => JSON.stringify(capabilities) !== JSON.stringify(initialCapabilities),
        [capabilities, initialCapabilities]
    )
    const capabilityItems = useMemo(
        () =>
            capabilityKeys.map((key) => ({
                key,
                label: capabilityMetaByKey[key]?.label || key,
                description: capabilityMetaByKey[key]?.description || 'Capability entitlement toggle.',
                category: capabilityMetaByKey[key]?.category || 'Build',
            })),
        [capabilityKeys]
    )
    const capabilitySections = useMemo(() => {
        const order: CapabilityCategory[] = ['Build', 'Quarantine', 'Image Scanning']
        return order
            .map((category) => ({
                category,
                items: capabilityItems.filter((item) => item.category === category),
            }))
            .filter((section) => section.items.length > 0)
    }, [capabilityItems])
    const allCurrentPageSelected = useMemo(
        () => tenants.length > 0 && tenants.every((tenant) => selectedTenantIds.has(tenant.id)),
        [tenants, selectedTenantIds]
    )

    useEffect(() => {
        if (capabilityKeys.length === 0) {
            return
        }
        if (!capabilityKeys.includes(bulkCapabilityKey)) {
            setBulkCapabilityKey(capabilityKeys[0])
        }
    }, [capabilityKeys, bulkCapabilityKey])

    useEffect(() => {
        setSelectedTenantIds((prev) => {
            const pageIds = new Set(tenants.map((tenant) => tenant.id))
            const next = new Set<string>()
            prev.forEach((tenantId) => {
                if (pageIds.has(tenantId)) {
                    next.add(tenantId)
                }
            })
            return next
        })
    }, [tenants])

    const openDrawer = async (tenant: Tenant) => {
        setSelectedTenant(tenant)
        setDrawerOpen(true)
        setCapabilitiesLoading(true)
        setCapabilityError(null)
        setReasonError(null)
        setChangeReason(DEFAULT_REASON)
        try {
            const cfg = await operationCapabilityService.getAdminCapabilities({ tenantId: tenant.id })
            const keysFromConfig = Object.keys(cfg) as Array<keyof OperationCapabilitiesConfig>
            setCapabilityKeys((prev) => (prev.length === 0 ? keysFromConfig : prev))
            setCapabilities(cfg)
            setInitialCapabilities(cfg)
        } catch (err) {
            const fallback = operationCapabilityService.defaultConfig()
            setCapabilities(fallback)
            setInitialCapabilities(fallback)
            setCapabilityError(err instanceof Error ? err.message : 'Failed to load tenant capabilities')
        } finally {
            setCapabilitiesLoading(false)
        }
    }

    const closeDrawer = () => {
        if (saving) {
            return
        }
        setDrawerOpen(false)
        setSelectedTenant(null)
        setCapabilityError(null)
        setReasonError(null)
        setChangeReason(DEFAULT_REASON)
        setCapabilities(operationCapabilityService.defaultConfig())
        setInitialCapabilities(operationCapabilityService.defaultConfig())
    }

    const handleCapabilityToggle = (key: keyof OperationCapabilitiesConfig, value: boolean) => {
        setCapabilities((prev) => ({ ...prev, [key]: value }))
        setReasonError(null)
    }

    const toggleTenantSelection = (tenantId: string) => {
        setSelectedTenantIds((prev) => {
            const next = new Set(prev)
            if (next.has(tenantId)) {
                next.delete(tenantId)
            } else {
                if (next.size >= MAX_BULK_SELECTION) {
                    toast.error(`Bulk updates support up to ${MAX_BULK_SELECTION} tenants per run.`)
                    return prev
                }
                next.add(tenantId)
            }
            return next
        })
        setBulkReasonError(null)
    }

    const toggleSelectAllCurrentPage = () => {
        setSelectedTenantIds((prev) => {
            const next = new Set(prev)
            if (allCurrentPageSelected) {
                tenants.forEach((tenant) => next.delete(tenant.id))
            } else {
                for (const tenant of tenants) {
                    if (next.size >= MAX_BULK_SELECTION) {
                        toast.error(`Bulk updates support up to ${MAX_BULK_SELECTION} tenants per run.`)
                        break
                    }
                    next.add(tenant.id)
                }
            }
            return next
        })
        setBulkReasonError(null)
    }

    const handleBulkApply = async () => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        const targetTenantIds = Array.from(selectedTenantIds)
        if (targetTenantIds.length === 0) {
            return
        }
        if (!bulkChangeReason.trim()) {
            setBulkReasonError('Entitlement change reason is required for bulk updates.')
            return
        }

        try {
            setBulkSaving(true)
            setBulkReasonError(null)

            let updatedCount = 0
            let skippedCount = 0
            const failedTenantNames: string[] = []

            for (const tenantId of targetTenantIds) {
                const tenantName = tenants.find((item) => item.id === tenantId)?.name || tenantId
                try {
                    const current = await operationCapabilityService.getAdminCapabilities({ tenantId })
                    if (Boolean(current[bulkCapabilityKey]) === bulkCapabilityValue) {
                        skippedCount += 1
                        continue
                    }

                    const nextConfig: OperationCapabilitiesConfig = {
                        ...current,
                        [bulkCapabilityKey]: bulkCapabilityValue,
                    }

                    await operationCapabilityService.updateAdminCapabilities(nextConfig, {
                        tenantId,
                        changeReason: bulkChangeReason.trim(),
                    })
                    updatedCount += 1
                } catch {
                    failedTenantNames.push(tenantName)
                }
            }

            if (failedTenantNames.length > 0) {
                toast.error(
                    `Bulk update partial: updated ${updatedCount}, skipped ${skippedCount}, failed ${failedTenantNames.length}.`
                )
            } else {
                toast.success(`Bulk update complete: updated ${updatedCount}, skipped ${skippedCount}.`)
            }

            setSelectedTenantIds(new Set())
            setBulkChangeReason('Bulk operational capability update')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Bulk capability update failed')
        } finally {
            setBulkSaving(false)
        }
    }

    const handleSave = async () => {
        if (!canManageAdmin) {
            toast.error('Read-only mode.')
            return
        }
        if (!selectedTenant) {
            return
        }
        if (capabilitiesChanged && !changeReason.trim()) {
            setReasonError('Entitlement change reason is required when capability assignments change.')
            return
        }
        try {
            setSaving(true)
            setReasonError(null)
            const updated = await operationCapabilityService.updateAdminCapabilities(capabilities, {
                tenantId: selectedTenant.id,
                changeReason: changeReason.trim(),
            })
            setCapabilities(updated)
            setInitialCapabilities(updated)
            toast.success(`Operational capabilities updated for ${selectedTenant.name}`)
            closeDrawer()
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to update operational capabilities')
        } finally {
            setSaving(false)
        }
    }

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-slate-900 dark:text-white">Operational Capabilities</h1>
                <p className="mt-2 text-slate-600 dark:text-slate-400">
                    Manage capability entitlements at tenant scale using the tenant list and edit drawer.
                </p>
            </div>

            {!canManageAdmin && (
                <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: capability entitlement edit actions are hidden for System Administrator Viewer.
                </div>
            )}

            <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                    <div className="md:col-span-2">
                        <label className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">
                            Search Tenants
                        </label>
                        <input
                            value={search}
                            onChange={(event) => {
                                setSearch(event.target.value)
                                setPage(1)
                            }}
                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/25 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
                            placeholder="Search by tenant name or slug"
                        />
                    </div>
                    <div>
                        <label className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">
                            Status
                        </label>
                        <select
                            value={statusFilter}
                            onChange={(event) => {
                                setStatusFilter(event.target.value as 'all' | TenantStatus)
                                setPage(1)
                            }}
                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/25 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
                        >
                            <option value="all">All Statuses</option>
                            <option value="active">Active</option>
                            <option value="pending">Pending</option>
                            <option value="suspended">Suspended</option>
                            <option value="deleted">Deleted</option>
                        </select>
                    </div>
                </div>
            </div>

            <div className="rounded-lg border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <div className="border-b border-slate-200 px-4 py-3 dark:border-slate-700">
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">Tenant Directory</h2>
                    <p className="text-xs text-slate-600 dark:text-slate-400">
                        Select a tenant row to edit capability entitlements in a drawer.
                    </p>
                </div>

                {loading ? (
                    <div className="px-4 py-6 text-sm text-slate-600 dark:text-slate-300">Loading tenants...</div>
                ) : error ? (
                    <div className="px-4 py-6 text-sm text-rose-700 dark:text-rose-300">{error}</div>
                ) : tenants.length === 0 ? (
                    <div className="px-4 py-6 text-sm text-slate-600 dark:text-slate-300">
                        No tenants found for the current filters.
                    </div>
                ) : (
                    <>
                        {canManageAdmin && selectedTenantIds.size > 0 && (
                            <div className="border-b border-slate-200 bg-blue-50/60 px-4 py-3 dark:border-slate-700 dark:bg-blue-950/30">
                                <div className="mb-3 flex items-center justify-between gap-3">
                                    <p className="text-sm font-medium text-slate-900 dark:text-white">
                                        {selectedTenantIds.size} tenant(s) selected
                                    </p>
                                    <p className="text-xs text-slate-600 dark:text-slate-400">
                                        Max {MAX_BULK_SELECTION} per bulk run
                                    </p>
                                    <button
                                        type="button"
                                        onClick={() => setSelectedTenantIds(new Set())}
                                        className="rounded-md border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                    >
                                        Clear Selection
                                    </button>
                                </div>
                                <div className="grid grid-cols-1 gap-3 md:grid-cols-4">
                                    <div>
                                        <label htmlFor="bulk-capability-key" className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                                            Capability
                                        </label>
                                        <select
                                            id="bulk-capability-key"
                                            value={bulkCapabilityKey}
                                            onChange={(event) => setBulkCapabilityKey(event.target.value as keyof OperationCapabilitiesConfig)}
                                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/25 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
                                        >
                                            {capabilitySections.map((section) => (
                                                <optgroup key={section.category} label={section.category}>
                                                    {section.items.map((item) => (
                                                        <option key={item.key} value={item.key}>{item.label}</option>
                                                    ))}
                                                </optgroup>
                                            ))}
                                        </select>
                                    </div>
                                    <div>
                                        <label htmlFor="bulk-capability-action" className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                                            Action
                                        </label>
                                        <select
                                            id="bulk-capability-action"
                                            value={bulkCapabilityValue ? 'enable' : 'disable'}
                                            onChange={(event) => setBulkCapabilityValue(event.target.value === 'enable')}
                                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/25 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
                                        >
                                            <option value="enable">Enable</option>
                                            <option value="disable">Disable</option>
                                        </select>
                                    </div>
                                    <div className="md:col-span-2">
                                        <label htmlFor="bulk-capability-reason" className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-300">
                                            Entitlement Change Reason
                                        </label>
                                        <input
                                            id="bulk-capability-reason"
                                            value={bulkChangeReason}
                                            onChange={(event) => {
                                                setBulkChangeReason(event.target.value)
                                                setBulkReasonError(null)
                                            }}
                                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/25 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
                                            placeholder="Reason for bulk capability changes"
                                        />
                                    </div>
                                </div>
                                {bulkReasonError ? (
                                    <p className="mt-2 text-xs text-rose-700 dark:text-rose-300">{bulkReasonError}</p>
                                ) : null}
                                <div className="mt-3 flex justify-end">
                                    <button
                                        type="button"
                                        onClick={() => void handleBulkApply()}
                                        disabled={bulkSaving}
                                        className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                                    >
                                        {bulkSaving ? 'Applying...' : 'Apply Bulk Update'}
                                    </button>
                                </div>
                            </div>
                        )}

                        <div className="overflow-x-auto">
                            <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                                <thead className="bg-slate-50 dark:bg-slate-800">
                                    <tr>
                                        <th className="w-10 px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                                            {canManageAdmin && (
                                                <input
                                                    type="checkbox"
                                                    checked={allCurrentPageSelected}
                                                    onChange={toggleSelectAllCurrentPage}
                                                    aria-label="Select all tenants on this page"
                                                    className="h-4 w-4 rounded border-slate-300 bg-white dark:border-slate-600 dark:bg-slate-900"
                                                />
                                            )}
                                        </th>
                                        <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                                            Tenant
                                        </th>
                                        <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                                            Slug
                                        </th>
                                        <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                                            Status
                                        </th>
                                        <th className="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                                            Actions
                                        </th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                                    {tenants.map((tenant) => (
                                        <tr key={tenant.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/60">
                                            <td className="px-4 py-3">
                                                {canManageAdmin && (
                                                    <input
                                                        type="checkbox"
                                                        checked={selectedTenantIds.has(tenant.id)}
                                                        onChange={() => toggleTenantSelection(tenant.id)}
                                                        aria-label={`Select ${tenant.name}`}
                                                        className="h-4 w-4 rounded border-slate-300 bg-white dark:border-slate-600 dark:bg-slate-900"
                                                    />
                                                )}
                                            </td>
                                            <td className="px-4 py-3 text-sm font-medium text-slate-900 dark:text-white">{tenant.name}</td>
                                            <td className="px-4 py-3 text-sm text-slate-600 dark:text-slate-300">{tenant.slug}</td>
                                            <td className="px-4 py-3 text-sm">
                                                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                                                    {tenant.status || 'active'}
                                                </span>
                                            </td>
                                            <td className="px-4 py-3 text-right">
                                                {canManageAdmin ? (
                                                    <button
                                                        type="button"
                                                        onClick={() => void openDrawer(tenant)}
                                                        className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-800 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                                    >
                                                        Edit Capabilities
                                                    </button>
                                                ) : (
                                                    <span className="text-xs text-slate-500 dark:text-slate-400">Read-only</span>
                                                )}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>

                        <div className="flex items-center justify-between border-t border-slate-200 px-4 py-3 text-sm dark:border-slate-700">
                            <div className="text-slate-600 dark:text-slate-300">
                                Showing {tenants.length} of {pagination.total} tenants
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    type="button"
                                    onClick={() => setPage((prev) => Math.max(1, prev - 1))}
                                    disabled={pagination.page <= 1}
                                    className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                >
                                    Previous
                                </button>
                                <span className="text-xs text-slate-600 dark:text-slate-300">
                                    Page {pagination.page} of {Math.max(pagination.totalPages || 1, 1)}
                                </span>
                                <button
                                    type="button"
                                    onClick={() => setPage((prev) => prev + 1)}
                                    disabled={pagination.page >= (pagination.totalPages || 1)}
                                    className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                >
                                    Next
                                </button>
                            </div>
                        </div>
                    </>
                )}
            </div>

            <Drawer
                isOpen={drawerOpen}
                onClose={closeDrawer}
                title={selectedTenant ? `Edit Capabilities: ${selectedTenant.name}` : 'Edit Capabilities'}
                description={selectedTenant ? `${selectedTenant.slug} • ${selectedTenant.status}` : undefined}
                width="xl"
            >
                {!selectedTenant ? (
                    <p className="text-sm text-slate-600 dark:text-slate-300">No tenant selected.</p>
                ) : capabilitiesLoading ? (
                    <p className="text-sm text-slate-600 dark:text-slate-300">Loading capability assignments...</p>
                ) : capabilityError ? (
                    <div className="space-y-3">
                        <p className="text-sm text-rose-700 dark:text-rose-300">{capabilityError}</p>
                        <p className="text-xs text-slate-600 dark:text-slate-400">
                            Editing fallback defaults. Save to persist explicit values for this tenant.
                        </p>
                    </div>
                ) : null}

                {selectedTenant && (
                    <div className="space-y-4">
                        <div className="space-y-3 rounded-md border border-slate-300 p-3 dark:border-slate-600">
                            {capabilitySections.map((section) => (
                                <div key={section.category} className="rounded-md border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-800/60">
                                    <h4 className="px-2 pb-1 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                                        {section.category}
                                    </h4>
                                    <div className="space-y-1">
                                        {section.items.map((item) => (
                                            <label key={item.key} className="flex items-start justify-between gap-4 rounded px-2 py-1 text-sm text-slate-700 dark:text-slate-300">
                                                <span>
                                                    <span className="block font-medium text-slate-900 dark:text-white">{item.label}</span>
                                                    <span className="block text-xs text-slate-600 dark:text-slate-400">{item.description}</span>
                                                </span>
                                                <input
                                                    type="checkbox"
                                                    checked={Boolean(capabilities[item.key])}
                                                    onChange={(event) => handleCapabilityToggle(item.key, event.target.checked)}
                                                    disabled={capabilitiesLoading || saving || !canManageAdmin}
                                                    className="mt-1 h-4 w-4 rounded border-slate-300 bg-white dark:border-slate-600 dark:bg-slate-900"
                                                />
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>

                        <div>
                            <label htmlFor="operational-capability-change-reason" className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
                                Entitlement Change Reason
                            </label>
                            <textarea
                                id="operational-capability-change-reason"
                                value={changeReason}
                                onChange={(event) => {
                                    setChangeReason(event.target.value)
                                    setReasonError(null)
                                }}
                                rows={2}
                                disabled={saving || !canManageAdmin}
                                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/25 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
                                placeholder="Reason for capability entitlement changes"
                            />
                            {reasonError ? (
                                <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{reasonError}</p>
                            ) : null}
                        </div>

                        <div className="flex items-center justify-end gap-2 border-t border-slate-200 pt-3 dark:border-slate-700">
                            {canManageAdmin && (
                                <button
                                    type="button"
                                    onClick={() => {
                                        setCapabilities(initialCapabilities)
                                        setReasonError(null)
                                    }}
                                    disabled={!capabilitiesChanged || saving}
                                    className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-800 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                >
                                    Reset
                                </button>
                            )}
                            {canManageAdmin && (
                                <button
                                    type="button"
                                    onClick={handleSave}
                                    disabled={!capabilitiesChanged || saving || capabilitiesLoading}
                                    className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                                >
                                    {saving ? 'Saving...' : 'Save Capabilities'}
                                </button>
                            )}
                        </div>
                    </div>
                )}
            </Drawer>
        </div>
    )
}

export default OperationalCapabilitiesPage
