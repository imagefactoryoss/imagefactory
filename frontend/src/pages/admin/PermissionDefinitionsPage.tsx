import { useCanManageAdmin } from '@/hooks/useAccess'
import { usePermissions } from '@/hooks/usePermissions'
import { adminService } from '@/services/adminService'
import { Permission } from '@/types'
import { MagnifyingGlassIcon, PencilIcon, PlusIcon, ShieldCheckIcon, TrashIcon } from '@heroicons/react/24/outline'
import React, { useMemo, useState } from 'react'
import toast from 'react-hot-toast'

const PermissionDefinitionsPage: React.FC = () => {
    const canManageAdmin = useCanManageAdmin()
    const { permissions, loading, refetch } = usePermissions()
    const [searchQuery, setSearchQuery] = useState('')
    const [resourceFilter, setResourceFilter] = useState<string>('')
    const [showCreateModal, setShowCreateModal] = useState(false)
    const [showEditModal, setShowEditModal] = useState(false)
    const [editingPermission, setEditingPermission] = useState<Permission | null>(null)
    const [formData, setFormData] = useState({
        resource: '',
        action: '',
        description: '',
        category: ''
    })
    const [submitting, setSubmitting] = useState(false)

    // Get unique resources from permissions
    const resources = useMemo(() => {
        return Array.from(new Set(permissions.map((p) => p.resource))).sort()
    }, [permissions])

    // Filter permissions based on search and resource filter
    const filteredPermissions = useMemo(() => {
        return permissions.filter((p) => {
            const matchesSearch =
                p.resource.toLowerCase().includes(searchQuery.toLowerCase()) ||
                p.action.toLowerCase().includes(searchQuery.toLowerCase()) ||
                (p.description && p.description.toLowerCase().includes(searchQuery.toLowerCase())) ||
                (p.category && p.category.toLowerCase().includes(searchQuery.toLowerCase()))

            const matchesResource = !resourceFilter || p.resource === resourceFilter

            return matchesSearch && matchesResource
        })
    }, [permissions, searchQuery, resourceFilter])

    // Group permissions by resource
    const groupedPermissions = useMemo(() => {
        const groups: Record<string, Permission[]> = {}
        filteredPermissions.forEach(permission => {
            if (!groups[permission.resource]) {
                groups[permission.resource] = []
            }
            groups[permission.resource].push(permission)
        })
        return groups
    }, [filteredPermissions])

    const handleCreate = async () => {
        if (!formData.resource || !formData.action) {
            toast.error('Resource and action are required')
            return
        }

        // Check for duplicate permissions
        const existingPermission = permissions.find(
            p => p.resource === formData.resource && p.action === formData.action
        )

        if (existingPermission) {
            toast.error(`Permission "${formData.resource}:${formData.action}" already exists`)
            return
        }

        try {
            setSubmitting(true)
            await adminService.createPermission({
                resource: formData.resource,
                action: formData.action,
                description: formData.description || undefined,
                category: formData.category || undefined,
            })
            toast.success('Permission created successfully')
            setShowCreateModal(false)
            setFormData({ resource: '', action: '', description: '', category: '' })
            refetch(true) // Force refresh to get updated permissions
        } catch (error: any) {
            toast.error(error.message || 'Failed to create permission')
        } finally {
            setSubmitting(false)
        }
    }

    const handleEdit = async () => {
        if (!editingPermission) return

        try {
            setSubmitting(true)
            await adminService.updatePermission(editingPermission.id, {
                description: formData.description || undefined,
                category: formData.category || undefined,
            })
            toast.success('Permission updated successfully')
            setShowEditModal(false)
            setEditingPermission(null)
            setFormData({ resource: '', action: '', description: '', category: '' })
            refetch(true) // Force refresh to get updated permissions
        } catch (error: any) {
            toast.error(error.message || 'Failed to update permission')
        } finally {
            setSubmitting(false)
        }
    }

    const handleDelete = async (permission: Permission) => {
        if (permission.isSystemPermission) {
            toast.error('Cannot delete system permissions')
            return
        }

        if (!confirm(`Are you sure you want to delete the permission "${permission.resource}:${permission.action}"?`)) {
            return
        }

        try {
            await adminService.deletePermission(permission.id)
            toast.success('Permission deleted successfully')
            refetch(true) // Force refresh to get updated permissions
        } catch (error: any) {
            toast.error(error.message || 'Failed to delete permission')
        }
    }

    const openEditModal = (permission: Permission) => {
        if (permission.isSystemPermission) {
            toast.error('Cannot edit system permissions')
            return
        }

        setEditingPermission(permission)
        setFormData({
            resource: permission.resource,
            action: permission.action,
            description: permission.description || '',
            category: permission.category || ''
        })
        setShowEditModal(true)
    }

    const openCreateModalForResource = (resource?: string) => {
        setFormData({
            resource: resource || '',
            action: '',
            description: '',
            category: ''
        })
        setShowCreateModal(true)
    }

    const resetForm = () => {
        setFormData({ resource: '', action: '', description: '', category: '' })
        setEditingPermission(null)
        setShowCreateModal(false)
        setShowEditModal(false)
    }

    // Calculate statistics
    const stats = useMemo(() => {
        const totalPermissions = permissions.length
        const totalResources = resources.length
        const systemPermissions = permissions.filter(p => p.isSystemPermission).length
        const userPermissions = totalPermissions - systemPermissions

        return {
            totalPermissions,
            totalResources,
            systemPermissions,
            userPermissions
        }
    }, [permissions, resources])

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 dark:border-blue-400"></div>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Permission Definitions</h1>
                    <p className="text-gray-600 dark:text-gray-300">Manage system permissions and create new ones</p>
                </div>
                {canManageAdmin && (
                    <button
                        onClick={() => openCreateModalForResource()}
                        className="bg-blue-600 hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2"
                    >
                        <PlusIcon className="h-5 w-5" />
                        Create Permission
                    </button>
                )}
            </div>

            {!canManageAdmin && (
                <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: permission create, edit, and delete actions are hidden for System Administrator Viewer.
                </div>
            )}

            {/* Statistics Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <div className="bg-white dark:bg-slate-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-slate-700">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <ShieldCheckIcon className="h-8 w-8 text-blue-600 dark:text-blue-400" />
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-gray-600 dark:text-gray-300">Total Permissions</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats.totalPermissions}</p>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-slate-700">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="h-8 w-8 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                                <span className="text-green-600 dark:text-green-400 font-bold text-sm">R</span>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-gray-600 dark:text-gray-300">Resources</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats.totalResources}</p>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-slate-700">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <ShieldCheckIcon className="h-8 w-8 text-red-600 dark:text-red-400" />
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-gray-600 dark:text-gray-300">System Permissions</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats.systemPermissions}</p>
                        </div>
                    </div>
                </div>

                <div className="bg-white dark:bg-slate-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-slate-700">
                    <div className="flex items-center">
                        <div className="flex-shrink-0">
                            <div className="h-8 w-8 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
                                <span className="text-purple-600 dark:text-purple-400 font-bold text-sm">U</span>
                            </div>
                        </div>
                        <div className="ml-4">
                            <p className="text-sm font-medium text-gray-600 dark:text-gray-300">User Permissions</p>
                            <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats.userPermissions}</p>
                        </div>
                    </div>
                </div>
            </div>

            {/* Filters */}
            <div className="bg-white dark:bg-slate-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-slate-700">
                <div className="flex gap-4">
                    <div className="flex-1">
                        <div className="relative">
                            <MagnifyingGlassIcon className="h-5 w-5 absolute left-3 top-3 text-gray-400 dark:text-gray-500" />
                            <input
                                type="text"
                                placeholder="Search permissions..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                            />
                        </div>
                    </div>
                    <div className="w-64">
                        <select
                            value={resourceFilter}
                            onChange={(e) => setResourceFilter(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                        >
                            <option value="">All Resources</option>
                            {resources.map((resource) => (
                                <option key={resource} value={resource}>{resource}</option>
                            ))}
                        </select>
                    </div>
                </div>
            </div>

            {/* Permissions List */}
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-gray-200 dark:border-slate-700">
                {Object.keys(groupedPermissions).length === 0 ? (
                    <div className="p-8 text-center text-gray-500 dark:text-gray-400">
                        No permissions found
                    </div>
                ) : (
                    <div className="divide-y divide-gray-200 dark:divide-slate-700">
                        {Object.entries(groupedPermissions).map(([resource, perms]) => (
                            <div key={resource} className="p-6">
                                <div className="flex items-center justify-between mb-4">
                                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{resource}</h3>
                                    {canManageAdmin && (
                                        <button
                                            onClick={() => openCreateModalForResource(resource)}
                                            className="inline-flex items-center gap-1 px-2 py-1 text-sm bg-blue-100 hover:bg-blue-200 dark:bg-blue-900/30 dark:hover:bg-blue-800/50 text-blue-700 dark:text-blue-300 rounded-md transition-colors border border-blue-200 dark:border-blue-700"
                                            title={`Add permission for ${resource}`}
                                        >
                                            <PlusIcon className="h-4 w-4" />
                                            Add
                                        </button>
                                    )}
                                </div>
                                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                                    {perms.map((permission) => (
                                        <div key={permission.id} className="border border-gray-200 dark:border-slate-600 rounded-lg p-4 bg-gray-50 dark:bg-slate-700">
                                            <div className="flex items-start justify-between">
                                                <div className="flex-1">
                                                    <div className="flex items-center gap-2">
                                                        <span className="font-medium text-gray-900 dark:text-white">
                                                            {permission.action}
                                                        </span>
                                                        {permission.isSystemPermission && (
                                                            <ShieldCheckIcon className="h-4 w-4 text-green-600 dark:text-green-400" title="System Permission" />
                                                        )}
                                                    </div>
                                                    {permission.description && (
                                                        <p className="text-sm text-gray-600 dark:text-gray-300 mt-1">{permission.description}</p>
                                                    )}
                                                    {permission.category && (
                                                        <span className="inline-block mt-2 px-2 py-1 text-xs bg-gray-100 dark:bg-slate-600 text-gray-800 dark:text-gray-200 rounded">
                                                            {permission.category}
                                                        </span>
                                                    )}
                                                </div>
                                                {canManageAdmin && !permission.isSystemPermission && (
                                                    <div className="flex gap-2 ml-4">
                                                        <button
                                                            onClick={() => openEditModal(permission)}
                                                            className="text-blue-600 hover:text-blue-800 p-1"
                                                            title="Edit"
                                                        >
                                                            <PencilIcon className="h-4 w-4" />
                                                        </button>
                                                        <button
                                                            onClick={() => handleDelete(permission)}
                                                            className="text-red-600 hover:text-red-800 p-1"
                                                            title="Delete"
                                                        >
                                                            <TrashIcon className="h-4 w-4" />
                                                        </button>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Create Modal */}
            {showCreateModal && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-slate-800 rounded-lg p-6 w-full max-w-md border border-gray-200 dark:border-slate-700">
                        <h2 className="text-xl font-bold mb-4 text-gray-900 dark:text-white">Create Permission</h2>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Resource</label>
                                <input
                                    type="text"
                                    value={formData.resource}
                                    onChange={(e) => setFormData({ ...formData, resource: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                                    placeholder="e.g., users, roles, permissions"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Action</label>
                                <input
                                    type="text"
                                    value={formData.action}
                                    onChange={(e) => setFormData({ ...formData, action: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                                    placeholder="e.g., create, read, update, delete"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                                <input
                                    type="text"
                                    value={formData.description}
                                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                                    placeholder="Optional description"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Category</label>
                                <input
                                    type="text"
                                    value={formData.category}
                                    onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                                    placeholder="Optional category"
                                />
                            </div>
                        </div>
                        <div className="flex gap-3 mt-6">
                            <button
                                onClick={handleCreate}
                                disabled={submitting}
                                className="flex-1 bg-blue-600 hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-700 text-white py-2 rounded-lg disabled:opacity-50"
                            >
                                {submitting ? 'Creating...' : 'Create'}
                            </button>
                            <button
                                onClick={resetForm}
                                className="flex-1 bg-gray-300 hover:bg-gray-400 dark:bg-slate-600 dark:hover:bg-slate-500 text-gray-700 dark:text-gray-200 py-2 rounded-lg"
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Edit Modal */}
            {showEditModal && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-slate-800 rounded-lg p-6 w-full max-w-md border border-gray-200 dark:border-slate-700">
                        <h2 className="text-xl font-bold mb-4 text-gray-900 dark:text-white">Edit Permission</h2>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Resource</label>
                                <input
                                    type="text"
                                    value={formData.resource}
                                    disabled
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-gray-100 dark:bg-slate-600 text-gray-500 dark:text-gray-400"
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Resource cannot be changed</p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Action</label>
                                <input
                                    type="text"
                                    value={formData.action}
                                    disabled
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-gray-100 dark:bg-slate-600 text-gray-500 dark:text-gray-400"
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Action cannot be changed</p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                                <input
                                    type="text"
                                    value={formData.description}
                                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Category</label>
                                <input
                                    type="text"
                                    value={formData.category}
                                    onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent"
                                />
                            </div>
                        </div>
                        <div className="flex gap-3 mt-6">
                            <button
                                onClick={handleEdit}
                                disabled={submitting}
                                className="flex-1 bg-blue-600 hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-700 text-white py-2 rounded-lg disabled:opacity-50"
                            >
                                {submitting ? 'Updating...' : 'Update'}
                            </button>
                            <button
                                onClick={resetForm}
                                className="flex-1 bg-gray-300 hover:bg-gray-400 dark:bg-slate-600 dark:hover:bg-slate-500 text-gray-700 dark:text-gray-200 py-2 rounded-lg"
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

export default PermissionDefinitionsPage
