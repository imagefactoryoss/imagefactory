import { Permission } from '@/types'
import { CheckIcon, MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import React, { useMemo, useState } from 'react'

interface PermissionCategoryTabsProps {
    permissions: Permission[]
    assignedPermissionIds: Set<string>
    onPermissionToggle: (permissionId: string, selected: boolean) => void
    onSelectAll?: () => void
    onDeselectAll?: () => void
    loading?: boolean
    selectedPermissions?: Set<string>
    readOnly?: boolean
}

export const PermissionCategoryTabs: React.FC<PermissionCategoryTabsProps> = ({
    permissions,
    assignedPermissionIds,
    onPermissionToggle,
    onSelectAll,
    onDeselectAll,
    loading = false,
    selectedPermissions = new Set(),
    readOnly = false,
}) => {
    const [activeTab, setActiveTab] = useState<string>('')
    const [searchQuery, setSearchQuery] = useState('')

    // Get unique resources and sort them
    const resources = useMemo(() => {
        const resourceSet = new Set(permissions.map((p) => p.resource))
        const sorted = Array.from(resourceSet).sort()
        return sorted
    }, [permissions])

    // Set first resource as active tab on mount
    React.useEffect(() => {
        if (resources.length > 0 && !activeTab) {
            setActiveTab(resources[0])
        }
    }, [resources, activeTab])

    // Get permissions for active tab
    const tabPermissions = useMemo(() => {
        return permissions.filter((p) => p.resource === activeTab)
    }, [permissions, activeTab])

    // Filter permissions by search query
    const filteredPermissions = useMemo(() => {
        return tabPermissions.filter((p) => {
            const query = searchQuery.toLowerCase()
            return (
                p.resource.toLowerCase().includes(query) ||
                p.action.toLowerCase().includes(query) ||
                (p.description && p.description.toLowerCase().includes(query))
            )
        })
    }, [tabPermissions, searchQuery])

    // Count selected permissions in current tab
    const selectedInTab = useMemo(() => {
        return filteredPermissions.filter((p) => selectedPermissions.has(p.id)).length
    }, [filteredPermissions, selectedPermissions])

    // Handle select all in current tab
    const handleSelectTabAll = () => {
        filteredPermissions.forEach((p) => {
            if (!selectedPermissions.has(p.id)) {
                onPermissionToggle(p.id, true)
            }
        })
    }

    // Handle deselect all in current tab
    const handleDeselectTabAll = () => {
        filteredPermissions.forEach((p) => {
            if (selectedPermissions.has(p.id)) {
                onPermissionToggle(p.id, false)
            }
        })
    }

    if (resources.length === 0) {
        return (
            <div className="p-4 text-center text-gray-500 dark:text-gray-400">
                No permissions available
            </div>
        )
    }

    return (
        <div className="space-y-4">
            {/* Search Box */}
            <div className="relative">
                <MagnifyingGlassIcon className="absolute left-3 top-3 h-5 w-5 text-gray-400 dark:text-gray-500" />
                <input
                    type="text"
                    placeholder="Search permissions..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
                />
            </div>

            {/* Tabs */}
            <div className="border-b border-gray-200 dark:border-gray-700">
                <div className="flex flex-wrap gap-1">
                    {resources.map((resource) => {
                        const resourcePerms = permissions.filter((p) => p.resource === resource)
                        const selectedCount = resourcePerms.filter((p) => selectedPermissions.has(p.id)).length
                        const isActive = activeTab === resource

                        return (
                            <button
                                key={resource}
                                type="button"
                                onClick={() => {
                                    setActiveTab(resource)
                                    setSearchQuery('')
                                }}
                                className={`px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors ${isActive
                                    ? 'border-b-2 border-blue-500 text-blue-600 dark:text-blue-400'
                                    : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
                                    }`}
                            >
                                <span className="capitalize">{resource}</span>
                                {selectedCount > 0 && (
                                    <span className="ml-2 inline-block bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 text-xs font-semibold px-2 py-1 rounded-full">
                                        {selectedCount}
                                    </span>
                                )}
                            </button>
                        )
                    })}
                </div>
            </div>

            {/* Tab Content */}
            <div>
                {/* Tab Controls */}
                <div className="flex items-center justify-between py-2 px-3 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
                    <div className="text-sm text-gray-600 dark:text-gray-400">
                        {filteredPermissions.length} permissions
                        {selectedInTab > 0 && ` • ${selectedInTab} selected`}
                    </div>
                    {!readOnly && (
                        <div className="flex gap-2">
                            <button
                                type="button"
                                onClick={handleSelectTabAll}
                                disabled={loading}
                                className="text-sm px-3 py-1 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900 rounded disabled:opacity-50"
                            >
                                Select All
                            </button>
                            <button
                                type="button"
                                onClick={handleDeselectTabAll}
                                disabled={loading}
                                className="text-sm px-3 py-1 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded disabled:opacity-50"
                            >
                                Deselect
                            </button>
                        </div>
                    )}
                </div>

                {/* Permissions List */}
                <div className="space-y-2 max-h-96 overflow-y-auto py-3 px-3">
                    {filteredPermissions.length === 0 ? (
                        <div className="text-center text-gray-400 dark:text-gray-500 py-8">
                            No permissions match your search
                        </div>
                    ) : (
                        filteredPermissions.map((permission) => {
                            const isAssigned = assignedPermissionIds.has(permission.id)
                            const isSelected = selectedPermissions.has(permission.id)

                            return (
                                <div
                                    key={permission.id}
                                    className={`flex items-start gap-3 p-3 rounded-lg border transition-colors ${isSelected
                                        ? 'bg-blue-50 dark:bg-blue-900 border-blue-200 dark:border-blue-700'
                                        : isAssigned
                                            ? 'bg-green-50 dark:bg-green-900 border-green-200 dark:border-green-700'
                                            : 'border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800'
                                        }`}
                                >
                                    {!readOnly && (
                                        <input
                                            type="checkbox"
                                            checked={isSelected}
                                            onChange={(e) => onPermissionToggle(permission.id, e.target.checked)}
                                            disabled={loading}
                                            className="mt-1 w-4 h-4 text-blue-600 dark:text-blue-400 rounded focus:ring-blue-500 dark:focus:ring-blue-400 dark:border-gray-600 disabled:opacity-50 cursor-pointer"
                                        />
                                    )}
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2">
                                            <code className="text-sm font-mono text-gray-900 dark:text-gray-100">
                                                {permission.resource}:{permission.action}
                                            </code>
                                            {isAssigned && !isSelected && (
                                                <span className="inline-flex items-center gap-1 text-xs bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 px-2 py-1 rounded">
                                                    <CheckIcon className="h-3 w-3" />
                                                    Assigned
                                                </span>
                                            )}
                                        </div>
                                        {permission.description && (
                                            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{permission.description}</p>
                                        )}
                                        {permission.category && (
                                            <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">Category: {permission.category}</p>
                                        )}
                                    </div>
                                </div>
                            )
                        })
                    )}
                </div>
            </div>
        </div>
    )
}
