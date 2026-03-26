import { useRefresh } from '@/context/RefreshContext'
import { useCanManageAdmin } from '@/hooks/useAccess'
import { adminService } from '@/services/adminService'
import { api } from '@/services/api'
import { operationCapabilityService } from '@/services/operationCapabilityService'
import { OperationCapabilitiesConfig, Tenant, TenantManagementFilters } from '@/types'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { useNavigate } from 'react-router-dom'

// Types
interface ExternalTenant {
    id: string
    tenant_id: string
    name: string
    slug: string
    description: string
    contact_email: string
    status: string
    company: string
    critical_app: string
    org: string
    app_strategy: string
    record_type: string
    internal_flag: string
    prod_date: string
    tech_exec_email: string
    lob_primary_email: string
    app_mgr_netid: string
    app_mgr_first_name: string
    app_mgr_last_name: string
    app_mgr_email: string
}

interface TenantSelectionModalProps {
    isOpen: boolean
    onClose: () => void
    onSelect: (tenant: ExternalTenant, formData: any) => Promise<void>
    isLoading?: boolean
}

const defaultOperationCapabilities: OperationCapabilitiesConfig = operationCapabilityService.defaultConfig()

const capabilityLabels: Record<string, string> = {
    build: 'Image Build',
    quarantine_request: 'Quarantine Request',
    quarantine_release: 'Quarantine Release (Admin)',
    ondemand_image_scanning: 'On-Demand Image Scanning',
}

const capabilityLabelForKey = (key: string): string =>
    capabilityLabels[key] || key.replace(/_/g, ' ').replace(/\b\w/g, (value) => value.toUpperCase())

const withCapabilityValue = (
    current: OperationCapabilitiesConfig,
    key: string,
    value: boolean
): OperationCapabilitiesConfig =>
    ({ ...current, [key]: value } as OperationCapabilitiesConfig)

const getCapabilityValue = (current: OperationCapabilitiesConfig, key: string): boolean =>
    Boolean((current as unknown as Record<string, boolean>)[key])

