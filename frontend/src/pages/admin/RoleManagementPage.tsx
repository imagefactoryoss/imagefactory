import { PermissionCategoryTabs } from '@/components/PermissionCategoryTabs'
import { useRefresh } from '@/context/RefreshContext'
import { useCanManageAdmin } from '@/hooks/useAccess'
import { usePermissions } from '@/hooks/usePermissions'
import { adminService } from '@/services/adminService'
import { Permission, UserRoleWithPermissions } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

// Role Form Modal Component
interface RoleFormModalProps {
    isOpen: boolean
    role: UserRoleWithPermissions | null
    onClose: () => void
    onSubmit: (data: any) => Promise<void>
    isLoading?: boolean
}

const RoleFormModal: React.FC<RoleFormModalProps> = ({ isOpen, role, onClose, onSubmit, isLoading }) => {
    const [formData, setFormData] = useState({
        name: '',
        description: '',
        permissions: [] as string[],
    })
    const [errors, setErrors] = useState<Record<string, string>>({})
    const { permissions, loading: permissionsLoading, error: permissionsError } = usePermissions()

    // Convert permission IDs to format compatible with API ("resource:action")
    const selectedPermissionIds = useMemo(() => {
        return new Set(
            permissions
                .filter((p) => {
                    const permKey = `${p.resource}:${p.action}`
                    return formData.permissions.includes(permKey) || formData.permissions.includes(p.id)
                })
                .map((p) => p.id)
        )
    }, [permissions, formData.permissions])

    useEffect(() => {
        if (role) {
            setFormData({
                name: role.name || '',
                description: role.description || '',
                permissions: role.permissions?.map(p => {
                    if (typeof p === 'string') return p
                    // Convert Permission object to "resource:action" format
                    const perm = p as any
                    return `${perm.resource}:${perm.action}`
                }) || [],
            })
        } else {
            setFormData({
                name: '',
                description: '',
                permissions: [],
            })
        }
        setErrors({})
    }, [role, isOpen])

    const validateForm = () => {
        const newErrors: Record<string, string> = {}

        if (!formData.name) newErrors.name = 'Role name is required'
        if (formData.permissions.length === 0) newErrors.permissions = 'At least one permission is required'

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!validateForm()) return

        try {
            await onSubmit({
                name: formData.name,
                description: formData.description,
                permissions: formData.permissions,
            })
            onClose()
        } catch (error) {
        }
    }

    const handlePermissionToggle = (permissionId: string, selected: boolean) => {
        // Find the permission object to get resource:action
        const permission = permissions.find((p) => p.id === permissionId)
        if (!permission) return

        const permKey = `${permission.resource}:${permission.action}`
        setFormData((prev) => ({
            ...prev,
            permissions: selected
                ? [...prev.permissions, permKey]
                : prev.permissions.filter((p) => p !== permKey),
        }))
    }

    const handleSelectAll = () => {
        setFormData((prev) => ({
            ...prev,
            permissions: permissions.map((p) => `${p.resource}:${p.action}`),
        }))
    }

    const handleDeselectAll = () => {
        setFormData((prev) => ({
            ...prev,
            permissions: [],
        }))
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center p-4">
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-y-auto">
                {/* Header */}
                <div className="sticky top-0 px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between bg-white dark:bg-slate-800 z-10">
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                        {role ? 'Edit Role' : 'Create Role'}
                    </h2>
                    <button
                        onClick={onClose}
                        className="text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white"
                    >
                        ✕
                    </button>
                </div>

                {/* Form */}
                <form onSubmit={handleSubmit} className="p-6 space-y-4">
                    {/* Role Name */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Role Name *
                        </label>
                        <input
                            type="text"
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                            placeholder="Developer"
                        />
                        {errors.name && <p className="text-xs text-red-600 mt-1">{errors.name}</p>}
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
                            placeholder="Brief description of this role"
                            rows={2}
                        />
                    </div>

                    {/* Permissions */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
                            Permissions * ({formData.permissions.length} selected)
                        </label>
                        {permissionsError && (
                            <p className="text-xs text-red-600 mb-2">Error loading permissions: {permissionsError}</p>
                        )}
                        {permissionsLoading ? (
                            <p className="text-sm text-slate-500 dark:text-slate-400">Loading permissions...</p>
                        ) : (
                            <>
                                {errors.permissions && <p className="text-xs text-red-600 mb-2">{errors.permissions}</p>}
                                {/* Use tabbed interface for permissions */}
                                <PermissionCategoryTabs
                                    permissions={permissions}
                                    assignedPermissionIds={new Set()} // Empty since we don't have assigned yet
                                    selectedPermissions={selectedPermissionIds}
                                    onPermissionToggle={handlePermissionToggle}
                                    onSelectAll={handleSelectAll}
                                    onDeselectAll={handleDeselectAll}
                                />
                            </>
                        )}
                    </div>

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
                            {isLoading ? 'Saving...' : role ? 'Update Role' : 'Create Role'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    )
}

// Permission Viewer Modal Component
interface PermissionViewerModalProps {
    isOpen: boolean
    role: UserRoleWithPermissions | null
    onClose: () => void
}

const PermissionViewerModal: React.FC<PermissionViewerModalProps> = ({ isOpen, role, onClose }) => {
    if (!isOpen || !role) return null

    // Group permissions by resource
    const groupedPermissions: Record<string, Array<{ resource: string; action: string; description?: string }>> = {}

    if (role.permissions && Array.isArray(role.permissions)) {
        role.permissions.forEach((perm: Permission | string) => {
            let resource: string
            let action: string

            if (typeof perm === 'string') {
                // Parse "resource:action" format
                const parts = perm.split(':')
                resource = parts[0] || 'Other'
                action = parts[1] || perm
            } else {
                // Use object properties
                resource = perm.resource || 'Other'
                action = perm.action || perm.name || 'Unknown'
            }

            if (!groupedPermissions[resource]) {
                groupedPermissions[resource] = []
            }
            groupedPermissions[resource].push({
                resource,
                action,
                description: typeof perm === 'string' ? undefined : perm.description,
            })
        })
    }

    const sortedResources = Object.keys(groupedPermissions).sort()

    return (
        <div className="fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center p-4">
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-xl max-w-3xl w-full max-h-[80vh] overflow-y-auto">
                {/* Header */}
                <div className="sticky top-0 px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between bg-white dark:bg-slate-800">
                    <div>
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                            {role.name} - Permissions
                        </h2>
                        <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                            Total: {role.permissions?.length || 0} permissions
                        </p>
                    </div>
                    <button
                        onClick={onClose}
                        className="text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white"
                    >
                        ✕
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 space-y-6">
                    {sortedResources.length === 0 ? (
                        <p className="text-center text-slate-600 dark:text-slate-400">No permissions assigned</p>
                    ) : (
                        sortedResources.map((resource) => (
                            <div key={resource}>
                                <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3 capitalize">
                                    {resource} ({groupedPermissions[resource].length})
                                </h3>
                                <div className="grid grid-cols-2 gap-2">
                                    {groupedPermissions[resource].map((perm, idx) => (
                                        <div
                                            key={idx}
                                            className="flex items-start space-x-2 p-3 rounded bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800"
                                        >
                                            <span className="text-green-600 dark:text-green-400 mt-0.5">✓</span>
                                            <div className="flex-1">
                                                <p className="text-sm font-medium text-green-800 dark:text-green-200">
                                                    {perm.action}
                                                </p>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        ))
                    )}
                </div>

                {/* Footer */}
                <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex justify-end">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                    >
                        Close
                    </button>
                </div>
            </div>
        </div>
    )
}

const RoleManagementPage: React.FC = () => {
    const canManageAdmin = useCanManageAdmin()
    const [roles, setRoles] = useState<UserRoleWithPermissions[]>([])
    const [loading, setLoading] = useState(true)
    const [submitting, setSubmitting] = useState(false)
    const [showRoleModal, setShowRoleModal] = useState(false)
    const [showPermissionViewer, setShowPermissionViewer] = useState(false)
    const [selectedRole, setSelectedRole] = useState<UserRoleWithPermissions | null>(null)
    const { registerRefreshCallback, unregisterRefreshCallback } = useRefresh()

    useEffect(() => {
        loadRoles()

        // Register refresh callback
        const refreshCallback = async () => {
            await loadRoles()
        }
        registerRefreshCallback(refreshCallback)

        // Cleanup on unmount
        return () => {
            unregisterRefreshCallback(refreshCallback)
        }
    }, [])

    const loadRoles = async () => {
        try {
            setLoading(true)
            const response = await adminService.getRoles()
            setRoles(response || [])
        } catch (error: any) {
            toast.error(error.message || 'Failed to load roles')
            setRoles([])
        } finally {
            setLoading(false)
        }
    }

    const handleCreateRole = async (data: any) => {
        try {
            setSubmitting(true)
            await adminService.createRole(data)
            toast.success('Role created successfully')
            loadRoles()
        } catch (error: any) {
            toast.error(error.message || 'Failed to create role')
        } finally {
            setSubmitting(false)
        }
    }

    const handleUpdateRole = async (data: any) => {
        if (!selectedRole) return

        try {
            setSubmitting(true)
            await adminService.updateRole(selectedRole.id, data)
            toast.success('Role updated successfully')
            loadRoles()
        } catch (error: any) {
            toast.error(error.message || 'Failed to update role')
        } finally {
            setSubmitting(false)
        }
    }

    const handleDeleteRole = async (roleId: string) => {
        if (!confirm('Are you sure you want to delete this role? This action cannot be undone.')) return

        try {
            await adminService.deleteRole(roleId)
            toast.success('Role deleted successfully')
            loadRoles()
        } catch (error: any) {
            toast.error(error.message || 'Failed to delete role')
        }
    }

    // Calculate statistics
    const stats = useMemo(() => {
        const totalRoles = roles.length
        const totalPermissionsAssigned = roles.reduce((sum, role) => sum + (role.permissions?.length || 0), 0)
        const averagePermissionsPerRole = totalRoles > 0 ? Math.round(totalPermissionsAssigned / totalRoles) : 0

        return {
            totalRoles,
            totalPermissionsAssigned,
            averagePermissionsPerRole
        }
    }, [roles])

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                    <h1 className="text-3xl font-bold text-slate-900 dark:text-white">Roles & Permissions</h1>
                    <p className="mt-2 text-slate-600 dark:text-slate-400">
                        Define roles and manage permissions for your users
                    </p>
                </div>

                <div className="flex-shrink-0">
                    {canManageAdmin && (
                        <button
                            onClick={() => {
                                setSelectedRole(null)
                                setShowRoleModal(true)
                            }}
                            className="inline-flex items-center justify-center rounded-md border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900"
                        >
                            + Add Role
                        </button>
                    )}
                </div>
            </div>

            {!canManageAdmin && (
                <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: role create, edit, and delete actions are hidden for System Administrator Viewer.
                </div>
            )}

            {/* Info Banner - Role-Centric View */}
            <div className="rounded-lg border border-purple-200 dark:border-purple-900/50 bg-purple-50 dark:bg-purple-900/20 p-4">
                <div className="flex items-start space-x-3">
                    <svg className="h-5 w-5 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M18 5v8a2 2 0 01-2 2h-5l-5 4v-4H4a2 2 0 01-2-2V5a2 2 0 012-2h12a2 2 0 012 2zm-11-1H7v2h2V4zm2 4H9v2h2V8zm2-4h-2v2h2V4zm2 4h-2v2h2V8z" clipRule="evenodd" />
                    </svg>
                    <div className="flex-1">
                        <h3 className="text-sm font-semibold text-purple-900 dark:text-purple-200">📌 Role-Centric View</h3>
                        <p className="mt-1 text-sm text-purple-700 dark:text-purple-300">
                            Create and organize roles. Each role is a collection of permissions. Use this view when you want to <strong>create a new role</strong> or <strong>manage what a specific role can do</strong>.
                        </p>
                        <p className="mt-2 text-sm text-purple-700 dark:text-purple-300">
                            <strong>Different from Permissions page?</strong> The Permissions page is permission-centric—assign the same permission to multiple roles at once. This page is role-centric—define what permissions belong to a single role.
                        </p>
                    </div>
                </div>
            </div>

            {/* Statistics Cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-white dark:bg-slate-950 rounded-lg shadow-md p-6 border border-slate-200 dark:border-slate-800">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="h-10 w-10 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
                                <svg className="h-6 w-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                                </svg>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-slate-600 dark:text-slate-400">Total Roles</p>
                            <p className="text-2xl font-bold text-slate-900 dark:text-white">{stats.totalRoles}</p>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-950 rounded-lg shadow-md p-6 border border-slate-200 dark:border-slate-800">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="h-10 w-10 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                                <svg className="h-6 w-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-slate-600 dark:text-slate-400">Permissions Assigned</p>
                            <p className="text-2xl font-bold text-slate-900 dark:text-white">{stats.totalPermissionsAssigned}</p>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-950 rounded-lg shadow-md p-6 border border-slate-200 dark:border-slate-800">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="h-10 w-10 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
                                <svg className="h-6 w-6 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 8v8m-4-5v5m-4-2v2m-2 4h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                                </svg>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-slate-600 dark:text-slate-400">Avg Permissions/Role</p>
                            <p className="text-2xl font-bold text-slate-900 dark:text-white">{stats.averagePermissionsPerRole}</p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Roles Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {loading ? (
                    <div className="col-span-full flex justify-center items-center py-12">
                        <div className="flex justify-center items-center space-x-2">
                            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                            <span className="text-slate-600 dark:text-slate-400">Loading roles...</span>
                        </div>
                    </div>
                ) : roles.length === 0 ? (
                    <div className="col-span-full text-center py-12 text-slate-600 dark:text-slate-400">
                        No roles found. Create one to get started.
                    </div>
                ) : (
                    roles.map((role) => (
                        <div
                            key={role.id}
                            className="bg-white dark:bg-slate-950 rounded-lg shadow-md p-6 border border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700 transition-colors"
                        >
                            {/* Role Header */}
                            <div className="mb-4">
                                <h3 className="text-lg font-semibold text-slate-900 dark:text-white">{role.name}</h3>
                                <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{role.description || 'No description'}</p>
                            </div>

                            {/* Permissions Summary */}
                            <div className="mb-4">
                                <p className="text-xs font-medium text-slate-700 dark:text-slate-300 mb-2">
                                    {role.permissions?.length || 0} Permissions
                                </p>
                                <div className="flex flex-wrap gap-1">
                                    {role.permissions?.slice(0, 3).map((perm) => {
                                        const permName = typeof perm === 'string' ? perm : perm.name
                                        return (
                                            <span
                                                key={typeof perm === 'string' ? perm : perm.id}
                                                className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200"
                                            >
                                                {permName}
                                            </span>
                                        )
                                    })}
                                    {role.permissions && role.permissions.length > 3 && (
                                        <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200">
                                            +{role.permissions.length - 3} more
                                        </span>
                                    )}
                                </div>
                            </div>

                            {/* Actions */}
                            <div className="flex gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                                <button
                                    onClick={() => {
                                        setSelectedRole(role)
                                        setShowPermissionViewer(true)
                                    }}
                                    className="flex-1 px-3 py-2 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900 rounded-md"
                                >
                                    View Permissions
                                </button>
                                {canManageAdmin && (
                                    <button
                                        onClick={() => {
                                            setSelectedRole(role)
                                            setShowRoleModal(true)
                                        }}
                                        className="flex-1 px-3 py-2 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900 rounded-md"
                                    >
                                        Edit
                                    </button>
                                )}
                                {canManageAdmin && (
                                    <button
                                        onClick={() => handleDeleteRole(role.id)}
                                        className="flex-1 px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900 rounded-md"
                                    >
                                        Delete
                                    </button>
                                )}
                            </div>
                        </div>
                    ))
                )}
            </div>

            {/* Role Form Modal */}
            <RoleFormModal
                isOpen={showRoleModal}
                role={selectedRole}
                onClose={() => {
                    setShowRoleModal(false)
                    setSelectedRole(null)
                }}
                onSubmit={selectedRole ? handleUpdateRole : handleCreateRole}
                isLoading={submitting}
            />

            {/* Permission Viewer Modal */}
            <PermissionViewerModal
                isOpen={showPermissionViewer}
                role={selectedRole}
                onClose={() => {
                    setShowPermissionViewer(false)
                    setSelectedRole(null)
                }}
            />
        </div>
    )
}

export default RoleManagementPage
