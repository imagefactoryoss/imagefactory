import { PermissionCategoryTabs } from '@/components/PermissionCategoryTabs'
import { useCanManageAdmin } from '@/hooks/useAccess'
import { useAddPermissionToRole, usePermissions, useRemovePermissionFromRole, useRolePermissions } from '@/hooks/usePermissions'
import { adminService } from '@/services/adminService'
import { api } from '@/services/api'
import { UserRoleWithPermissions } from '@/types'
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import React, { useMemo, useState } from 'react'
import toast from 'react-hot-toast'

// Error boundary for this page
class PermissionPageErrorBoundary extends React.Component<
    { children: React.ReactNode },
    { hasError: boolean; error: string | null }
> {
    constructor(props: { children: React.ReactNode }) {
        super(props)
        this.state = { hasError: false, error: null }
    }

    static getDerivedStateFromError(error: Error) {
        return { hasError: true, error: error.message }
    }

    componentDidCatch(_error: Error, _errorInfo: React.ErrorInfo) {
    }

    render() {
        if (this.state.hasError) {
            return (
                <div className="p-4 bg-red-50 border border-red-200 rounded text-red-700">
                    <h2 className="font-bold">Error Loading Permissions Page</h2>
                    <p>{this.state.error}</p>
                    <button onClick={() => window.location.reload()} className="mt-2 px-4 py-2 bg-red-600 text-white rounded">
                        Reload Page
                    </button>
                </div>
            )
        }

        return this.props.children
    }
}