// External Tenant Selection Modal
const TenantSelectionModal: React.FC<TenantSelectionModalProps> = ({ isOpen, onClose, onSelect, isLoading }) => {
    const [searchQuery, setSearchQuery] = useState('')
    const [externalTenants, setExternalTenants] = useState<ExternalTenant[]>([])
    const [filteredTenants, setFilteredTenants] = useState<ExternalTenant[]>([])
    const [selectedTenant, setSelectedTenant] = useState<ExternalTenant | null>(null)
    const [loading, setLoading] = useState(false)
    const [formData, setFormData] = useState({
        adminName: '',
        adminEmail: '',
        apiRateLimit: 1000,
        storageLimit: 100,
        maxUsers: 50,
        operationCapabilities: defaultOperationCapabilities,
        capabilityChangeReason: 'Initial tenant onboarding entitlement assignment',
    })
    const [showForm, setShowForm] = useState(false)
    const [ldapUsers, setLdapUsers] = useState<any[]>([])
    const [showUserDropdown, setShowUserDropdown] = useState(false)
    const [searchTimeout, setSearchTimeout] = useState<NodeJS.Timeout | null>(null)

    // Reset form when modal opens/closes
    useEffect(() => {
        let active = true
        if (isOpen) {
            const resetForm = async () => {
                let capabilityDefaults = defaultOperationCapabilities
                try {
                    const globalCapabilities = await operationCapabilityService.getAdminCapabilities({ globalDefault: true })
                    if (active) {
                        capabilityDefaults = globalCapabilities
                    }
                } catch {
                    // Keep local defaults when backend capability profile cannot be loaded.
                }
                if (!active) {
                    return
                }
                setFormData({
                    adminName: '',
                    adminEmail: '',
                    apiRateLimit: 1000,
                    storageLimit: 100,
                    maxUsers: 50,
                    operationCapabilities: capabilityDefaults,
                    capabilityChangeReason: 'Initial tenant onboarding entitlement assignment',
                })
            }
            void resetForm()
            setSelectedTenant(null)
            setShowForm(false)
            setLdapUsers([])
            setShowUserDropdown(false)
        }
        return () => {
            active = false
        }
    }, [isOpen])

    // Load external tenants when modal opens or search query changes (debounced)
    useEffect(() => {
        if (!isOpen) return

        const timeout = setTimeout(() => {
            loadExternalTenants()
        }, searchQuery ? 400 : 0)

        return () => clearTimeout(timeout)
    }, [isOpen, searchQuery])

    // Keep filtered list in sync with fetched results
    useEffect(() => {
        setFilteredTenants(externalTenants)
    }, [externalTenants])

    const loadExternalTenants = async () => {
        try {
            setLoading(true)
            const params = searchQuery ? `?q=${encodeURIComponent(searchQuery)}` : ''
            const response = await api.get(`/external-tenants${params}`)
            const data = response.data
            setExternalTenants(data.tenants || [])
        } catch (error) {
            toast.error('Failed to load available tenants')
        } finally {
            setLoading(false)
        }
    }

    const handleSelectTenant = (tenant: ExternalTenant) => {
        setSelectedTenant(tenant)
        setShowForm(true)
    }

    const searchLDAPUsers = async (query: string) => {
        if (query.length < 3) {
            setLdapUsers([])
            return
        }

        try {
            const response = await api.post('/auth/ldap/search-users', { query, limit: 10 })
            const data = response.data
            setLdapUsers(data.users || [])
            setShowUserDropdown(true)
        } catch (error) {
            setLdapUsers([])
            setShowUserDropdown(false)
        }
    }

    const handleEmailChange = (value: string) => {
        setFormData({ ...formData, adminEmail: value })

        // Clear previous timeout
        if (searchTimeout) {
            clearTimeout(searchTimeout)
        }

        // Set new timeout for debounced search (only if 3+ characters)
        if (value.length >= 3) {
            const timeout = setTimeout(() => {
                searchLDAPUsers(value)
            }, 300)
            setSearchTimeout(timeout)
        } else {
            setLdapUsers([])
            setShowUserDropdown(false)
        }
    }

    const handleUserSelect = (user: any) => {
        setFormData({
            ...formData,
            adminEmail: user.email,
            adminName: user.full_name || `${user.first_name || ''} ${user.last_name || ''}`.trim()
        })
        setShowUserDropdown(false)
        setLdapUsers([])
    }

    const handleSubmit = async () => {
        if (!selectedTenant) return

        try {
            await onSelect(selectedTenant, formData)
            // Reset form state after successful submission
            setFormData({
                adminName: '',
                adminEmail: '',
                apiRateLimit: 1000,
                storageLimit: 100,
                maxUsers: 50,
                operationCapabilities: defaultOperationCapabilities,
                capabilityChangeReason: 'Initial tenant onboarding entitlement assignment',
            })
            setSearchQuery('')
            setSelectedTenant(null)
            setShowForm(false)
            setLdapUsers([])
            setShowUserDropdown(false)
            onClose()
        } catch (error) {
        }
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center p-4">
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
                {/* Header */}
                <div className="sticky top-0 bg-white dark:bg-slate-800 px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                        {showForm ? 'Configure Tenant Quotas' : 'Select Tenant to Onboard'}
                    </h2>
                    <button
                        onClick={() => {
                            setShowForm(false)
                            setSelectedTenant(null)
                            setFormData({
                                adminName: '',
                                adminEmail: '',
                                apiRateLimit: 1000,
                                storageLimit: 100,
                                maxUsers: 50,
                                operationCapabilities: defaultOperationCapabilities,
                                capabilityChangeReason: 'Initial tenant onboarding entitlement assignment',
                            })
                            setLdapUsers([])
                            setShowUserDropdown(false)
                            onClose()
                        }}
                        className="text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white"
                    >
                        ✕
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-4">
                    {!showForm ? (
                        <>
                            {/* Search Input */}
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                    Search Tenants (by name or ID)
                                </label>
                                <input
                                    type="text"
                                    value={searchQuery}
                                    onChange={(e) => setSearchQuery(e.target.value)}
                                    placeholder="Type to search..."
                                    className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500"
                                    autoFocus
                                />
                            </div>

                            {/* Tenants List */}
                            <div className="border border-slate-300 dark:border-slate-600 rounded-md max-h-96 overflow-y-auto">
                                {loading ? (
                                    <div className="flex justify-center items-center py-8">
                                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                                    </div>
                                ) : filteredTenants.length === 0 ? (
                                    <div className="px-4 py-8 text-center text-slate-600 dark:text-slate-400">
                                        {searchQuery ? 'No tenants match your search' : 'No tenants available'}
                                    </div>
                                ) : (
                                    <div className="divide-y divide-slate-200 dark:divide-slate-700">
                                        {filteredTenants.map((tenant) => (
                                            <button
                                                key={tenant.id}
                                                onClick={() => handleSelectTenant(tenant)}
                                                className="w-full px-4 py-3 text-left hover:bg-blue-50 dark:hover:bg-blue-900 transition-colors"
                                            >
                                                <div className="flex items-start justify-between">
                                                    <div>
                                                        <div className="font-medium text-slate-900 dark:text-white">
                                                            {tenant.name}
                                                        </div>
                                                        <div className="text-sm text-slate-600 dark:text-slate-400">
                                                            ID: {tenant.tenant_id} • {tenant.status} • {tenant.company}
                                                        </div>
                                                        <div className="text-sm text-slate-500 dark:text-slate-500 mt-1">
                                                            {tenant.description}
                                                        </div>
                                                    </div>
                                                    <div className="text-2xl">→</div>
                                                </div>
                                            </button>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </>
                    ) : (
                        <>
                            {/* Selected Tenant Summary */}
                            {selectedTenant && (
                                <div className="bg-blue-50 dark:bg-blue-900 border border-blue-200 dark:border-blue-700 rounded-md p-4 mb-4">
                                    <h3 className="font-semibold text-blue-900 dark:text-blue-100 mb-2">
                                        {selectedTenant.name}
                                    </h3>
                                    <div className="text-sm text-blue-800 dark:text-blue-200 space-y-1">
                                        <div>Tenant ID: <span className="font-mono">{selectedTenant.tenant_id}</span></div>
                                        <div>Contact: <span className="font-mono">{selectedTenant.contact_email}</span></div>
                                        <div>Status: {selectedTenant.status}</div>
                                    </div>
                                </div>
                            )}

                            {/* Quota Configuration */}
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Tenant Administrator Name
                                    </label>
                                    <input
                                        type="text"
                                        value={formData.adminName}
                                        onChange={(e) => setFormData({ ...formData, adminName: e.target.value })}
                                        placeholder="e.g., John Doe"
                                        className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500"
                                        required
                                    />
                                </div>

                                <div className="relative">
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Tenant Administrator Email (LDAP Search)
                                    </label>
                                    <input
                                        type="email"
                                        value={formData.adminEmail}
                                        onChange={(e) => handleEmailChange(e.target.value)}
                                        onFocus={() => formData.adminEmail.length >= 3 && setShowUserDropdown(true)}
                                        onBlur={() => setTimeout(() => setShowUserDropdown(false), 200)}
                                        placeholder="Type at least 3 characters to search LDAP users..."
                                        className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500"
                                        required
                                    />
                                    {showUserDropdown && ldapUsers.length > 0 && (
                                        <div className="absolute z-10 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-md shadow-lg max-h-48 overflow-y-auto">
                                            {ldapUsers.map((user) => (
                                                <button
                                                    key={user.email}
                                                    onClick={() => handleUserSelect(user)}
                                                    className="w-full px-4 py-2 text-left hover:bg-blue-50 dark:hover:bg-blue-900 transition-colors border-b border-slate-200 dark:border-slate-700 last:border-b-0"
                                                >
                                                    <div className="font-medium text-slate-900 dark:text-white">
                                                        {user.full_name || `${user.first_name || ''} ${user.last_name || ''}`.trim()}
                                                    </div>
                                                    <div className="text-sm text-slate-600 dark:text-slate-400">
                                                        {user.email}
                                                    </div>
                                                </button>
                                            ))}
                                        </div>
                                    )}
                                    {showUserDropdown && ldapUsers.length === 0 && formData.adminEmail.length >= 3 && (
                                        <div className="absolute z-10 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-md shadow-lg">
                                            <div className="px-4 py-2 text-sm text-slate-500 dark:text-slate-400">
                                                No users found. Enter email manually.
                                            </div>
                                        </div>
                                    )}
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        API Rate Limit (requests/minute)
                                    </label>
                                    <input
                                        type="number"
                                        value={formData.apiRateLimit}
                                        onChange={(e) => setFormData({ ...formData, apiRateLimit: parseInt(e.target.value) })}
                                        className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        min="100"
                                        step="100"
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Storage Limit (GB)
                                    </label>
                                    <input
                                        type="number"
                                        value={formData.storageLimit}
                                        onChange={(e) => setFormData({ ...formData, storageLimit: parseInt(e.target.value) })}
                                        className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        min="10"
                                        step="10"
                                    />
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Max Users
                                    </label>
                                    <input
                                        type="number"
                                        value={formData.maxUsers}
                                        onChange={(e) => setFormData({ ...formData, maxUsers: parseInt(e.target.value) })}
                                        className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        min="1"
                                        step="1"
                                    />
                                </div>

                                <div className="rounded-md border border-slate-300 dark:border-slate-600 p-3 space-y-3">
                                    <div>
                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-white">Capability Entitlements</h4>
                                        <p className="text-xs text-slate-600 dark:text-slate-400">
                                            These determine what the tenant can see and execute immediately after onboarding.
                                        </p>
                                    </div>
                                    {Object.keys(formData.operationCapabilities).sort().map((key) => (
                                        <label key={key} className="flex items-center justify-between text-sm text-slate-700 dark:text-slate-300">
                                            <span>{capabilityLabelForKey(key)}</span>
                                            <input
                                                type="checkbox"
                                                checked={getCapabilityValue(formData.operationCapabilities, key)}
                                                onChange={(e) =>
                                                    setFormData((prev) => ({
                                                        ...prev,
                                                        operationCapabilities: withCapabilityValue(prev.operationCapabilities, key, e.target.checked),
                                                    }))
                                                }
                                                className="h-4 w-4 rounded border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900"
                                            />
                                        </label>
                                    ))}
                                    <div>
                                        <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                                            Entitlement Change Reason
                                        </label>
                                        <textarea
                                            value={formData.capabilityChangeReason}
                                            onChange={(e) => setFormData((prev) => ({ ...prev, capabilityChangeReason: e.target.value }))}
                                            rows={2}
                                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                            placeholder="Why these capabilities are assigned for this tenant"
                                        />
                                    </div>
                                </div>
                            </div>
                        </>
                    )}
                </div>

                {/* Footer */}
                <div className="sticky bottom-0 bg-white dark:bg-slate-800 px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex gap-3">
                    <button
                        onClick={() => {
                            if (showForm) {
                                setShowForm(false)
                                setFormData({
                                    adminName: '',
                                    adminEmail: '',
                                    apiRateLimit: 1000,
                                    storageLimit: 100,
                                    maxUsers: 50,
                                    operationCapabilities: defaultOperationCapabilities,
                                    capabilityChangeReason: 'Initial tenant onboarding entitlement assignment',
                                })
                                setLdapUsers([])
                                setShowUserDropdown(false)
                            } else {
                                onClose()
                            }
                        }}
                        className="flex-1 px-4 py-2 text-slate-900 dark:text-white border border-slate-300 dark:border-slate-600 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700"
                    >
                        {showForm ? 'Back' : 'Cancel'}
                    </button>
                    {showForm && (
                        <button
                            onClick={handleSubmit}
                            disabled={isLoading || !selectedTenant}
                            className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
                        >
                            {isLoading ? 'Creating...' : 'Create Tenant'}
                        </button>
                    )}
                </div>
            </div>
        </div>
    )
}

// Tenant Form Modal Component (for editing)
interface TenantFormModalProps {
    isOpen: boolean
    tenant: Tenant | null
    onClose: () => void
    onSubmit: (data: any) => Promise<void>
    isLoading?: boolean
}

const TenantFormModal: React.FC<TenantFormModalProps> = ({ isOpen, tenant, onClose, onSubmit, isLoading }) => {
    const [formData, setFormData] = useState({
        name: '',
        slug: '',
        description: '',
        status: 'active',
        maxBuilds: 1000,
        storageLimit: 5000,
    })
    const [errors, setErrors] = useState<Record<string, string>>({})
    const [operationCapabilities, setOperationCapabilities] = useState<OperationCapabilitiesConfig>(defaultOperationCapabilities)
    const [initialOperationCapabilities, setInitialOperationCapabilities] = useState<OperationCapabilitiesConfig>(defaultOperationCapabilities)
    const [capabilityChangeReason, setCapabilityChangeReason] = useState('Tenant entitlement update')
    const [capabilitiesLoading, setCapabilitiesLoading] = useState(false)
    const [activeFormTab, setActiveFormTab] = useState<'details' | 'capabilities'>('details')

    useEffect(() => {
        setActiveFormTab('details')
        if (tenant) {
            setFormData({
                name: tenant.name || '',
                slug: tenant.slug || '',
                description: tenant.description || '',
                status: tenant.status || 'active',
                maxBuilds: tenant.quota?.maxBuilds || 1000,
                storageLimit: tenant.quota?.maxStorageGB || 5000,
            })
        } else {
            setFormData({
                name: '',
                slug: '',
                description: '',
                status: 'active',
                maxBuilds: 1000,
                storageLimit: 5000,
            })
        }
        setErrors({})
    }, [tenant, isOpen])

    useEffect(() => {
        if (!isOpen || !tenant?.id) {
            setOperationCapabilities(defaultOperationCapabilities)
            setInitialOperationCapabilities(defaultOperationCapabilities)
            setCapabilityChangeReason('Tenant entitlement update')
            return
        }

        let cancelled = false
        const loadCapabilities = async () => {
            try {
                setCapabilitiesLoading(true)
                const cfg = await operationCapabilityService.getAdminCapabilities({ tenantId: tenant.id })
                if (!cancelled) {
                    setOperationCapabilities(cfg)
                    setInitialOperationCapabilities(cfg)
                    setCapabilityChangeReason('Tenant entitlement update')
                }
            } catch {
                if (!cancelled) {
                    setOperationCapabilities(defaultOperationCapabilities)
                    setInitialOperationCapabilities(defaultOperationCapabilities)
                }
            } finally {
                if (!cancelled) {
                    setCapabilitiesLoading(false)
                }
            }
        }

        void loadCapabilities()
        return () => {
            cancelled = true
        }
    }, [isOpen, tenant?.id])

    const validateForm = () => {
        const newErrors: Record<string, string> = {}

        if (!formData.name) newErrors.name = 'Tenant group name is required'
        if (!formData.slug) newErrors.slug = 'Slug is required'
        if (!/^[a-z0-9-]+$/.test(formData.slug)) newErrors.slug = 'Slug must contain only lowercase letters, numbers, and hyphens'
        if (formData.maxBuilds < 1) newErrors.maxBuilds = 'Max builds must be at least 1'
        if (formData.storageLimit < 1) newErrors.storageLimit = 'Storage limit must be at least 1 GB'
        const capabilitiesChanged = JSON.stringify(operationCapabilities) !== JSON.stringify(initialOperationCapabilities)
        if (capabilitiesChanged && !capabilityChangeReason.trim()) {
            newErrors.capabilityChangeReason = 'Entitlement change reason is required when capability assignments change'
            setActiveFormTab('capabilities')
        }

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!validateForm()) return

        try {
            await onSubmit({
                name: formData.name,
                slug: formData.slug,
                description: formData.description,
                status: formData.status,
                max_builds: formData.maxBuilds,
                storage_limit_gb: formData.storageLimit,
                operation_capabilities: operationCapabilities,
                capability_change_reason: capabilityChangeReason.trim(),
            })
            onClose()
        } catch (error) {
        }
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50">
            <button
                type="button"
                aria-label="Close tenant editor"
                onClick={onClose}
                className="absolute inset-0 bg-black/50"
            />
            <div className="absolute inset-y-0 right-0 w-full max-w-2xl">
                <div className="h-full bg-white dark:bg-slate-800 shadow-xl flex flex-col border-l border-slate-200 dark:border-slate-700">
                    {/* Header */}
                    <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between shrink-0">
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                            {tenant ? 'Edit Tenant' : 'Create Tenant'}
                        </h2>
                        <button
                            onClick={onClose}
                            className="text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white"
                        >
                            ✕
                        </button>
                    </div>

                    {/* Form */}
                    <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto p-6 space-y-4">
                        <div className="inline-flex rounded-md border border-slate-200 bg-slate-100 p-1 dark:border-slate-700 dark:bg-slate-900">
                            <button
                                type="button"
                                onClick={() => setActiveFormTab('details')}
                                className={`rounded px-3 py-1.5 text-xs font-medium transition-colors ${activeFormTab === 'details'
                                    ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white'
                                    : 'text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'}`}
                            >
                                Details
                            </button>
                            <button
                                type="button"
                                onClick={() => setActiveFormTab('capabilities')}
                                className={`rounded px-3 py-1.5 text-xs font-medium transition-colors ${activeFormTab === 'capabilities'
                                    ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white'
                                    : 'text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'}`}
                            >
                                Operation Capabilities
                            </button>
                        </div>

                        {activeFormTab === 'details' ? (
                            <>
                                {/* Tenant Group Name */}
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                        Tenant group name *
                                    </label>
                                    <input
                                        type="text"
                                        value={formData.name}
                                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        placeholder="e.g., Finance Team"
                                    />
                                    {errors.name && <p className="text-xs text-red-600 mt-1">{errors.name}</p>}
                                </div>

                                {/* Slug */}
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                        Slug *
                                    </label>
                                    <input
                                        type="text"
                                        value={formData.slug}
                                        onChange={(e) => setFormData({ ...formData, slug: e.target.value.toLowerCase() })}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        placeholder="your-company"
                                    />
                                    {errors.slug && <p className="text-xs text-red-600 mt-1">{errors.slug}</p>}
                                </div>

                                {/* Description */}
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                        Description
                                    </label>
                                    <textarea
                                        value={formData.description}
                                        onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        placeholder="Brief description of the tenant"
                                        rows={2}
                                    />
                                </div>

                                {/* Status */}
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                        Status
                                    </label>
                                    <select
                                        value={formData.status}
                                        onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                    >
                                        <option value="active">Active</option>
                                        <option value="suspended">Suspended</option>
                                        <option value="pending">Pending</option>
                                        <option value="archived">Archived</option>
                                    </select>
                                </div>

                                {/* Max Builds */}
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                        Max Builds
                                    </label>
                                    <input
                                        type="number"
                                        value={formData.maxBuilds}
                                        onChange={(e) => setFormData({ ...formData, maxBuilds: parseInt(e.target.value) })}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        min="1"
                                    />
                                    {errors.maxBuilds && <p className="text-xs text-red-600 mt-1">{errors.maxBuilds}</p>}
                                </div>

                                {/* Storage Limit */}
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                        Storage Limit (GB)
                                    </label>
                                    <input
                                        type="number"
                                        value={formData.storageLimit}
                                        onChange={(e) => setFormData({ ...formData, storageLimit: parseInt(e.target.value) })}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        min="1"
                                    />
                                    {errors.storageLimit && <p className="text-xs text-red-600 mt-1">{errors.storageLimit}</p>}
                                </div>
                            </>
                        ) : null}

                        {activeFormTab === 'capabilities' ? (
                            <div className="space-y-4">
                                <div className="rounded-md border border-slate-300 dark:border-slate-600 p-3 space-y-3">
                                    <div>
                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-white">Operation Capability Entitlements</h4>
                                        <p className="text-xs text-slate-600 dark:text-slate-400">
                                            Adjust tenant-visible capabilities for build, quarantine, and scan operations.
                                        </p>
                                    </div>
                                    {capabilitiesLoading ? (
                                        <p className="text-xs text-slate-500 dark:text-slate-400">Loading current capability assignments...</p>
                                    ) : (
                                        <>
                                            {Object.keys(operationCapabilities).sort().map((key) => (
                                                <label key={key} className="flex items-center justify-between text-sm text-slate-700 dark:text-slate-300">
                                                    <span>{capabilityLabelForKey(key)}</span>
                                                    <input
                                                        type="checkbox"
                                                        checked={getCapabilityValue(operationCapabilities, key)}
                                                        onChange={(e) =>
                                                            setOperationCapabilities((prev) => withCapabilityValue(prev, key, e.target.checked))
                                                        }
                                                        className="h-4 w-4 rounded border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900"
                                                    />
                                                </label>
                                            ))}
                                        </>
                                    )}
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                        Entitlement Change Reason
                                    </label>
                                    <textarea
                                        value={capabilityChangeReason}
                                        onChange={(e) => setCapabilityChangeReason(e.target.value)}
                                        rows={2}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                                        placeholder="Reason for capability entitlement changes"
                                    />
                                    {errors.capabilityChangeReason && <p className="text-xs text-red-600 mt-1">{errors.capabilityChangeReason}</p>}
                                </div>
                            </div>
                        ) : null}

                        {/* Buttons */}
                        <div className="flex gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                            <button
                                type="button"
                                onClick={onClose}
                                className="flex-1 px-4 py-2 text-slate-900 dark:text-white border border-slate-300 dark:border-slate-600 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700"
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={isLoading}
                                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
                            >
                                {isLoading ? 'Saving...' : tenant ? 'Update Tenant' : 'Create Tenant'}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    )
}

// Main Page Component
const TenantManagementPage: React.FC = () => {
    const canManageAdmin = useCanManageAdmin()
    const navigate = useNavigate()
    const { registerRefreshCallback, unregisterRefreshCallback } = useRefresh()
    const [tenants, setTenants] = useState<Tenant[]>([])
    const [loading, setLoading] = useState(true)
    const [submitting, setSubmitting] = useState(false)
    const [showTenantModal, setShowTenantModal] = useState(false)
    const [showSelectionModal, setShowSelectionModal] = useState(false)
    const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null)
    const [sortColumn, setSortColumn] = useState<'name' | 'status' | null>('name')
    const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')
    const [filters, setFilters] = useState<TenantManagementFilters>({
        page: 1,
        limit: 20,
    })
    const [pagination, setPagination] = useState({
        page: 1,
        limit: 20,
        total: 0,
        totalPages: 0,
    })

    useEffect(() => {
        loadTenants()
    }, [filters])

    // Register refresh callback
    useEffect(() => {
        const refreshCallback = async () => {
            await loadTenants()
        }

        registerRefreshCallback(refreshCallback)
        return () => unregisterRefreshCallback(refreshCallback)
    }, [])

    const loadTenants = async () => {
        try {
            setLoading(true)
            const response = await adminService.getTenants(filters)
            setTenants(response?.data || [])
            setPagination(response?.pagination || { page: 1, limit: 20, total: 0, totalPages: 0 })
        } catch (error: any) {
            toast.error(error.message || 'Failed to load tenants')
            setTenants([])
        } finally {
            setLoading(false)
        }
    }

    const handleSort = (column: 'name' | 'status') => {
        if (sortColumn === column) {
            setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
        } else {
            setSortColumn(column)
            setSortOrder('asc')
        }
    }

    const getSortedTenants = () => {
        if (!sortColumn) return tenants

        const sorted = [...tenants].sort((a, b) => {
            let aValue: string = ''
            let bValue: string = ''

            if (sortColumn === 'name') {
                aValue = a.name || ''
                bValue = b.name || ''
            } else if (sortColumn === 'status') {
                aValue = a.status || 'active'
                bValue = b.status || 'active'
            }

            const comparison = aValue.localeCompare(bValue)
            return sortOrder === 'asc' ? comparison : -comparison
        })

        return sorted
    }

    const SortHeader = ({ column, label }: { column: 'name' | 'status', label: string }) => {
        const isActive = sortColumn === column
        return (
            <button
                onClick={() => handleSort(column)}
                className="inline-flex items-center gap-1.5 text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer px-2 py-1 rounded font-medium group"
                type="button"
                title={`Click to sort by ${label}`}
            >
                <span>{label}</span>
                <span className={`text-xs transition-all ${isActive ? 'text-slate-900 dark:text-white font-bold' : 'text-slate-400 dark:text-slate-600 group-hover:text-slate-600 dark:group-hover:text-slate-400'}`}>
                    {isActive ? (sortOrder === 'asc' ? '▲' : '▼') : '⇅'}
                </span>
            </button>
        )
    }

    const handleOnboardNewTenant = async (externalTenant: ExternalTenant, formData: any) => {
        // Validate admin info
        if (!formData.adminName || !formData.adminName.trim()) {
            toast.error('Tenant administrator name is required')
            return
        }
        if (!formData.adminEmail || !formData.adminEmail.trim()) {
            toast.error('Tenant administrator email is required')
            return
        }
        if (!formData.capabilityChangeReason || !formData.capabilityChangeReason.trim()) {
            toast.error('Capability entitlement change reason is required')
            return
        }

        try {
            setSubmitting(true)
            const companyId = (window as any).__companyId || null
            // Auto-populate owner fields from ExternalTenant (APP_MGR fields)
            const adminName = externalTenant.app_mgr_first_name && externalTenant.app_mgr_last_name
                ? `${externalTenant.app_mgr_first_name} ${externalTenant.app_mgr_last_name}`.trim()
                : formData.adminName;
            const adminEmail = externalTenant.app_mgr_email || formData.adminEmail;
            const createResponse = await api.post('/tenants', {
                ...(companyId ? { company_id: companyId } : {}),
                external_tenant_id: externalTenant.id,
                tenant_code: externalTenant.tenant_id,
                name: externalTenant.name,
                slug: externalTenant.slug,
                admin_name: adminName,
                admin_email: adminEmail,
                contact_email: externalTenant.contact_email,
                status: externalTenant.status,
                company: externalTenant.company,
                critical_app: externalTenant.critical_app,
                org: externalTenant.org,
                app_strategy: externalTenant.app_strategy,
                record_type: externalTenant.record_type,
                internal_flag: externalTenant.internal_flag,
                prod_date: externalTenant.prod_date,
                tech_exec_email: externalTenant.tech_exec_email,
                lob_primary_email: externalTenant.lob_primary_email,
                app_mgr_netid: externalTenant.app_mgr_netid,
                app_mgr_first_name: externalTenant.app_mgr_first_name,
                app_mgr_last_name: externalTenant.app_mgr_last_name,
                app_mgr_email: externalTenant.app_mgr_email,
                api_rate_limit: formData.apiRateLimit,
                storage_limit: formData.storageLimit,
                max_users: formData.maxUsers,
            })
            const createdTenantId: string | undefined = createResponse?.data?.id
            if (createdTenantId) {
                await operationCapabilityService.updateAdminCapabilities(
                    formData.operationCapabilities || defaultOperationCapabilities,
                    {
                        tenantId: createdTenantId,
                        changeReason: formData.capabilityChangeReason,
                    }
                )
            }

            toast.success('Tenant onboarded successfully with default groups created')
            setShowSelectionModal(false)
            loadTenants()
        } catch (error: any) {
            toast.error(error.message || 'Failed to onboard tenant')
        } finally {
            setSubmitting(false)
        }
    }

    const handleUpdateTenant = async (data: any) => {
        if (!selectedTenant) return

        try {
            setSubmitting(true)
            await adminService.updateTenant(selectedTenant.id, {
                name: data.name,
                slug: data.slug,
                description: data.description,
                status: data.status,
                quota: {
                    maxBuilds: data.max_builds,
                    maxImages: 500,
                    maxStorageGB: data.storage_limit_gb,
                    maxConcurrentJobs: 5,
                },
            } as any)
            await operationCapabilityService.updateAdminCapabilities(
                data.operation_capabilities || defaultOperationCapabilities,
                {
                    tenantId: selectedTenant.id,
                    changeReason: data.capability_change_reason || 'Tenant entitlement update',
                }
            )
            toast.success('Tenant updated successfully')
            loadTenants()
        } catch (error: any) {
            toast.error(error.message || 'Failed to update tenant')
        } finally {
            setSubmitting(false)
        }
    }

    const handleDeleteTenant = async (tenantId: string) => {
        if (!confirm('Are you sure you want to delete this tenant? This action cannot be undone.')) return

        try {
            await adminService.deleteTenant(tenantId)
            toast.success('Tenant deleted successfully')
            loadTenants()
        } catch (error: any) {
            toast.error(error.message || 'Failed to delete tenant')
        }
    }

    const handleFilterChange = (newFilters: Partial<TenantManagementFilters>) => {
        setFilters((prev) => ({ ...prev, ...newFilters, page: 1 }))
    }

    const getStatusBadgeColor = (status: string) => {
        switch (status) {
            case 'active': return 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
            case 'suspended': return 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200'
            case 'pending': return 'bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200'
            case 'archived': return 'bg-gray-100 dark:bg-gray-900 text-gray-800 dark:text-gray-200'
            default: return 'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200'
        }
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-slate-900 dark:text-white">Tenants</h1>
                    <p className="mt-2 text-slate-600 dark:text-slate-400">
                        Create and manage tenants for your organization
                    </p>
                </div>
                {canManageAdmin && (
                    <button
                        onClick={() => setShowSelectionModal(true)}
                        className="inline-flex items-center justify-center rounded-md border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900"
                    >
                        + Add Tenant
                    </button>
                )}
            </div>

            {!canManageAdmin && (
                <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: tenant create, edit, and delete actions are hidden for System Administrator Viewer.
                </div>
            )}

            {/* Filters */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Search</label>
                    <input
                        type="text"
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        placeholder="Search by name or slug..."
                        value={filters.search || ''}
                        onChange={(e) => handleFilterChange({ search: e.target.value })}
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Status</label>
                    <select
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        value={filters.status?.[0] || ''}
                        onChange={(e) => handleFilterChange({ status: e.target.value ? [e.target.value as any] : undefined })}
                    >
                        <option value="">All Status</option>
                        <option value="active">Active</option>
                        <option value="suspended">Suspended</option>
                        <option value="pending">Pending</option>
                        <option value="archived">Archived</option>
                    </select>
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Items per page</label>
                    <select
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        value={filters.limit || 20}
                        onChange={(e) => handleFilterChange({ limit: parseInt(e.target.value) })}
                    >
                        <option value={10}>10</option>
                        <option value={20}>20</option>
                        <option value={50}>50</option>
                        <option value={100}>100</option>
                    </select>
                </div>
            </div>

            {/* Tenants Table */}
            <div className="bg-white dark:bg-slate-950 rounded-lg shadow-md overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                        <thead className="bg-slate-50 dark:bg-slate-900">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    <SortHeader column="name" label="Tenant" />
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    <SortHeader column="status" label="Status" />
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    Resources
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    Contact
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    Created
                                </th>
                                <th className="px-6 py-3 text-right text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    Actions
                                </th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                            {loading ? (
                                <tr>
                                    <td colSpan={6} className="px-6 py-4 text-center">
                                        <div className="flex justify-center items-center space-x-2">
                                            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                                            <span className="text-slate-600 dark:text-slate-400">Loading tenants...</span>
                                        </div>
                                    </td>
                                </tr>
                            ) : tenants.length === 0 ? (
                                <tr>
                                    <td colSpan={6} className="px-6 py-4 text-center text-slate-600 dark:text-slate-400">
                                        No tenants found. {filters.search && 'Try adjusting your search filters.'}
                                    </td>
                                </tr>
                            ) : (
                                getSortedTenants().map((tenant) => (
                                    <tr
                                        key={tenant.id}
                                        onClick={() => navigate(`/admin/tenants/${tenant.id}`)}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-900 transition-colors cursor-pointer"
                                    >
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div>
                                                <div className="text-sm font-medium text-slate-900 dark:text-white">{tenant.name}</div>
                                                <div className="text-sm text-slate-600 dark:text-slate-400">{tenant.slug}</div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusBadgeColor(tenant.status || 'active')}`}>
                                                {tenant.status || 'active'}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm text-slate-900 dark:text-white">
                                                <div>Builds: {tenant.quota?.maxBuilds || '-'}</div>
                                                <div>Storage: {tenant.quota?.maxStorageGB || '-'} GB</div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-400">
                                            {tenant.contactEmail || '-'}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-400">
                                            {tenant.createdAt ? new Date(tenant.createdAt).toLocaleDateString() : '-'}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <div className="flex justify-end space-x-2">
                                                {canManageAdmin && (
                                                    <button
                                                        onClick={(e) => {
                                                            e.stopPropagation()
                                                            setSelectedTenant(tenant)
                                                            setShowTenantModal(true)
                                                        }}
                                                        className="inline-flex items-center justify-center h-8 w-8 rounded hover:bg-blue-100 dark:hover:bg-blue-900 text-blue-600 dark:text-blue-400"
                                                        title="Edit tenant"
                                                    >
                                                        ✎
                                                    </button>
                                                )}

                                                {canManageAdmin && (
                                                    <button
                                                        onClick={(e) => {
                                                            e.stopPropagation()
                                                            handleDeleteTenant(tenant.id)
                                                        }}
                                                        className="inline-flex items-center justify-center h-8 w-8 rounded hover:bg-red-100 dark:hover:bg-red-900 text-red-600 dark:text-red-400"
                                                        title="Delete tenant"
                                                    >
                                                        ✕
                                                    </button>
                                                )}
                                            </div>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Pagination */}
            {pagination.totalPages > 1 && (
                <div className="flex items-center justify-between">
                    <div className="text-sm text-slate-700 dark:text-slate-300">
                        Showing {((pagination.page - 1) * pagination.limit) + 1} to {Math.min(pagination.page * pagination.limit, pagination.total)} of {pagination.total} tenants
                    </div>
                    <div className="flex space-x-2">
                        <button
                            onClick={() => handleFilterChange({ page: pagination.page - 1 })}
                            disabled={pagination.page === 1}
                            className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-slate-900 dark:text-white hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-50"
                        >
                            Previous
                        </button>
                        <span className="px-4 py-2 text-slate-700 dark:text-slate-300">
                            Page {pagination.page} of {pagination.totalPages}
                        </span>
                        <button
                            onClick={() => handleFilterChange({ page: pagination.page + 1 })}
                            disabled={pagination.page === pagination.totalPages}
                            className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-slate-900 dark:text-white hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-50"
                        >
                            Next
                        </button>
                    </div>
                </div>
            )}

            {/* Tenant Selection Modal (for onboarding) */}
            {canManageAdmin && (
                <TenantSelectionModal
                    isOpen={showSelectionModal}
                    onClose={() => setShowSelectionModal(false)}
                    onSelect={handleOnboardNewTenant}
                    isLoading={submitting}
                />
            )}

            {/* Tenant Form Modal (for editing) */}
            {canManageAdmin && (
                <TenantFormModal
                    isOpen={showTenantModal}
                    tenant={selectedTenant}
                    onClose={() => {
                        setShowTenantModal(false)
                        setSelectedTenant(null)
                    }}
                    onSubmit={handleUpdateTenant}
                    isLoading={submitting}
                />
            )}
        </div>
    )
}

export default TenantManagementPage
