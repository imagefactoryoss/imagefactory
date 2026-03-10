import Drawer from '@/components/ui/Drawer'
import { NIL_TENANT_ID } from '@/constants/tenant'
import { useRefresh } from '@/context/RefreshContext'
import { useCanManageAdmin, useIsSystemAdmin } from '@/hooks/useAccess'
import { adminService } from '@/services/adminService'
import { Permission, Tenant, UserManagementFilters, UserRoleWithPermissions, UserWithRoles } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

// User Details Drawer Component
interface UserDetailsDrawerProps {
    isOpen: boolean
    user: UserWithRoles | null
    roles: UserRoleWithPermissions[]
    tenants: Tenant[]
    onClose: () => void
    onSubmit: (data: any) => Promise<void>
    isLoading?: boolean
    isCreating?: boolean
    getTenantName?: (tenantId: string) => string
}

const UserDetailsDrawer: React.FC<UserDetailsDrawerProps> = ({ isOpen, user, roles, tenants, onClose, onSubmit, isLoading, isCreating, getTenantName: getTenantNameProp }) => {
    const isSystemAdmin = useIsSystemAdmin()
    const [tenantNames, setTenantNames] = useState<Record<string, string>>({})
    const [tenantRoles, setTenantRoles] = useState<Record<string, UserRoleWithPermissions[]>>({})
    const [systemTenantId, setSystemTenantId] = useState<string | null>(null)
    const [formData, setFormData] = useState({
        email: '',
        firstName: '',
        lastName: '',
        password: '',
        status: 'active',
        systemAccess: 'none' as string,
        tenantIds: [] as string[],
        roleAssignments: [] as Array<{ tenantId: string; roleId: string; role?: UserRoleWithPermissions }>,
    })
    const [errors, setErrors] = useState<Record<string, string>>({})
    const [editingTenant, setEditingTenant] = useState<{ tenantId: string; roleId?: string; tempTenantId?: string } | null>(null)

    // Load tenant names for tenant IDs in user's roles
    useEffect(() => {
        if (user?.rolesByTenant && Object.keys(user.rolesByTenant).length > 0) {
            const loadTenantNames = async () => {
                const tenantIds = Object.keys(user.rolesByTenant || {})
                const namesToLoad = tenantIds.filter(id =>
                    !tenantNames[id] &&
                    !tenants?.find(t => t.id === id) &&
                    id !== NIL_TENANT_ID
                )

                if (namesToLoad.length === 0) return

                // setLoadingTenantNames(true)
                try {
                    const namePromises = namesToLoad.map(async (tenantId) => {
                        try {
                            const tenant = await adminService.getTenantById(tenantId)
                            return { tenantId, name: tenant.name }
                        } catch (error) {
                            return { tenantId, name: `Tenant (${tenantId.slice(0, 8)}...)` }
                        }
                    })

                    const results = await Promise.all(namePromises)
                    const newNames: Record<string, string> = {}
                    results.forEach(({ tenantId, name }) => {
                        newNames[tenantId] = name
                    })

                    setTenantNames(prev => ({ ...prev, ...newNames }))
                } catch (error) {
                } finally {
                    // setLoadingTenantNames(false)
                }
            }

            loadTenantNames()
        }
    }, [user?.rolesByTenant, tenants, tenantNames])

    const getTenantDisplayName = (tenantId: string): string => {
        // Handle nil UUID
        if (tenantId === NIL_TENANT_ID) {
            return 'Unknown Tenant'
        }
        // First try the passed getTenantName function
        if (getTenantNameProp) {
            return getTenantNameProp(tenantId)
        }
        // Then try the pre-loaded tenants
        const tenant = tenants?.find(t => t.id === tenantId)
        if (tenant?.name) {
            return tenant.name
        }
        // Then try the dynamically loaded names
        if (tenantNames[tenantId]) {
            return tenantNames[tenantId]
        }
        // Finally fallback
        return `Tenant (${tenantId.slice(0, 8)}...)`
    }

    const normalizeRoleType = (roleName: string): string =>
        roleName.trim().toLowerCase().replace(/\s+/g, '_')
    const isSystemAccessRoleType = (roleName: string): boolean => {
        const normalized = normalizeRoleType(roleName)
        return normalized === 'system_administrator' ||
            normalized === 'system_administrator_viewer' ||
            normalized === 'security_reviewer'
    }
    const systemAccessRoleOptions = useMemo(() => {
        return Array.from(
            new Map(
                roles
                    .filter((role) => isSystemAccessRoleType(role.name))
                    .map((role) => [role.id, role])
            ).values()
        ).sort((a, b) => a.name.localeCompare(b.name))
    }, [roles])

    useEffect(() => {
        let cancelled = false

        const loadSystemTenant = async () => {
            if (!isSystemAdmin || tenants.length === 0) return

            for (const tenant of tenants) {
                try {
                    const groups = await adminService.getGroupsByTenant(tenant.id)
                    const hasSystemAccessGroup = (groups || []).some((group: any) =>
                        group?.role_type === 'system_administrator' ||
                        group?.role_type === 'system_administrator_viewer' ||
                        group?.role_type === 'security_reviewer'
                    )
                    if (hasSystemAccessGroup) {
                        if (!cancelled) setSystemTenantId(tenant.id)
                        return
                    }
                } catch {
                    // Ignore tenant group lookup errors and continue.
                }
            }
            if (!cancelled) setSystemTenantId(null)
        }

        loadSystemTenant()
        return () => {
            cancelled = true
        }
    }, [isSystemAdmin, tenants])

    const loadTenantRoles = async (tenantId: string) => {
        if (tenantRoles[tenantId]) return // Already loaded

        try {
            const [tenantScopedRoles, tenantGroups] = await Promise.all([
                adminService.getRolesByTenant(tenantId),
                adminService.getGroupsByTenant(tenantId).catch(() => []),
            ])

            const baseFilteredRoles = tenantScopedRoles.filter(
                role => role.name.toLowerCase() !== 'system administrator'
            )
            const allowedRoleTypes = new Set(
                (tenantGroups || [])
                    .filter((group: any) => (group?.status || '').toLowerCase() === 'active')
                    .map((group: any) => String(group?.role_type || '').toLowerCase())
                    .filter((roleType: string) => roleType.length > 0)
            )
            const filteredRoles = allowedRoleTypes.size > 0
                ? baseFilteredRoles.filter(role => allowedRoleTypes.has(normalizeRoleType(role.name)))
                : baseFilteredRoles

            const rolesByName = new Map<string, UserRoleWithPermissions>()
            for (const role of filteredRoles) {
                const nameKey = role.name.trim().toLowerCase()
                const existing = rolesByName.get(nameKey)
                if (!existing || (role.permissions?.length || 0) > (existing.permissions?.length || 0)) {
                    rolesByName.set(nameKey, role)
                }
            }
            const uniqueRoles = Array.from(rolesByName.values()).sort((a, b) =>
                a.name.localeCompare(b.name)
            )

            setTenantRoles(prev => ({
                ...prev,
                [tenantId]: uniqueRoles,
            }))
        } catch (error: any) {
            toast.error(error.message || 'Failed to load tenant roles')
            setTenantRoles(prev => ({
                ...prev,
                [tenantId]: [],
            }))
        }
    }

    useEffect(() => {
        if (user) {
            // Extract tenant IDs and role assignments from user's current assignments
            const tenantIds: string[] = []
            const roleAssignments: Array<{ tenantId: string; roleId: string; role?: UserRoleWithPermissions }> = []
            let detectedSystemAccess = 'none'

            // Use rolesByTenant if available (per-tenant assignments from API)
            if (user.rolesByTenant && Object.keys(user.rolesByTenant).length > 0) {
                Object.entries(user.rolesByTenant).forEach(([tenantId, assignedRoles]) => {
                    const systemRole = assignedRoles.find((role) =>
                        systemAccessRoleOptions.some((option) => option.id === role.id) ||
                        ['system administrator', 'system administrator viewer', 'security reviewer'].includes(role.name.toLowerCase())
                    )
                    if (systemRole) {
                        detectedSystemAccess = systemRole.id
                        if (!systemTenantId) {
                            setSystemTenantId(tenantId)
                        }
                        return
                    }

                    if (!tenantIds.includes(tenantId)) {
                        tenantIds.push(tenantId)
                    }
                    // Add role assignment for this tenant (use first role if multiple)
                    if (assignedRoles.length > 0) {
                        roleAssignments.push({
                            tenantId,
                            roleId: assignedRoles[0].id,
                            role: assignedRoles[0],
                        })
                    }
                })
            } else if (user.tenant?.id) {
                // Fallback: use primary tenant and roles
                tenantIds.push(user.tenant.id)
                if (user.roles.length > 0) {
                    roleAssignments.push({
                        tenantId: user.tenant.id,
                        roleId: user.roles[0].id,
                        role: user.roles[0],
                    })
                }
            }

            setFormData({
                email: user.email || '',
                firstName: user.name?.split(' ')[0] || '',
                lastName: user.name?.split(' ').slice(1).join(' ') || '',
                password: '',
                status: user.status || 'active',
                systemAccess: detectedSystemAccess,
                tenantIds,
                roleAssignments,
            })

            // Don't auto-select first tenant - let user choose what to do
        } else {
            setFormData({
                email: '',
                firstName: '',
                lastName: '',
                password: '',
                status: 'active',
                systemAccess: 'none',
                tenantIds: [],
                roleAssignments: [],
            })
        }
        setErrors({})
    }, [user, isOpen, systemTenantId, systemAccessRoleOptions])

    // Load roles for tenant when editing modal opens
    useEffect(() => {
        const tenantId = editingTenant?.tenantId || editingTenant?.tempTenantId
        if (tenantId) {
            loadTenantRoles(tenantId)
        }
    }, [editingTenant?.tenantId, editingTenant?.tempTenantId])

    const validateForm = () => {
        const newErrors: Record<string, string> = {}

        // Skip LDAP-managed field validation for LDAP users
        const isLDAPUser = user?.auth_method === 'ldap'

        if (!isLDAPUser) {
            if (!formData.email) newErrors.email = 'Email is required'
            else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) newErrors.email = 'Invalid email format'

            if (!formData.firstName) newErrors.firstName = 'First name is required'
            if (!formData.lastName) newErrors.lastName = 'Last name is required'
        }

        if (isCreating && !formData.password) newErrors.password = 'Password is required'
        if (formData.password && (formData.password.length < 8 || formData.password.length > 128)) {
            newErrors.password = 'Password must be between 8 and 128 characters'
        }

        const hasAnyAssignment = formData.tenantIds.length > 0 || formData.systemAccess !== 'none'
        if (isSystemAdmin && isCreating && !hasAnyAssignment) {
            newErrors.tenantIds = 'At least one tenant must be assigned when creating new users'
        }

        // Validate that each assigned tenant has a role
        if (formData.tenantIds.length > 0) {
            for (const tenantId of formData.tenantIds) {
                const hasRole = formData.roleAssignments.some(ra => ra.tenantId === tenantId && ra.roleId)
                if (!hasRole) {
                    newErrors[`role-${tenantId}`] = 'Role is required for this tenant'
                }
            }
        }

        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!validateForm()) return

        try {
            const isLDAPUser = user?.auth_method === 'ldap'

            // For LDAP users, exclude fields managed by LDAP directory
            const submitData: any = {
                ...formData,
                roleAssignments: (() => {
                    const baseAssignments = formData.roleAssignments.filter((assignment) =>
                        !systemTenantId || assignment.tenantId !== systemTenantId
                    )
                    if (!systemTenantId || formData.systemAccess === 'none') {
                        return baseAssignments
                    }
                    const roleId = formData.systemAccess
                    if (!roleId) {
                        return baseAssignments
                    }
                    return [
                        ...baseAssignments,
                        { tenantId: systemTenantId, roleId },
                    ]
                })(),
                status: formData.status,
                tenantIds: formData.tenantIds || undefined,
            }

            // Only submit firstName, lastName, email if NOT an LDAP user
            if (!isLDAPUser) {
                submitData.firstName = formData.firstName
                submitData.lastName = formData.lastName
                submitData.email = formData.email
            }

            // Only include password if provided and not an LDAP user
            if (!isLDAPUser && (isCreating || formData.password)) {
                submitData.password = formData.password
            }

            await onSubmit(submitData)
            onClose()
        } catch (error) {
        }
    }

    const assignableTenants = tenants.filter((tenant) => tenant.id !== systemTenantId)
    const unassignedTenants = assignableTenants.filter((tenant) => !formData.tenantIds.includes(tenant.id))

    return (
        <Drawer
            isOpen={isOpen}
            onClose={onClose}
            title={user ? `Edit User: ${user.name}` : 'Create New User'}
        >
            <form onSubmit={handleSubmit} className="space-y-4">
                {/* Warning for suspended users */}
                {user && user.status === 'suspended' && (
                    <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                        <p className="text-sm text-red-800 dark:text-red-200">
                            <strong>⚠️ User Suspended:</strong> This user account is currently suspended. To make edits, you must first reactivate the user by clicking the "Activate" button in the user list.
                        </p>
                    </div>
                )}

                {user && user.auth_method === 'ldap' && (
                    <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md">
                        <p className="text-sm text-blue-800 dark:text-blue-200">
                            <strong>ℹ️ LDAP User:</strong> This user's name and email are managed by your LDAP directory. These fields cannot be edited here.
                        </p>
                    </div>
                )}

                {/* Email */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Email
                    </label>
                    <input
                        type="email"
                        value={formData.email}
                        onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                        disabled={!user || user.status === 'suspended' || user?.auth_method === 'ldap'}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                        placeholder="user@example.com"
                    />
                    {errors.email && <p className="text-xs text-red-600 mt-1">{errors.email}</p>}
                </div>

                {/* First Name */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        First Name
                    </label>
                    <input
                        type="text"
                        value={formData.firstName}
                        onChange={(e) => setFormData({ ...formData, firstName: e.target.value })}
                        disabled={user?.status === 'suspended' || user?.auth_method === 'ldap'}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                        placeholder="John"
                    />
                    {errors.firstName && <p className="text-xs text-red-600 mt-1">{errors.firstName}</p>}
                </div>

                {/* Last Name */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Last Name
                    </label>
                    <input
                        type="text"
                        value={formData.lastName}
                        onChange={(e) => setFormData({ ...formData, lastName: e.target.value })}
                        disabled={user?.status === 'suspended' || user?.auth_method === 'ldap'}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                        placeholder="Doe"
                    />
                    {errors.lastName && <p className="text-xs text-red-600 mt-1">{errors.lastName}</p>}
                </div>

                {/* Password */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Password {user && '(leave blank to keep unchanged)'}
                    </label>
                    <input
                        type="password"
                        value={formData.password}
                        onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                        disabled={user?.status === 'suspended' || user?.auth_method === 'ldap'}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                        placeholder={user ? '(leave blank)' : 'At least 8 characters'}
                    />
                    {errors.password && <p className="text-xs text-red-600 mt-1">{errors.password}</p>}
                </div>

                {/* Status */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Status
                    </label>
                    <select
                        value={formData.status}
                        onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                        disabled={user?.status === 'suspended'}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        <option value="active">Active</option>
                        <option value="inactive">Inactive</option>
                        <option value="suspended">Suspended</option>
                    </select>
                </div>

                {/* Tenant & Role Assignments (System Admin Only) */}
                {isSystemAdmin && (
                    <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                        <div className="mb-4">
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                                System Access
                            </label>
                            <select
                                value={formData.systemAccess}
                                onChange={(e) => setFormData({
                                    ...formData,
                                    systemAccess: e.target.value,
                                })}
                                disabled={user?.status === 'suspended'}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                <option value="none">None</option>
                                {systemAccessRoleOptions.map((role) => (
                                    <option key={role.id} value={role.id}>{role.name}</option>
                                ))}
                            </select>
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                System access is managed separately from tenant role assignments.
                            </p>
                        </div>

                        <div className="flex items-center justify-between mb-3">
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
                                Tenant Assignments {!user && '(Required for new users)'}
                            </label>
                            <button
                                type="button"
                                onClick={() => setEditingTenant({ tenantId: '', roleId: undefined })}
                                disabled={user?.status === 'suspended'}
                                className="px-3 py-1 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded-md flex items-center gap-1 disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                <span>+</span>
                                <span>Add</span>
                            </button>
                        </div>

                        {/* Current Assignments */}
                        {formData.tenantIds.length > 0 && (
                            <div className="space-y-2 mb-4">
                                {formData.tenantIds.map((tenantId) => {
                                    const roleAssignment = formData.roleAssignments.find(ra => ra.tenantId === tenantId)
                                    const role = roleAssignment?.role || roles.find(r => r.id === roleAssignment?.roleId)

                                    return (
                                        <div
                                            key={tenantId}
                                            className="flex items-center justify-between p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md"
                                        >
                                            <div className="flex-1">
                                                <div className="font-medium text-slate-900 dark:text-white">
                                                    {getTenantDisplayName(tenantId)}
                                                </div>
                                                <div className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                                                    Role: <span className="font-medium">{role?.name || '(No role assigned)'}</span>
                                                </div>
                                            </div>
                                            <div className="flex gap-2 ml-3">
                                                <button
                                                    type="button"
                                                    onClick={() => {
                                                        const currentAssignment = formData.roleAssignments.find(ra => ra.tenantId === tenantId)
                                                        setEditingTenant({
                                                            tenantId,
                                                            roleId: currentAssignment?.roleId
                                                        })
                                                    }}
                                                    disabled={user?.status === 'suspended'}
                                                    className="px-2 py-1 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed"
                                                >
                                                    Edit
                                                </button>
                                                <button
                                                    type="button"
                                                    onClick={() => {
                                                        const newTenantIds = formData.tenantIds.filter(id => id !== tenantId)
                                                        const newAssignments = formData.roleAssignments.filter(ra => ra.tenantId !== tenantId)
                                                        setFormData({
                                                            ...formData,
                                                            tenantIds: newTenantIds,
                                                            roleAssignments: newAssignments,
                                                        })
                                                    }}
                                                    disabled={user?.status === 'suspended'}
                                                    className="px-2 py-1 text-xs bg-red-600 hover:bg-red-700 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed"
                                                >
                                                    Remove
                                                </button>
                                            </div>
                                        </div>
                                    )
                                })}
                            </div>
                        )}

                        {isCreating && formData.tenantIds.length === 0 && formData.systemAccess === 'none' && (
                            <p className="text-xs text-amber-600 dark:text-amber-400 mt-2">
                                At least one tenant must be assigned when creating new users
                            </p>
                        )}
                        {errors.tenantIds && <p className="text-xs text-red-600 mt-2">{errors.tenantIds}</p>}
                    </div>
                )}

                {/* Buttons */}
                <div className="flex gap-3 pt-6 border-t border-slate-200 dark:border-slate-700">
                    <button
                        type="button"
                        onClick={onClose}
                        className="flex-1 px-4 py-2 text-slate-900 dark:text-white border border-slate-300 dark:border-slate-600 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700"
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        disabled={isLoading || user?.status === 'suspended'}
                        className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {isLoading ? 'Saving...' : user ? 'Update User' : 'Create User'}
                    </button>
                </div>
            </form>

            {/* Tenant Assignment Edit Modal */}
            {editingTenant && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-slate-800 rounded-lg shadow-xl max-w-md w-full mx-4">
                        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-700">
                            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                                {editingTenant.tenantId ? 'Edit Tenant Assignment' : 'Add Tenant Assignment'}
                            </h3>
                            {editingTenant.tenantId && (
                                <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                                    {tenants.find(t => t.id === editingTenant.tenantId)?.name || 'Unknown Tenant'}
                                </p>
                            )}
                        </div>
                        <div className="px-6 py-4">
                            <div className="mb-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md">
                                <p className="text-sm text-blue-800 dark:text-blue-200">
                                    <strong>Add a new tenant assignment:</strong> Select a tenant and role below. Only tenants that aren't already assigned to this user will appear in the dropdown. To change roles for existing assignments, use the "Edit" button next to each tenant assignment above.
                                </p>
                            </div>
                            <div className="space-y-4">
                                {/* Tenant Selection - Only for adding new assignments */}
                                {!editingTenant.tenantId && (
                                    <div>
                                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                            Tenant
                                        </label>
                                        {unassignedTenants.length === 0 ? (
                                            <div className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-slate-50 dark:bg-slate-700 text-slate-500 dark:text-slate-400">
                                                All available tenants are already assigned to this user. Use the "Edit" buttons above to modify existing assignments.
                                            </div>
                                        ) : (
                                            <select
                                                value={editingTenant.tempTenantId || ''}
                                                onChange={(e) => {
                                                    const tenantId = e.target.value
                                                    setEditingTenant({ ...editingTenant, tempTenantId: tenantId, roleId: undefined })
                                                }}
                                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                                            >
                                                <option value="">-- Select a tenant --</option>
                                                {unassignedTenants
                                                    .map((tenant) => (
                                                        <option key={tenant.id} value={tenant.id}>
                                                            {tenant.name}
                                                        </option>
                                                    ))}
                                            </select>
                                        )}
                                    </div>
                                )}

                                {/* Role Selection */}
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                        Role
                                    </label>
                                    {(!editingTenant.tenantId && !editingTenant.tempTenantId) ? (
                                        <div className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-slate-50 dark:bg-slate-700 text-slate-500 dark:text-slate-400">
                                            {unassignedTenants.length === 0
                                                ? 'No additional tenants available to assign'
                                                : 'Select a tenant above first'
                                            }
                                        </div>
                                    ) : (() => {
                                        const tenantId = editingTenant.tenantId || editingTenant.tempTenantId
                                        if (!tenantId || !tenantRoles[tenantId]) {
                                            return (
                                                <div className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-slate-50 dark:bg-slate-700 text-slate-500 dark:text-slate-400 flex items-center">
                                                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-slate-500 dark:border-slate-400 mr-2"></div>
                                                    Loading roles...
                                                </div>
                                            )
                                        }
                                        return (
                                            <select
                                                value={editingTenant.roleId || ''}
                                                onChange={(e) => {
                                                    setEditingTenant({ ...editingTenant, roleId: e.target.value })
                                                }}
                                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                                            >
                                                <option value="">-- Select a role --</option>
                                                {(tenantRoles[tenantId] || [])
                                                    .filter((role: any) => {
                                                        // Only allow System Administrator role for System Administrators tenant
                                                        const selectedTenant = tenants.find(t => t.id === tenantId)
                                                        if (role.name === 'System Administrator') {
                                                            return selectedTenant?.name === 'System Administrators'
                                                        }
                                                        return true
                                                    })
                                                    .map((role: any) => (
                                                        <option key={role.id} value={role.id}>
                                                            {role.name}
                                                        </option>
                                                    ))}
                                            </select>
                                        )
                                    })()}
                                </div>
                            </div>
                        </div>
                        <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex justify-end gap-3">
                            <button
                                type="button"
                                onClick={() => setEditingTenant(null)}
                                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-700 border border-slate-300 dark:border-slate-600 rounded-md hover:bg-slate-50 dark:hover:bg-slate-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                            >
                                Cancel
                            </button>
                            <button
                                type="button"
                                disabled={
                                    (() => {
                                        const tenantId = editingTenant.tenantId || editingTenant.tempTenantId
                                        if (!tenantId) return true
                                        return !(tenantRoles[tenantId] || []).length ||
                                            !editingTenant.roleId ||
                                            (!editingTenant.tenantId && unassignedTenants.length === 0)
                                    })()
                                }
                                onClick={() => {
                                    const tenantId = editingTenant.tenantId || editingTenant.tempTenantId
                                    if (!tenantId || !editingTenant.roleId) return

                                    if (editingTenant.tenantId) {
                                        // Editing existing assignment
                                        const existingIndex = formData.roleAssignments.findIndex(ra => ra.tenantId === tenantId)
                                        const role = tenantRoles[tenantId]?.find((r: any) => r.id === editingTenant.roleId)
                                        const updatedAssignments = [...formData.roleAssignments]
                                        if (existingIndex >= 0) {
                                            updatedAssignments[existingIndex] = { tenantId: tenantId!, roleId: editingTenant.roleId!, role }
                                        } else {
                                            updatedAssignments.push({ tenantId: tenantId!, roleId: editingTenant.roleId!, role })
                                        }
                                        const updatedFormData = { ...formData, roleAssignments: updatedAssignments }

                                        // For existing users, just update form data and close modal (don't auto-save)
                                        setFormData(updatedFormData)
                                        setEditingTenant(null)
                                    } else {
                                        // Adding new assignment
                                        const role = tenantRoles[tenantId]?.find(r => r.id === editingTenant.roleId)
                                        const updatedFormData = {
                                            ...formData,
                                            tenantIds: [...formData.tenantIds, tenantId],
                                            roleAssignments: [...formData.roleAssignments, { tenantId, roleId: editingTenant.roleId, role }]
                                        }

                                        // For existing users, just update form data and close modal (don't auto-save)
                                        setFormData(updatedFormData)
                                        setEditingTenant(null)
                                    }
                                }}
                                className={`px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 ${(() => {
                                    const tenantId = editingTenant.tenantId || editingTenant.tempTenantId
                                    if (!tenantId) return true
                                    return !(tenantRoles[tenantId] || []).length ||
                                        !editingTenant.roleId ||
                                        (!editingTenant.tenantId && unassignedTenants.length === 0)
                                })()
                                    ? 'bg-slate-400 cursor-not-allowed'
                                    : 'bg-blue-600 hover:bg-blue-700'
                                    }`}
                            >
                                {(() => {
                                    const tenantId = editingTenant.tenantId || editingTenant.tempTenantId
                                    if (!tenantId) return 'Loading...'
                                    return !tenantRoles[tenantId] ? 'Loading...' :
                                        (!editingTenant.tenantId && unassignedTenants.length === 0) ? 'No tenants available' :
                                            editingTenant.tenantId ? 'Update Assignment' : 'Add Assignment'
                                })()}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </Drawer>
    )
}

// User View Drawer Component (with editable roles)
interface UserViewDrawerProps {
    isOpen: boolean
    user: UserWithRoles | null
    roles: UserRoleWithPermissions[]
    tenants: Tenant[]
    getTenantName?: (tenantId: string) => string
    onClose: () => void
    onEdit: () => void
    canManageAdmin: boolean
    isLoading?: boolean
}

const UserViewDrawer: React.FC<UserViewDrawerProps> = ({ isOpen, user, roles, onClose, onEdit, tenants, getTenantName: getTenantNameProp, canManageAdmin }) => {
    const [tenantNames, setTenantNames] = useState<Record<string, string>>({})
    const [selectedRolePermissions, setSelectedRolePermissions] = useState<{
        roleName: string
        permissions: Permission[]
    } | null>(null)
    const rolePermissionCounts = useMemo(() => {
        const lookup = new Map<string, number>()
        for (const role of roles) {
            lookup.set(role.id, role.permissions?.length || 0)
        }
        return lookup
    }, [roles])

    // Load tenant names for tenant IDs in user's roles
    useEffect(() => {
        if (user?.rolesByTenant && Object.keys(user.rolesByTenant).length > 0) {
            const loadTenantNames = async () => {
                const tenantIds = Object.keys(user.rolesByTenant || {})
                const namesToLoad = tenantIds.filter(id =>
                    !tenantNames[id] &&
                    !tenants?.find(t => t.id === id) &&
                    id !== NIL_TENANT_ID
                )

                if (namesToLoad.length === 0) return

                // setLoadingTenantNames(true)
                try {
                    const namePromises = namesToLoad.map(async (tenantId) => {
                        try {
                            const tenant = await adminService.getTenantById(tenantId)
                            return { tenantId, name: tenant.name }
                        } catch (error) {
                            return { tenantId, name: `Tenant (${tenantId.slice(0, 8)}...)` }
                        }
                    })

                    const results = await Promise.all(namePromises)
                    const newNames: Record<string, string> = {}
                    results.forEach(({ tenantId, name }) => {
                        newNames[tenantId] = name
                    })

                    setTenantNames(prev => ({ ...prev, ...newNames }))
                } catch (error) {
                } finally {
                    // setLoadingTenantNames(false)
                }
            }

            loadTenantNames()
        }
    }, [user?.rolesByTenant, tenants, tenantNames])

    const getTenantDisplayName = (tenantId: string): string => {
        // Handle nil UUID
        if (tenantId === NIL_TENANT_ID) {
            return 'Unknown Tenant'
        }
        // First try the passed getTenantName function
        if (getTenantNameProp) {
            return getTenantNameProp(tenantId)
        }
        // Then try the pre-loaded tenants
        const tenant = tenants?.find(t => t.id === tenantId)
        if (tenant?.name) {
            return tenant.name
        }
        // Then try the dynamically loaded names
        if (tenantNames[tenantId]) {
            return tenantNames[tenantId]
        }
        // Finally fallback
        return `Tenant (${tenantId.slice(0, 8)}...)`
    }

    const getStatusBadgeColor = (status: string) => {
        switch (status) {
            case 'active': return 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
            case 'inactive': return 'bg-gray-100 dark:bg-gray-900 text-gray-800 dark:text-gray-200'
            case 'suspended': return 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200'
            default: return 'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200'
        }
    }

    const getRolePermissions = (role: UserRoleWithPermissions): Permission[] => {
        const roleFromCatalog = roles.find(r => r.id === role.id)
        return roleFromCatalog?.permissions || role.permissions || []
    }

    const getRolePermissionCount = (role: UserRoleWithPermissions): number => {
        const roleCountFromCatalog = rolePermissionCounts.get(role.id)
        if (typeof roleCountFromCatalog === 'number') {
            return roleCountFromCatalog
        }
        return role.permissions?.length || 0
    }

    const openPermissionDetails = (role: UserRoleWithPermissions) => {
        setSelectedRolePermissions({
            roleName: role.name,
            permissions: getRolePermissions(role),
        })
    }

    const groupedSelectedPermissions = useMemo(() => {
        if (!selectedRolePermissions) return []
        const groups = new Map<string, Permission[]>()
        for (const permission of selectedRolePermissions.permissions) {
            const key = permission.resource || 'other'
            const existing = groups.get(key) || []
            existing.push(permission)
            groups.set(key, existing)
        }
        return Array.from(groups.entries())
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([resource, permissions]) => ({
                resource,
                permissions: permissions.sort((a, b) => a.action.localeCompare(b.action)),
            }))
    }, [selectedRolePermissions])

    if (!user) return null

    return (
        <Drawer
            isOpen={isOpen}
            onClose={onClose}
            title={`User: ${user.name}`}
        >
            <div className="space-y-6">
                {/* User Info */}
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Full Name
                        </label>
                        <p className="text-slate-900 dark:text-white">{user.name}</p>
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Email
                        </label>
                        <p className="text-slate-900 dark:text-white">{user.email}</p>
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Status
                        </label>
                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusBadgeColor(user.status || 'active')}`}>
                            {user.status || 'active'}
                        </span>
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            MFA Status
                        </label>
                        <p className="text-slate-900 dark:text-white">
                            {user.isMFAEnabled ? (
                                <span className="text-green-600 dark:text-green-400 font-medium">✓ Enabled</span>
                            ) : (
                                <span className="text-slate-600 dark:text-slate-400">Disabled</span>
                            )}
                        </p>
                    </div>
                </div>

                {/* Tenant Assignments with Roles - Read-only display */}
                <div className="border-t border-slate-200 dark:border-slate-700 pt-4 space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
                            Tenant Assignments
                        </label>
                        {user.rolesByTenant && Object.keys(user.rolesByTenant).length > 0 ? (
                            <div className="space-y-3">
                                {Object.entries(user.rolesByTenant).map(([tenantId, tenantRoles]) => {
                                    const tenantName = getTenantDisplayName(tenantId)
                                    return (
                                        <div
                                            key={tenantId}
                                            className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md"
                                        >
                                            <div className="mb-3">
                                                <div className="text-sm font-semibold text-slate-900 dark:text-white">
                                                    {tenantName}
                                                </div>
                                            </div>
                                            <div className="space-y-2">
                                                {tenantRoles.map((role) => (
                                                    <div
                                                        key={role.id}
                                                        className="pl-3 border-l-2 border-blue-300 dark:border-blue-600"
                                                    >
                                                        <div className="flex items-center justify-between">
                                                            <div>
                                                                <div className="text-sm font-medium text-slate-900 dark:text-white">
                                                                    {role.name}
                                                                </div>
                                                                {role.description && (
                                                                    <div className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                                                                        {role.description}
                                                                    </div>
                                                                )}
                                                            </div>
                                                            <button
                                                                type="button"
                                                                onClick={() => openPermissionDetails(role)}
                                                                className="px-2 py-1 text-xs bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-100 rounded hover:bg-blue-200 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
                                                                aria-label={`View permissions for ${role.name}`}
                                                            >
                                                                {getRolePermissionCount(role)} permissions
                                                            </button>
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    )
                                })}
                            </div>
                        ) : (
                            <div className="p-3 bg-slate-50 dark:bg-slate-800 rounded-md border border-slate-200 dark:border-slate-700">
                                <p className="text-sm text-slate-600 dark:text-slate-400">
                                    No roles assigned.
                                </p>
                            </div>
                        )}
                    </div>
                </div>

                {/* Actions */}
                <div className="flex gap-3 pt-6 border-t border-slate-200 dark:border-slate-700">
                    <button
                        onClick={onClose}
                        className="flex-1 px-4 py-2 text-slate-900 dark:text-white border border-slate-300 dark:border-slate-600 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700"
                    >
                        Close
                    </button>
                    {canManageAdmin && (
                        <button
                            onClick={onEdit}
                            className="flex-1 px-4 py-2 bg-slate-600 text-white rounded-md hover:bg-slate-700"
                        >
                            Edit User
                        </button>
                    )}
                </div>

                {selectedRolePermissions && (
                    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
                        <div className="w-full max-w-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-xl">
                            <div className="px-4 py-3 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between gap-3">
                                <div>
                                    <h3 className="text-base font-semibold text-slate-900 dark:text-white">
                                        {selectedRolePermissions.roleName} Permissions
                                    </h3>
                                    <p className="text-xs text-slate-600 dark:text-slate-400">
                                        {selectedRolePermissions.permissions.length} total
                                    </p>
                                </div>
                                <button
                                    type="button"
                                    onClick={() => setSelectedRolePermissions(null)}
                                    className="px-2.5 py-1 text-xs text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-600 rounded hover:bg-slate-50 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
                                >
                                    Close
                                </button>
                            </div>
                            <div className="px-4 py-3 max-h-[22rem] overflow-y-auto">
                                {selectedRolePermissions.permissions.length > 0 ? (
                                    <div className="space-y-3">
                                        {groupedSelectedPermissions.map((group) => (
                                            <div key={group.resource} className="space-y-1.5">
                                                <div className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                                                    {group.resource}
                                                </div>
                                                <div className="flex flex-wrap gap-1.5">
                                                    {group.permissions.map((permission) => (
                                                        <span
                                                            key={permission.id}
                                                            title={permission.description || permission.name}
                                                            className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 dark:bg-slate-900/70 text-slate-800 dark:text-slate-200 border border-slate-200 dark:border-slate-700"
                                                        >
                                                            {permission.action}
                                                        </span>
                                                    ))}
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                ) : (
                                    <p className="text-sm text-slate-600 dark:text-slate-400">
                                        No permissions configured for this role.
                                    </p>
                                )}
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Drawer>
    )
}

const UserManagementPage: React.FC = () => {
    const { registerRefreshCallback, unregisterRefreshCallback } = useRefresh()
    const isSystemAdmin = useIsSystemAdmin()
    const canManageAdmin = useCanManageAdmin()
    const [users, setUsers] = useState<UserWithRoles[]>([])
    const [roles, setRoles] = useState<UserRoleWithPermissions[]>([])
    const [tenants, setTenants] = useState<Tenant[]>([])
    // Loading state is tracked by the actual requests
    const [, setLoadingRoles] = useState(false)
    const [, setLoadingTenants] = useState(false)
    const [loading, setLoading] = useState(true)
    const [submitting, setSubmitting] = useState(false)
    const [filters, setFilters] = useState<UserManagementFilters>({
        page: 1,
        limit: 20,
    })
    const [sortColumn, setSortColumn] = useState<'name' | 'email' | 'status' | null>('name')
    const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')
    const [showUserDrawer, setShowUserDrawer] = useState(false)
    const [selectedUser, setSelectedUser] = useState<UserWithRoles | null>(null)
    const [pagination, setPagination] = useState({
        page: 1,
        limit: 20,
        total: 0,
        totalPages: 0,
    })
    const [isCreatingUser, setIsCreatingUser] = useState(false)
    const [showViewDrawer, setShowViewDrawer] = useState(false)
    const [viewingUser, setViewingUser] = useState<UserWithRoles | null>(null)

    // Helper function to get tenant name with fallback
    const getTenantName = (tenantId: string): string => {
        const tenant = tenants.find(t => t.id === tenantId)
        if (tenant?.name) {
            return tenant.name
        }
        return `Tenant (${tenantId.slice(0, 8)}...)`
    }

    useEffect(() => {
        loadUsers()
    }, [filters])

    // Load tenants on component mount if system admin
    useEffect(() => {
        if (isSystemAdmin) {
            loadTenants()
            loadRoles()
        }
    }, [isSystemAdmin])

    // Register refresh callback
    useEffect(() => {
        const refreshCallback = async () => {
            await loadUsers()
        }

        registerRefreshCallback(refreshCallback)
        return () => unregisterRefreshCallback(refreshCallback)
    }, [])

    const loadUsers = async () => {
        try {
            setLoading(true)
            const response = await adminService.getUsers(filters)
            setUsers(response?.data || [])
            setPagination(response?.pagination || { page: 1, limit: 20, total: 0, totalPages: 0 })
        } catch (error: any) {
            toast.error(error.message || 'Failed to load users')
            setUsers([])
        } finally {
            setLoading(false)
        }
    }

    const loadRoles = async () => {
        try {
            setLoadingRoles(true)
            const response = await adminService.getRoles()
            setRoles(response)
        } catch (error: any) {
            toast.error('Failed to load roles')
        } finally {
            setLoadingRoles(false)
        }
    }

    const loadTenants = async () => {
        try {
            setLoadingTenants(true)
            const response = await adminService.getTenants()
            setTenants(response?.data || [])
        } catch (error: any) {
            toast.error('Failed to load tenants')
        } finally {
            setLoadingTenants(false)
        }
    }

    const handleOpenUserDrawer = async (user: UserWithRoles | null = null) => {
        try {
            let userData = user
            // Only fetch user data if we don't already have full details (with roles)
            // Check if user has rolesByTenant property (indicating full details were loaded)
            const hasFullDetails = user && 'rolesByTenant' in user
            if (user?.id && !hasFullDetails) {
                userData = await adminService.getUserById(user.id)
            }
            setSelectedUser(userData || null)
            setIsCreatingUser(!userData)

            // Load roles if system admin (needed for tenant assignments display and editing)
            if (isSystemAdmin) {
                await loadRoles()
            }

            setShowUserDrawer(true)
        } catch (error: any) {
            toast.error('Failed to load user details')
        }
    }

    const handleViewUser = async (user: UserWithRoles) => {
        try {
            // Fetch full user details including assigned roles
            const fullUserData = await adminService.getUserById(user.id)
            setViewingUser(fullUserData)
            // Note: No need to load all system roles for viewing - user roles are already included in fullUserData
            setShowViewDrawer(true)
        } catch (error: any) {
            toast.error('Failed to load user details')
        }
    }

    const handleCloseViewDrawer = () => {
        setShowViewDrawer(false)
        setViewingUser(null)
    }

    const handleEditFromView = async () => {
        handleCloseViewDrawer()
        await handleOpenUserDrawer(viewingUser)
    }

    const handleCreateUser = async (data: any) => {
        try {
            setSubmitting(true)
            await adminService.createUser({
                email: data.email,
                firstName: data.firstName,
                lastName: data.lastName,
                password: data.password,
                status: data.status,
                tenantIds: data.tenantIds,
                roleAssignments: data.roleAssignments,
            })
            toast.success('User created successfully')
            loadUsers()
        } catch (error: any) {
            toast.error(error.message || 'Failed to create user')
        } finally {
            setSubmitting(false)
        }
    }

    const handleUpdateUser = async (data: any) => {
        if (!selectedUser) return

        try {
            setSubmitting(true)
            await adminService.updateUser(selectedUser.id, {
                firstName: data.firstName,
                lastName: data.lastName,
                status: data.status,
                tenantIds: data.tenantIds,
                role_assignments: data.roleAssignments,
            } as any)
            toast.success('User updated successfully')
            loadUsers()
        } catch (error: any) {
            toast.error(error.message || 'Failed to update user')
        } finally {
            setSubmitting(false)
        }
    }

    const handleDeleteUser = async (userId: string) => {
        if (!confirm('Are you sure you want to delete this user? This action cannot be undone.')) return

        try {
            await adminService.deleteUser(userId)
            toast.success('User deleted successfully')
            loadUsers()
        } catch (error: any) {
            toast.error(error.message || 'Failed to delete user')
        }
    }

    const handleSuspendUser = async (userId: string) => {
        if (!confirm('Are you sure you want to suspend this user?')) return

        try {
            await adminService.suspendUser(userId)
            toast.success('User suspended successfully')
            loadUsers()
        } catch (error: any) {
            toast.error(error.message || 'Failed to suspend user')
        }
    }

    const handleActivateUser = async (userId: string) => {
        if (!confirm('Are you sure you want to activate this user?')) return

        try {
            await adminService.activateUser(userId)
            toast.success('User activated successfully')
            loadUsers()
        } catch (error: any) {
            toast.error(error.message || 'Failed to activate user')
        }
    }

    const handleFilterChange = (newFilters: Partial<UserManagementFilters>) => {
        setFilters(prev => ({ ...prev, ...newFilters, page: 1 }))
    }

    const handleSort = (column: 'name' | 'email' | 'status') => {
        if (sortColumn === column) {
            setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
        } else {
            setSortColumn(column)
            setSortOrder('asc')
        }
    }

    const getSortedUsers = () => {
        if (!sortColumn) return users

        const sorted = [...users].sort((a, b) => {
            let aValue: string = ''
            let bValue: string = ''

            if (sortColumn === 'name') {
                aValue = a.name || ''
                bValue = b.name || ''
            } else if (sortColumn === 'email') {
                aValue = a.email || ''
                bValue = b.email || ''
            } else if (sortColumn === 'status') {
                aValue = a.status || 'active'
                bValue = b.status || 'active'
            }

            const comparison = aValue.localeCompare(bValue)
            return sortOrder === 'asc' ? comparison : -comparison
        })

        return sorted
    }

    const SortHeader = ({ column, label }: { column: 'name' | 'email' | 'status', label: string }) => {
        const isActive = sortColumn === column
        return (
            <button
                onClick={() => handleSort(column)}
                className="inline-flex items-center gap-1.5 text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer px-2 py-1 rounded font-medium group"
                type="button"
                title={`Click to sort by ${label}`}
            >
                <span>{label}</span>
                <span className={`text-xs transition-all ${isActive
                    ? 'text-slate-900 dark:text-white font-bold'
                    : 'text-slate-400 dark:text-slate-600 group-hover:text-slate-600 dark:group-hover:text-slate-400'
                    }`}>
                    {isActive
                        ? (sortOrder === 'asc' ? '▲' : '▼')
                        : '⇅'
                    }
                </span>
            </button>
        )
    }

    // @ts-ignore Function will be used for role badge styling
    const getRoleBadgeColor = (_roleName: string) => {
        switch (_roleName.toLowerCase()) {
            case 'admin': return 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200'
            case 'tenant_admin': return 'bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200'
            case 'developer': return 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200'
            case 'viewer': return 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
            default: return 'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200'
        }
    }

    const getStatusBadgeColor = (status: string) => {
        switch (status) {
            case 'active': return 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
            case 'inactive': return 'bg-gray-100 dark:bg-gray-900 text-gray-800 dark:text-gray-200'
            case 'suspended': return 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200'
            default: return 'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200'
        }
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-slate-900 dark:text-white">User Management</h1>
                    <p className="mt-2 text-slate-600 dark:text-slate-400">
                        Manage user accounts, roles, and permissions across the system.
                    </p>
                </div>
                {canManageAdmin && (
                    <button
                        onClick={() => handleOpenUserDrawer()}
                        className="inline-flex items-center justify-center rounded-md border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900"
                    >
                        + Add User
                    </button>
                )}
            </div>

            {!canManageAdmin && (
                <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Read-only mode: user create, edit, suspend, activate, and delete actions are hidden for System Administrator Viewer.
                </div>
            )}

            {/* Filters */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Search</label>
                    <input
                        type="text"
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        placeholder="Search by name or email..."
                        value={filters.search || ''}
                        onChange={(e) => handleFilterChange({ search: e.target.value })}
                    />
                </div>

                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Status</label>
                    <select
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        value={filters.status || ''}
                        onChange={(e) => handleFilterChange({ status: e.target.value as any })}
                    >
                        <option value="">All Status</option>
                        <option value="active">Active</option>
                        <option value="inactive">Inactive</option>
                        <option value="suspended">Suspended</option>
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

            {/* Users Table */}
            <div className="bg-white dark:bg-slate-950 rounded-lg shadow-md overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                        <thead className="bg-slate-50 dark:bg-slate-900">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    <SortHeader column="name" label="User" />
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    <SortHeader column="status" label="Status" />
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    MFA
                                </th>
                                <th className="px-6 py-3 text-right text-xs font-medium text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                                    Actions
                                </th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                            {loading ? (
                                <tr>
                                    <td colSpan={4} className="px-6 py-4 text-center">
                                        <div className="flex justify-center items-center space-x-2">
                                            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                                            <span className="text-slate-600 dark:text-slate-400">Loading users...</span>
                                        </div>
                                    </td>
                                </tr>
                            ) : users.length === 0 ? (
                                <tr>
                                    <td colSpan={4} className="px-6 py-4 text-center text-slate-600 dark:text-slate-400">
                                        No users found. {filters.search && 'Try adjusting your search filters.'}
                                    </td>
                                </tr>
                            ) : (
                                getSortedUsers().map((user) => (
                                    <tr key={user.id} className="hover:bg-slate-50 dark:hover:bg-slate-900 transition-colors">
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="flex items-center">
                                                <div className="h-10 w-10 flex-shrink-0 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center text-white font-semibold">
                                                    {user.name?.charAt(0).toUpperCase() || 'U'}
                                                </div>
                                                <div className="ml-4">
                                                    <div className="text-sm font-medium text-slate-900 dark:text-white">{user.name}</div>
                                                    <div className="text-sm text-slate-600 dark:text-slate-400">{user.email}</div>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusBadgeColor(user.status || 'active')}`}>
                                                {user.status || 'active'}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            {user.isMFAEnabled ? (
                                                <span className="text-green-600 dark:text-green-400 text-sm font-medium">✓ Enabled</span>
                                            ) : (
                                                <span className="text-slate-600 dark:text-slate-400 text-sm">Disabled</span>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <div className="flex justify-end gap-2">
                                                <button
                                                    onClick={() => handleViewUser(user)}
                                                    className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400 hover:bg-green-100 dark:hover:bg-green-900/50 transition-colors"
                                                    title="View user details"
                                                >
                                                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                                    </svg>
                                                </button>
                                                {canManageAdmin && (
                                                    <button
                                                        onClick={() => handleOpenUserDrawer(user)}
                                                        disabled={user.status === 'suspended'}
                                                        className={`inline-flex items-center justify-center w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors ${user.status === 'suspended' ? 'opacity-50 cursor-not-allowed' : ''}`}
                                                        title={user.status === 'suspended' ? 'Cannot edit suspended user' : 'Edit user'}
                                                    >
                                                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                        </svg>
                                                    </button>
                                                )}

                                                {canManageAdmin && (user.status === 'suspended' ? (
                                                    <button
                                                        onClick={() => handleActivateUser(user.id)}
                                                        className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400 hover:bg-green-100 dark:hover:bg-green-900/50 transition-colors"
                                                        title="Activate user"
                                                    >
                                                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                                        </svg>
                                                    </button>
                                                ) : (
                                                    <button
                                                        onClick={() => handleSuspendUser(user.id)}
                                                        className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-yellow-50 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400 hover:bg-yellow-100 dark:hover:bg-yellow-900/50 transition-colors"
                                                        title="Suspend user"
                                                    >
                                                        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                                                            <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                                                        </svg>
                                                    </button>
                                                ))}

                                                {canManageAdmin && (
                                                    <button
                                                        onClick={() => handleDeleteUser(user.id)}
                                                        className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/50 transition-colors"
                                                        title="Delete user"
                                                    >
                                                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                        </svg>
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
                        Showing {((pagination.page - 1) * pagination.limit) + 1} to {Math.min(pagination.page * pagination.limit, pagination.total)} of {pagination.total} users
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

            {/* User Details Drawer */}
            {canManageAdmin && (
                <UserDetailsDrawer
                    isOpen={showUserDrawer}
                    user={selectedUser}
                    roles={roles}
                    tenants={tenants}
                    getTenantName={getTenantName}
                    onClose={() => {
                        setShowUserDrawer(false)
                        setSelectedUser(null)
                    }}
                    onSubmit={selectedUser ? handleUpdateUser : handleCreateUser}
                    isLoading={submitting}
                    isCreating={isCreatingUser}
                />
            )}

            {/* User View Drawer */}
            <UserViewDrawer
                isOpen={showViewDrawer}
                user={viewingUser}
                roles={roles}
                tenants={tenants}
                getTenantName={getTenantName}
                onClose={handleCloseViewDrawer}
                onEdit={handleEditFromView}
                canManageAdmin={canManageAdmin}
                isLoading={submitting}
            />
        </div>
    )
}

export default UserManagementPage