const PermissionManagementPage: React.FC = () => {
    const canManageAdmin = useCanManageAdmin()
    const { permissions, loading: permissionsLoading } = usePermissions()
    const [selectedRole, setSelectedRole] = useState<UserRoleWithPermissions | null>(null)
    const [roles, setRoles] = useState<UserRoleWithPermissions[]>([])
    const [rolesLoading, setRolesLoading] = useState(true)
    const [roleSearchQuery, setRoleSearchQuery] = useState('')
    const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set())
    const [bulkOperationLoading, setBulkOperationLoading] = useState(false)
    const [multiSelectMode, setMultiSelectMode] = useState(false)
    const [selectedRoles, setSelectedRoles] = useState<Set<string>>(new Set())
    const { rolePermissions, refetch: refetchRolePermissions } = useRolePermissions(selectedRole?.id || null)
    const { addPermission, loading: addingPermission } = useAddPermissionToRole()
    const { removePermission, loading: removingPermission } = useRemovePermissionFromRole()

    // Fetch all roles on mount
    React.useEffect(() => {
        const loadRoles = async () => {
            try {
                setRolesLoading(true)
                const data = await adminService.getRoles()
                setRoles(data)
                if (data.length > 0 && !selectedRole) {
                    setSelectedRole(data[0])
                }
            } catch (error) {
                toast.error('Failed to load roles')
            } finally {
                setRolesLoading(false)
            }
        }
        loadRoles()
    }, [])

    // Filter roles based on search
    const filteredRoles = useMemo(() => {
        return roles.filter((role) =>
            role.name.toLowerCase().includes(roleSearchQuery.toLowerCase()) ||
            (role.description && role.description.toLowerCase().includes(roleSearchQuery.toLowerCase()))
        )
    }, [roles, roleSearchQuery])

    // Get assigned permission IDs for the selected role
    const assignedPermissionIds = useMemo(() => {
        // Use rolePermissions from API if available, otherwise fall back to selectedRole.permissions
        const sourcePermissions = rolePermissions.length > 0 ? rolePermissions : (selectedRole?.permissions || [])
        return new Set(sourcePermissions.map((p) => p.id))
    }, [rolePermissions, selectedRole?.permissions])

    const handleRoleChange = (roleId: string) => {
        const role = roles.find((r) => r.id === roleId)
        setSelectedRole(role || null)
        setSelectedPermissions(new Set()) // Clear selection when changing roles
    }

    const handleSelectAll = () => {
        const allPermissionIds = new Set(permissions.map(p => p.id))
        setSelectedPermissions(allPermissionIds)
    }

    const handleDeselectAll = () => {
        setSelectedPermissions(new Set())
    }

    const handlePermissionToggle = (permissionId: string, selected: boolean) => {
        const newSelected = new Set(selectedPermissions)
        if (selected) {
            newSelected.add(permissionId)
        } else {
            newSelected.delete(permissionId)
        }
        setSelectedPermissions(newSelected)
    }

    const handleBulkAddPermissions = async () => {
        if (!selectedRole || selectedPermissions.size === 0) return

        setBulkOperationLoading(true)
        try {
            const promises = Array.from(selectedPermissions).map(permId =>
                addPermission(selectedRole.id, permId)
            )
            await Promise.all(promises)
            setSelectedPermissions(new Set())
            refetchRolePermissions()
            toast.success(`Added ${selectedPermissions.size} permissions to role`)
        } catch (error) {
            toast.error('Failed to add some permissions')
        } finally {
            setBulkOperationLoading(false)
        }
    }

    const handleBulkRemovePermissions = async () => {
        if (!selectedRole || selectedPermissions.size === 0) return

        const confirmed = window.confirm(
            `Are you sure you want to remove ${selectedPermissions.size} permission${selectedPermissions.size !== 1 ? 's' : ''} from "${selectedRole.name}"?`
        )

        if (!confirmed) return

        setBulkOperationLoading(true)
        try {
            const promises = Array.from(selectedPermissions).map(permId =>
                removePermission(selectedRole.id, permId)
            )
            await Promise.all(promises)
            setSelectedPermissions(new Set())
            refetchRolePermissions()
            toast.success(`Removed ${selectedPermissions.size} permissions from role`)
        } catch (error) {
            toast.error('Failed to remove some permissions')
        } finally {
            setBulkOperationLoading(false)
        }
    }

    const handleToggleRoleSelection = (roleId: string) => {
        const newSelected = new Set(selectedRoles)
        if (newSelected.has(roleId)) {
            newSelected.delete(roleId)
        } else {
            newSelected.add(roleId)
        }
        setSelectedRoles(newSelected)
    }

    const handleBulkAddToMultipleRoles = async () => {
        if (selectedRoles.size === 0 || selectedPermissions.size === 0) return

        setBulkOperationLoading(true)
        try {
            // For each selected permission, use the bulk endpoint to assign it to all selected roles
            const bulkPromises = Array.from(selectedPermissions).map(permId =>
                api.post(`/permissions/${permId}/roles`, {
                    roleIds: Array.from(selectedRoles),
                })
            )

            await Promise.all(bulkPromises)
            setSelectedPermissions(new Set())
            setSelectedRoles(new Set())
            refetchRolePermissions()
            toast.success(
                `Added ${selectedPermissions.size} permission${selectedPermissions.size !== 1 ? 's' : ''} to ${selectedRoles.size} role${selectedRoles.size !== 1 ? 's' : ''}`
            )
        } catch (error) {
            toast.error('Failed to add some permissions')
        } finally {
            setBulkOperationLoading(false)
        }
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Permission Management</h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">Manage permissions for roles and users</p>
                </div>
            </div>

            {!canManageAdmin && (
                <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: permission assignment actions are hidden for System Administrator Viewer.
                </div>
            )}

            {/* Info Banner */}
            <div className="rounded-lg border border-blue-200 dark:border-blue-900/50 bg-blue-50 dark:bg-blue-900/20 p-4">
                <div className="flex items-start space-x-3">
                    <svg className="h-5 w-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M18 5v8a2 2 0 01-2 2h-5l-5 4v-4H4a2 2 0 01-2-2V5a2 2 0 012-2h12a2 2 0 012 2zm-11-1H7v2h2V4zm2 4H9v2h2V8zm2-4h-2v2h2V4zm2 4h-2v2h2V8z" clipRule="evenodd" />
                    </svg>
                    <div className="flex-1">
                        <h3 className="text-sm font-semibold text-blue-900 dark:text-blue-200">📌 Permission-Centric View</h3>
                        <p className="mt-1 text-sm text-blue-700 dark:text-blue-300">
                            Select a permission, then assign it to <strong>one or multiple roles</strong> at once. Use the toggle switch to switch between Single-role or Multi-role assignment modes.
                        </p>
                        <p className="mt-2 text-sm text-blue-700 dark:text-blue-300">
                            <strong>Different from Roles page?</strong> The Roles page is role-centric—create a new role and define what permissions it has. This page is permission-centric—pick a permission and decide which roles should have it.
                        </p>
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                {/* Left Sidebar - Roles List */}
                <div className="lg:col-span-1">
                    <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow">
                        <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-3 flex items-center justify-between">
                            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Roles</h2>
                            {canManageAdmin && (
                                <div className="flex items-center space-x-2">
                                    <span className={`text-xs font-medium ${!multiSelectMode ? 'text-gray-900 dark:text-white' : 'text-gray-500 dark:text-gray-400'}`}>Single</span>
                                    <button
                                        onClick={() => {
                                            setMultiSelectMode(!multiSelectMode)
                                            setSelectedRoles(new Set())
                                        }}
                                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${multiSelectMode
                                            ? 'bg-blue-600'
                                            : 'bg-gray-300 dark:bg-gray-600'
                                            }`}
                                        title="Toggle multi-select mode"
                                    >
                                        <span
                                            className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${multiSelectMode ? 'translate-x-5' : 'translate-x-0.5'
                                                }`}
                                        />
                                    </button>
                                    <span className={`text-xs font-medium ${multiSelectMode ? 'text-gray-900 dark:text-white' : 'text-gray-500 dark:text-gray-400'}`}>Multi</span>
                                </div>
                            )}
                        </div>

                        {/* Role Search */}
                        <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
                            <div className="relative">
                                <MagnifyingGlassIcon className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400 dark:text-gray-500" />
                                <input
                                    type="text"
                                    placeholder="Search roles..."
                                    value={roleSearchQuery}
                                    onChange={(e) => setRoleSearchQuery(e.target.value)}
                                    className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                        </div>

                        <div className="max-h-96 overflow-y-auto">
                            {rolesLoading ? (
                                <div className="px-4 py-6 text-center text-gray-500 dark:text-gray-400">Loading roles...</div>
                            ) : filteredRoles.length === 0 ? (
                                <div className="px-4 py-6 text-center text-gray-500 dark:text-gray-400">
                                    {roleSearchQuery ? 'No roles match your search' : 'No roles found'}
                                </div>
                            ) : (
                                <div className="divide-y divide-gray-200 dark:divide-gray-700">
                                    {filteredRoles.map((role) => (
                                        <button
                                            key={role.id}
                                            onClick={() => {
                                                if (multiSelectMode) {
                                                    handleToggleRoleSelection(role.id)
                                                } else {
                                                    handleRoleChange(role.id)
                                                }
                                            }}
                                            className={`w-full px-4 py-3 text-left transition-colors flex items-center space-x-3 ${multiSelectMode
                                                ? selectedRoles.has(role.id)
                                                    ? 'bg-blue-50 dark:bg-blue-900/30 text-gray-900 dark:text-white'
                                                    : 'hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100'
                                                : selectedRole?.id === role.id
                                                    ? 'bg-blue-50 dark:bg-blue-900/40 text-gray-900 dark:text-white border-l-2 border-blue-500 dark:border-blue-400'
                                                    : 'hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100'
                                                }`}
                                        >
                                            {multiSelectMode && (
                                                <input
                                                    type="checkbox"
                                                    checked={selectedRoles.has(role.id)}
                                                    onChange={() => handleToggleRoleSelection(role.id)}
                                                    onClick={(e) => e.stopPropagation()}
                                                    className="w-4 h-4 rounded border-gray-300"
                                                />
                                            )}
                                            <div className="flex-1">
                                                <div className="font-medium text-gray-900 dark:text-white">{role.name}</div>
                                                <div className="mt-1 text-xs text-gray-600 dark:text-gray-400">{role.permissions?.length || 0} permissions</div>
                                                {role.is_system && (
                                                    <div className="mt-1 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 dark:bg-purple-900/40 text-purple-800 dark:text-purple-300">
                                                        System
                                                    </div>
                                                )}
                                            </div>
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                {/* Right Content Area */}
                <div className="lg:col-span-2 space-y-6">
                    {/* Role Details Card */}
                    {multiSelectMode ? (
                        // Multi-select mode info
                        selectedRoles.size > 0 && (
                            <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow">
                                <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
                                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Selected Roles</h2>
                                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                                        {selectedRoles.size} role{selectedRoles.size !== 1 ? 's' : ''} selected
                                    </p>
                                </div>
                                <div className="px-6 py-4">
                                    <div className="space-y-2">
                                        {Array.from(selectedRoles).map((roleId) => {
                                            const role = roles.find(r => r.id === roleId)
                                            return role ? (
                                                <div key={roleId} className="flex items-center justify-between p-2 bg-blue-50 dark:bg-blue-900/30 rounded">
                                                    <span className="font-medium text-gray-900 dark:text-white">{role.name}</span>
                                                    <button
                                                        onClick={() => handleToggleRoleSelection(roleId)}
                                                        className="text-xs text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 font-medium"
                                                    >
                                                        Remove
                                                    </button>
                                                </div>
                                            ) : null
                                        })}
                                    </div>
                                </div>
                            </div>
                        )
                    ) : (
                        // Single-select mode info
                        selectedRole && (
                            <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow">
                                <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
                                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{selectedRole.name}</h2>
                                    {selectedRole.description && <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{selectedRole.description}</p>}
                                </div>
                                <div className="px-6 py-4">
                                    <div className="grid grid-cols-2 gap-4">
                                        <div>
                                            <p className="text-sm text-gray-600 dark:text-gray-400">Assigned Permissions</p>
                                            <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
                                                {rolePermissions.length > 0 ? rolePermissions.length : (selectedRole.permissions?.length || 0)}
                                            </p>
                                        </div>
                                        <div>
                                            <p className="text-sm text-gray-600 dark:text-gray-400">Total Available</p>
                                            <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{permissions.length}</p>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )
                    )}

                    {/* Permissions List Card */}
                    <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow">
                        <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
                            <div className="flex items-center justify-between">
                                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                                    {multiSelectMode ? 'Select Permissions to Add' : 'Available Permissions'}
                                </h3>
                                {!selectedRole && !multiSelectMode && (
                                    <p className="text-sm text-gray-600 dark:text-gray-400">Select a role to manage</p>
                                )}
                                {multiSelectMode && selectedRoles.size === 0 && (
                                    <p className="text-sm text-gray-600 dark:text-gray-400">Select roles to continue</p>
                                )}
                            </div>
                        </div>

                        {/* Bulk Actions */}
                        {canManageAdmin && selectedPermissions.size > 0 && (
                            <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4 bg-blue-50 dark:bg-blue-900/20">
                                <div className="flex items-center justify-between">
                                    <span className="text-sm text-blue-700 dark:text-blue-400">
                                        {selectedPermissions.size} permission{selectedPermissions.size !== 1 ? 's' : ''} selected
                                        {multiSelectMode && selectedRoles.size > 0 && ` • ${selectedRoles.size} role${selectedRoles.size !== 1 ? 's' : ''} selected`}
                                    </span>
                                    <div className="flex space-x-2">
                                        {multiSelectMode && selectedRoles.size > 0 ? (
                                            <>
                                                <button
                                                    onClick={handleBulkAddToMultipleRoles}
                                                    disabled={bulkOperationLoading}
                                                    className="px-3 py-1 bg-green-600 text-white text-sm rounded-md hover:bg-green-700 disabled:opacity-50"
                                                >
                                                    {bulkOperationLoading ? 'Adding...' : 'Add to Selected Roles'}
                                                </button>
                                                <button
                                                    onClick={() => setSelectedRoles(new Set())}
                                                    className="px-3 py-1 bg-gray-600 text-white text-sm rounded-md hover:bg-gray-700"
                                                >
                                                    Clear Roles
                                                </button>
                                            </>
                                        ) : (
                                            selectedRole && (
                                                <>
                                                    <button
                                                        onClick={handleBulkAddPermissions}
                                                        disabled={bulkOperationLoading}
                                                        className="px-3 py-1 bg-green-600 text-white text-sm rounded-md hover:bg-green-700 disabled:opacity-50"
                                                    >
                                                        {bulkOperationLoading ? 'Adding...' : 'Add Selected'}
                                                    </button>
                                                    <button
                                                        onClick={handleBulkRemovePermissions}
                                                        disabled={bulkOperationLoading}
                                                        className="px-3 py-1 bg-red-600 text-white text-sm rounded-md hover:bg-red-700 disabled:opacity-50"
                                                    >
                                                        {bulkOperationLoading ? 'Removing...' : 'Remove Selected'}
                                                    </button>
                                                    <button
                                                        onClick={() => setSelectedPermissions(new Set())}
                                                        className="px-3 py-1 bg-gray-600 text-white text-sm rounded-md hover:bg-gray-700"
                                                    >
                                                        Deselect All
                                                    </button>
                                                </>
                                            )
                                        )}
                                    </div>
                                </div>
                            </div>
                        )}

                        {/* Permissions Tabs */}
                        <div className="px-6 py-4">
                            {permissionsLoading ? (
                                <div className="text-center py-6 text-gray-500 dark:text-gray-400">Loading permissions...</div>
                            ) : (
                                <PermissionCategoryTabs
                                    permissions={permissions}
                                    assignedPermissionIds={assignedPermissionIds}
                                    selectedPermissions={selectedPermissions}
                                    onPermissionToggle={handlePermissionToggle}
                                    onSelectAll={handleSelectAll}
                                    onDeselectAll={handleDeselectAll}
                                    loading={addingPermission || removingPermission}
                                    readOnly={!canManageAdmin}
                                />
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default function WrappedPermissionManagementPage() {
    return (
        <PermissionPageErrorBoundary>
            <PermissionManagementPage />
        </PermissionPageErrorBoundary>
    )
}
