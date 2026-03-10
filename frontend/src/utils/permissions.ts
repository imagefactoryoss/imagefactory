import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'

/**
 * Check if the current user has a specific permission for a resource
 */
export function hasPermission(resource: string, action: string): boolean {
    const { user, roles, rolesByTenant, groups } = useAuthStore.getState()
    const { selectedTenantId } = useTenantStore.getState()

    if (!user) return false

    const normalizedResource = normalizeResource(resource)

    // System admin has all permissions
    if (groups?.some((group: any) => group.role_type === 'system_administrator')) return true

    // Check RBAC roles for the selected tenant
    if (selectedTenantId && rolesByTenant?.[selectedTenantId]) {
        for (const role of rolesByTenant[selectedTenantId]) {
            if (role.permissions && role.permissions.some(perm =>
                (perm.resource === normalizedResource || perm.resource === '*') &&
                (perm.action === action || perm.action === '*')
            )) {
                return true
            }
        }
    }

    // Check global RBAC roles
    if (roles) {
        for (const role of roles) {
            if (role.permissions && role.permissions.some(perm =>
                (perm.resource === normalizedResource || perm.resource === '*') &&
                (perm.action === action || perm.action === '*')
            )) {
                return true
            }
        }
    }

    // Check tenant groups (legacy system)
    if (groups && selectedTenantId) {
        const tenantGroups = groups.filter(g => g.tenant_id === selectedTenantId)
        for (const group of tenantGroups) {
            // Map group role_type to permissions
            const rolePermissions = getLegacyRolePermissions(group.role_type)
            if (rolePermissions.some(perm =>
                (perm.resource === normalizedResource || perm.resource === '*') &&
                (perm.action === action || perm.action === '*')
            )) {
                return true
            }
        }
    }

    return false
}

function normalizeResource(resource: string): string {
    switch (resource) {
        case 'project':
            return 'projects'
        default:
            return resource
    }
}

/**
 * Get permissions for legacy group-based roles
 */
function getLegacyRolePermissions(roleType: string): Array<{ resource: string, action: string }> {
    switch (roleType.toLowerCase()) {
        case 'owner':
            return [
                { resource: '*', action: '*' }
            ]
        case 'developer':
        case 'operator':
            return [
                { resource: 'build', action: '*' },
                { resource: 'projects', action: '*' },
                { resource: 'image', action: 'read' },
                { resource: 'member', action: 'read' }
            ]
        case 'viewer':
            return [
                { resource: '*', action: 'read' }
            ]
        default:
            return []
    }
}

/**
 * Check if user can create builds
 */
export function canCreateBuilds(): boolean {
    return hasPermission('build', 'create')
}

/**
 * Check if user can view builds
 */
export function canViewBuilds(): boolean {
    return hasPermission('build', 'read') || hasPermission('build', '*')
}

/**
 * Check if user can manage tenants
 */
export function canManageTenants(): boolean {
    return hasPermission('tenant', 'manage') || hasPermission('tenant', '*')
}

/**
 * Check if user can manage members
 */
export function canManageMembers(): boolean {
    return hasPermission('member', 'manage') || hasPermission('member', '*')
}

/**
 * Check if user can edit projects
 */
export function canEditProjects(): boolean {
    return hasPermission('project', 'update') || hasPermission('project', '*')
}

/**
 * Check if user can delete projects
 */
export function canDeleteProjects(): boolean {
    return hasPermission('project', 'delete') || hasPermission('project', '*')
}

/**
 * Check if user can view members
 */
export function canViewMembers(): boolean {
    return hasPermission('member', 'read') || hasPermission('member', '*')
}

/**
 * Check if user can view tenants
 */
export function canViewTenants(): boolean {
    return hasPermission('tenant', 'read') || hasPermission('tenant', '*')
}

/**
 * Check if user can create projects
 */
export function canCreateProjects(): boolean {
    return hasPermission('project', 'create')
}

/**
 * Check if user can create images
 */
export function canCreateImages(): boolean {
    return hasPermission('image', 'create')
}
