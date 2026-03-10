import { useRefresh } from '@/context/RefreshContext'
import { useCanManageAdmin, useIsSystemAdmin } from '@/hooks/useAccess'
import { adminService } from '@/services/adminService'
import { InvitationFilters, Tenant, UserInvitation, UserRoleWithPermissions } from '@/types'
import { ArrowPathIcon, TrashIcon } from '@heroicons/react/24/outline'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'

interface SendInvitationDialogProps {
    isOpen: boolean
    onClose: () => void
    onSubmit: (data: { email: string; tenantId: string; roleId: string; message?: string }) => Promise<void>
    tenants: Tenant[]
    isLoading?: boolean
}

const SendInvitationDialog: React.FC<SendInvitationDialogProps> = ({
    isOpen,
    onClose,
    onSubmit,
    tenants,
    isLoading = false,
}) => {
    const [formData, setFormData] = useState({
        email: '',
        tenantId: '',
        roleId: '',
        message: '',
    })
    const [errors, setErrors] = useState<Record<string, string>>({})
    const [roles, setRoles] = useState<UserRoleWithPermissions[]>([])
    const [loadingRoles, setLoadingRoles] = useState(false)

    // Auto-select first tenant if only one available
    useEffect(() => {
        if (tenants.length === 1 && !formData.tenantId) {
            setFormData(prev => ({ ...prev, tenantId: tenants[0].id }))
        }
    }, [tenants, formData.tenantId])

    // Fetch roles when tenant changes
    useEffect(() => {
        if (formData.tenantId) {
            loadRoles(formData.tenantId)
        } else {
            setRoles([])
        }
    }, [formData.tenantId])

    const loadRoles = async (tenantId: string) => {
        try {
            setLoadingRoles(true)
            const fetchedRoles = await adminService.getRolesByTenant(tenantId)
            setRoles(fetchedRoles)
            // Auto-select first role if available and no role is selected
            if (fetchedRoles.length > 0 && !formData.roleId) {
                setFormData(prev => ({ ...prev, roleId: fetchedRoles[0].id }))
            }
        } catch (error: any) {
            toast.error('Failed to load roles')
            setRoles([])
        } finally {
            setLoadingRoles(false)
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setErrors({})

        // Client-side validation
        const validationErrors: Record<string, string> = {}
        if (!formData.email.trim()) {
            validationErrors.email = 'Email is required'
        }
        if (tenants.length === 0) {
            validationErrors.tenantId = 'No tenants available. Please create a tenant first.'
        } else if (!formData.tenantId.trim()) {
            validationErrors.tenantId = 'Please select a tenant'
        }
        if (roles.length === 0) {
            validationErrors.roleId = 'No roles available for the selected tenant.'
        } else if (!formData.roleId.trim()) {
            validationErrors.roleId = 'Please select a role'
        }

        if (Object.keys(validationErrors).length > 0) {
            setErrors(validationErrors)
            return
        }

        try {
            await onSubmit(formData)
            setFormData({ email: '', tenantId: '', roleId: '', message: '' })
            onClose()
        } catch (error: any) {
            if (error.message.includes('validation')) {
                // Parse validation errors
                const serverValidationErrors: Record<string, string> = {}
                if (error.message.includes('email')) serverValidationErrors.email = 'Invalid email address'
                if (error.message.includes('tenant')) serverValidationErrors.tenantId = 'Please select a tenant'
                if (error.message.includes('role')) serverValidationErrors.roleId = 'Please select a role'
                setErrors(serverValidationErrors)
            } else {
                toast.error(error.message || 'Failed to send invitation')
            }
        }
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
            <div className="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                <div className="mt-3">
                    <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Send User Invitation</h3>

                    <form onSubmit={handleSubmit} className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Email Address *
                            </label>
                            <input
                                type="email"
                                value={formData.email}
                                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                                placeholder="user@example.com"
                                className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 ${errors.email ? 'border-red-500' : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white'
                                    }`}
                                required
                                disabled={isLoading}
                            />
                            {errors.email && (
                                <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.email}</p>
                            )}
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Tenant *
                            </label>
                            <select
                                value={formData.tenantId}
                                onChange={(e) => setFormData({ ...formData, tenantId: e.target.value })}
                                className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 ${errors.tenantId ? 'border-red-500' : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white'
                                    }`}
                                required
                                disabled={isLoading}
                            >
                                <option value="">Select a tenant</option>
                                {tenants.length === 0 ? (
                                    <option value="" disabled>No tenants available</option>
                                ) : (
                                    tenants.map((tenant) => (
                                        <option key={tenant.id} value={tenant.id}>
                                            {tenant.name}
                                        </option>
                                    ))
                                )}
                            </select>
                            {errors.tenantId && (
                                <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.tenantId}</p>
                            )}
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Role *
                            </label>
                            <select
                                value={formData.roleId}
                                onChange={(e) => setFormData({ ...formData, roleId: e.target.value })}
                                className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 ${errors.roleId ? 'border-red-500' : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white'
                                    }`}
                                required
                                disabled={isLoading || loadingRoles}
                            >
                                <option value="">
                                    {loadingRoles ? 'Loading roles...' : 'Select a role'}
                                </option>
                                {roles.length === 0 && !loadingRoles ? (
                                    <option value="" disabled>No roles available</option>
                                ) : (
                                    roles
                                        .filter(role => {
                                            // Only allow System Administrator role for System Administrators tenant
                                            const selectedTenant = tenants.find(t => t.id === formData.tenantId)
                                            if (role.name === 'System Administrator') {
                                                return selectedTenant?.name === 'System Administrators'
                                            }
                                            return true
                                        })
                                        .map((role) => (
                                            <option key={role.id} value={role.id}>
                                                {role.name} - {role.description}
                                            </option>
                                        ))
                                )}
                            </select>
                            {errors.roleId && (
                                <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.roleId}</p>
                            )}
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Message (Optional)
                            </label>
                            <textarea
                                value={formData.message}
                                onChange={(e) => setFormData({ ...formData, message: e.target.value })}
                                placeholder="Personal message to include with the invitation..."
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                                rows={3}
                                disabled={isLoading}
                            />
                        </div>

                        <div className="flex justify-end space-x-3 pt-4">
                            <button
                                type="button"
                                onClick={onClose}
                                disabled={isLoading}
                                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-600 border border-gray-300 dark:border-gray-500 rounded-md hover:bg-gray-200 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={isLoading}
                                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                            >
                                {isLoading ? 'Sending...' : 'Send Invitation'}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    )
}

const UserInvitationsPage: React.FC = () => {
    const isSystemAdmin = useIsSystemAdmin()
    const canManageAdmin = useCanManageAdmin()
    const { registerRefreshCallback, unregisterRefreshCallback } = useRefresh()

    const [invitations, setInvitations] = useState<UserInvitation[]>([])
    const [tenants, setTenants] = useState<Tenant[]>([])
    const [loading, setLoading] = useState(true)
    const [submitting, setSubmitting] = useState(false)
    const [showSendDialog, setShowSendDialog] = useState(false)
    const [filters, setFilters] = useState<InvitationFilters>({
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
        if (isSystemAdmin) {
            loadTenants()
        }
        loadInvitations()
    }, [filters, isSystemAdmin])

    useEffect(() => {
        const refreshCallback = async () => {
            await loadInvitations()
        }

        registerRefreshCallback(refreshCallback)
        return () => unregisterRefreshCallback(refreshCallback)
    }, [])

    const loadInvitations = async () => {
        try {
            setLoading(true)
            const response = await adminService.listInvitations(filters)
            // Backend returns { invitations: [], total: number }
            // Frontend expects PaginatedResponse with data and pagination
            setInvitations(response.invitations || [])
            setPagination({
                page: filters.page || 1,
                limit: filters.limit || 20,
                total: response.total || 0,
                totalPages: Math.ceil((response.total || 0) / (filters.limit || 20)),
            })
        } catch (error: any) {
            toast.error('Failed to load invitations')
        } finally {
            setLoading(false)
        }
    }

    const loadTenants = async () => {
        try {
            const response = await adminService.getTenants()
            const tenantList = response?.data || []
            setTenants(tenantList)
        } catch (error: any) {
            toast.error('Failed to load tenants')
            setTenants([]) // Ensure tenants is empty array
        }
    }

    const handleSendInvitation = async (data: { email: string; tenantId: string; roleId: string; message?: string }) => {
        setSubmitting(true)
        try {
            await adminService.sendInvitation(data)
            toast.success('Invitation sent successfully')
            await loadInvitations()
        } catch (error: any) {
            throw error
        } finally {
            setSubmitting(false)
        }
    }

    const handleRevokeInvitation = async (invitationId: string) => {
        if (!confirm('Are you sure you want to revoke this invitation?')) return

        try {
            await adminService.revokeInvitation(invitationId)
            toast.success('Invitation revoked successfully')
            await loadInvitations()
        } catch (error: any) {
            toast.error('Failed to revoke invitation')
        }
    }

    const handleResendInvitation = async (invitationId: string) => {
        try {
            await adminService.resendInvitation(invitationId)
            toast.success('Invitation resent successfully')
        } catch (error: any) {
            toast.error('Failed to resend invitation')
        }
    }

    const handleFilterChange = (newFilters: Partial<InvitationFilters>) => {
        setFilters({ ...filters, ...newFilters, page: 1 })
    }

    const getTenantName = (tenantId: string) => {
        if (!tenantId) return 'Unknown Tenant'
        const tenant = tenants.find(t => t.id === tenantId)
        return tenant?.name || `Tenant (${tenantId.slice(0, 8)}...)`
    }

    if (!isSystemAdmin) {
        return (
            <div className="p-6">
                <div className="text-center">
                    <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">Access Denied</h2>
                    <p className="text-gray-600 dark:text-gray-400">You need system administrator privileges to manage user invitations.</p>
                </div>
            </div>
        )
    }

    return (
        <div className="p-6">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white">User Invitations</h1>
                    <p className="text-gray-600 dark:text-gray-400">Manage user invitations and track their status</p>
                </div>
                {canManageAdmin && (
                    <button
                        onClick={() => setShowSendDialog(true)}
                        disabled={tenants.length === 0}
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        title={tenants.length === 0 ? 'No tenants available. Please create a tenant first.' : 'Send a new invitation'}
                    >
                        Invite User
                    </button>
                )}
            </div>

            {!canManageAdmin && (
                <div className="mb-4 rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: invitation send/resend/revoke actions are hidden for System Administrator Viewer.
                </div>
            )}

            {/* Filters */}
            <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow mb-6 border border-gray-200 dark:border-gray-700">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Status
                        </label>
                        <select
                            value={filters.status?.[0] || ''}
                            onChange={(e) => handleFilterChange({
                                status: e.target.value ? [e.target.value] : undefined
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                        >
                            <option value="">All Status</option>
                            <option value="pending">Pending</option>
                            <option value="accepted">Accepted</option>
                            <option value="expired">Expired</option>
                            <option value="revoked">Revoked</option>
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Tenant
                        </label>
                        <select
                            value={filters.tenantId || ''}
                            onChange={(e) => handleFilterChange({ tenantId: e.target.value || undefined })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                        >
                            <option value="">All Tenants</option>
                            {tenants.map((tenant) => (
                                <option key={tenant.id} value={tenant.id}>
                                    {tenant.name}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Search
                        </label>
                        <input
                            type="text"
                            placeholder="Search by email..."
                            value={filters.search || ''}
                            onChange={(e) => handleFilterChange({ search: e.target.value || undefined })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                        />
                    </div>

                    <div className="flex items-end">
                        <button
                            onClick={() => setFilters({ page: 1, limit: 20 })}
                            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-600 border border-gray-300 dark:border-gray-500 rounded-md hover:bg-gray-200 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                        >
                            Clear Filters
                        </button>
                    </div>
                </div>
            </div>

            {/* Invitations Table */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden border border-gray-200 dark:border-gray-700">
                {loading ? (
                    <div className="p-8 text-center">
                        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                        <p className="mt-2 text-gray-600 dark:text-gray-400">Loading invitations...</p>
                    </div>
                ) : invitations.length === 0 ? (
                    <div className="p-8 text-center text-gray-500 dark:text-gray-400">
                        No invitations found
                    </div>
                ) : (
                    <>
                        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Email
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Tenant
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Status
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Role
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Expires
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Created
                                    </th>
                                    {canManageAdmin && (
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Actions
                                        </th>
                                    )}
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {invitations.map((invitation) => (
                                    <tr key={invitation.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                {invitation.email}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm text-gray-900 dark:text-gray-300">
                                                {getTenantName(invitation.tenantId)}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${invitation.status === 'pending' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' :
                                                invitation.status === 'accepted' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' :
                                                    invitation.status === 'expired' ? 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300' :
                                                        'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                                                }`}>
                                                {invitation.status}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm text-gray-900 dark:text-gray-300">
                                                {invitation.roleName || invitation.roleId || 'Default'}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm text-gray-900 dark:text-gray-300">
                                                {new Date(invitation.expiresAt).toLocaleDateString()}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm text-gray-900 dark:text-gray-300">
                                                {new Date(invitation.createdAt).toLocaleDateString()}
                                            </div>
                                        </td>
                                        {canManageAdmin && (
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                                <div className="flex space-x-2">
                                                    {invitation.status === 'pending' && (
                                                        <div className="flex space-x-2">
                                                            <button
                                                                onClick={() => handleResendInvitation(invitation.id)}
                                                                className="p-1 text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded"
                                                                title="Resend invitation"
                                                            >
                                                                <ArrowPathIcon className="w-4 h-4" />
                                                            </button>
                                                            <button
                                                                onClick={() => handleRevokeInvitation(invitation.id)}
                                                                className="p-1 text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                                                                title="Revoke invitation"
                                                            >
                                                                <TrashIcon className="w-4 h-4" />
                                                            </button>
                                                        </div>
                                                    )}
                                                </div>
                                            </td>
                                        )}
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </>
                )}

                {/* Pagination */}
                {pagination.totalPages > 1 && (
                    <div className="bg-white dark:bg-gray-800 px-4 py-3 flex items-center justify-between border-t border-gray-200 dark:border-gray-700 sm:px-6">
                        <div className="flex-1 flex justify-between sm:hidden">
                            <button
                                onClick={() => handleFilterChange({ page: Math.max(1, pagination.page - 1) })}
                                disabled={pagination.page === 1}
                                className="relative inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
                            >
                                Previous
                            </button>
                            <button
                                onClick={() => handleFilterChange({ page: Math.min(pagination.totalPages, pagination.page + 1) })}
                                disabled={pagination.page === pagination.totalPages}
                                className="ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
                            >
                                Next
                            </button>
                        </div>
                        <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                            <div>
                                <p className="text-sm text-gray-700 dark:text-gray-300">
                                    Showing <span className="font-medium">{(pagination.page - 1) * pagination.limit + 1}</span> to{' '}
                                    <span className="font-medium">
                                        {Math.min(pagination.page * pagination.limit, pagination.total)}
                                    </span> of{' '}
                                    <span className="font-medium">{pagination.total}</span> results
                                </p>
                            </div>
                            <div>
                                <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px">
                                    <button
                                        onClick={() => handleFilterChange({ page: Math.max(1, pagination.page - 1) })}
                                        disabled={pagination.page === 1}
                                        className="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-sm font-medium text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
                                    >
                                        Previous
                                    </button>
                                    {Array.from({ length: Math.min(5, pagination.totalPages) }, (_, i) => {
                                        const pageNum = Math.max(1, Math.min(pagination.totalPages - 4, pagination.page - 2)) + i
                                        return (
                                            <button
                                                key={pageNum}
                                                onClick={() => handleFilterChange({ page: pageNum })}
                                                className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium ${pageNum === pagination.page
                                                    ? 'z-10 bg-blue-50 dark:bg-blue-900 border-blue-500 text-blue-600 dark:text-blue-400'
                                                    : 'bg-white dark:bg-gray-700 border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600'
                                                    }`}
                                            >
                                                {pageNum}
                                            </button>
                                        )
                                    })}
                                    <button
                                        onClick={() => handleFilterChange({ page: Math.min(pagination.totalPages, pagination.page + 1) })}
                                        disabled={pagination.page === pagination.totalPages}
                                        className="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-sm font-medium text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
                                    >
                                        Next
                                    </button>
                                </nav>
                            </div>
                        </div>
                    </div>
                )}
            </div>

            {/* Send Invitation Dialog */}
            <SendInvitationDialog
                isOpen={showSendDialog}
                onClose={() => setShowSendDialog(false)}
                onSubmit={handleSendInvitation}
                tenants={tenants}
                isLoading={submitting}
            />
        </div>
    )
}

export default UserInvitationsPage
