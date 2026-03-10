import { UserRoleWithPermissions } from '@/types';
import React, { useEffect, useState } from 'react';

interface TenantRoleSelectorProps {
    rolesByTenant?: Record<string, UserRoleWithPermissions[]>
    tenants?: Array<{ id: string; name: string }>
    onSelectionChange?: (tenantId: string, roleId: string, roleName: string) => void
}

const TenantRoleSelector: React.FC<TenantRoleSelectorProps> = ({ rolesByTenant = {}, tenants = [], onSelectionChange }) => {
    const [selectedTenantId, setSelectedTenantId] = useState<string>('')
    const [selectedRoleId, setSelectedRoleId] = useState<string>('')

    // Get tenant list from rolesByTenant keys, merged with passed tenants
    const userTenants = React.useMemo(() => {
        const uniqueTenants = new Map<string, { id: string; name: string }>()

        // Add from rolesByTenant keys
        Object.keys(rolesByTenant).forEach((tenantId) => {
            const tenant = tenants.find((t) => t.id === tenantId)
            uniqueTenants.set(tenantId, {
                id: tenantId,
                name: tenant?.name || `Tenant (${tenantId.slice(0, 8)}...)`,
            })
        })

        return Array.from(uniqueTenants.values())
    }, [rolesByTenant, tenants])

    // Get roles for selected tenant from rolesByTenant
    const tenantRoles = React.useMemo(() => {
        if (!selectedTenantId || !rolesByTenant[selectedTenantId]) return []
        return rolesByTenant[selectedTenantId] || []
    }, [selectedTenantId, rolesByTenant])

    // Initialize with first tenant if not set
    useEffect(() => {
        if (!selectedTenantId && userTenants.length > 0) {
            setSelectedTenantId(userTenants[0].id)
        }
    }, [userTenants, selectedTenantId])

    // Initialize with first role when tenant changes
    useEffect(() => {
        if (!selectedRoleId && tenantRoles.length > 0) {
            setSelectedRoleId(tenantRoles[0].id)
        }
    }, [tenantRoles, selectedRoleId])

    const handleTenantChange = (tenantId: string) => {
        setSelectedTenantId(tenantId)
        setSelectedRoleId('') // Reset role when tenant changes
    }

    const handleRoleChange = (roleId: string) => {
        setSelectedRoleId(roleId)
        const role = tenantRoles.find((r) => r.id === roleId)
        if (onSelectionChange && role) {
            onSelectionChange(selectedTenantId, roleId, role.name)
        }
    }

    const currentTenant = userTenants.find((t) => t.id === selectedTenantId)
    const currentRole = tenantRoles.find((r) => r.id === selectedRoleId)

    if (userTenants.length === 0) {
        return null
    }

    return (
        <div className="card mb-8">
            <div className="card-header">
                <h3 className="text-lg font-semibold text-foreground">Active Context</h3>
            </div>
            <div className="card-body space-y-4">
                <div>
                    <label className="block text-sm font-medium text-muted-foreground mb-2">Select Tenant</label>
                    <select
                        value={selectedTenantId}
                        onChange={(e) => handleTenantChange(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-slate-800 text-foreground"
                    >
                        {userTenants.map((tenant) => (
                            <option key={tenant.id} value={tenant.id}>
                                {tenant.name}
                            </option>
                        ))}
                    </select>
                </div>

                {tenantRoles.length > 0 && (
                    <div>
                        <label className="block text-sm font-medium text-muted-foreground mb-2">Select Role</label>
                        <select
                            value={selectedRoleId}
                            onChange={(e) => handleRoleChange(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-slate-800 text-foreground"
                        >
                            <option value="">-- Select a role --</option>
                            {tenantRoles.map((role) => (
                                <option key={role.id} value={role.id}>
                                    {role.name}
                                </option>
                            ))}
                        </select>
                    </div>
                )}

                <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md">
                    <p className="text-sm text-blue-900 dark:text-blue-200">
                        <strong>Current Context:</strong> {currentTenant?.name || 'No tenant selected'}
                        {currentRole && ` • ${currentRole.name}`}
                    </p>
                </div>
            </div>
        </div>
    )
}

export default TenantRoleSelector
