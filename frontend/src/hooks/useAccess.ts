import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'

/**
 * Check if user has access to any tenant (via group memberships)
 */
export function useHasTenantAccess(): boolean {
    const { groups } = useAuthStore()
    // User has tenant access if they're in at least one group
    // and at least one group has a valid tenant_id
    return (groups && groups.length > 0 && groups.some(g => g.tenant_id)) || false
}

/**
 * Get all tenant IDs user has access to
 */
export function useUserTenants(): string[] {
    const { groups } = useAuthStore()
    if (!groups) return []
    const tenantIds = new Set(groups
        .filter(g => g.tenant_id)
        .map(g => g.tenant_id)
    )
    return Array.from(tenantIds)
}

/**
 * Check if user is system administrator
 */
export function useIsSystemAdmin(): boolean {
    const { isSystemAdmin } = useAuthStore()
    return !!isSystemAdmin
}

/**
 * Check if user has Security Reviewer access based on backend profile roles/groups.
 */
export function useIsSecurityReviewer(): boolean {
    const { roles, groups } = useAuthStore()

    const hasRole = (roles || []).some((role: any) => {
        const normalized = String(role?.name || '')
            .trim()
            .toLowerCase()
            .replace(/[\s-]+/g, '_')
        return normalized === 'security_reviewer'
    })
    if (hasRole) return true

    return (groups || []).some((group: any) => {
        const normalized = String(group?.role_type || '')
            .trim()
            .toLowerCase()
            .replace(/[\s-]+/g, '_')
        return normalized === 'security_reviewer'
    })
}

/**
 * Check if user has full system admin write privileges.
 */
export function useCanManageAdmin(): boolean {
    const { isSystemAdmin } = useAuthStore()
    return !!isSystemAdmin
}

/**
 * Check if user is admin in specific tenant
 */
export function useIsTenantAdmin(tenantId: string): boolean {
    const { groups } = useAuthStore()
    return (
        groups &&
        groups.some(
            g => g.tenant_id === tenantId && (g.is_admin || g.role_type === 'administrator')
        )
    ) || false
}

/**
 * Check if user has a specific role in a specific tenant
 */
export function useHasRole(tenantId: string, roleType: string): boolean {
    const { groups } = useAuthStore()
    return (
        groups &&
        groups.some(
            g => g.tenant_id === tenantId && g.role_type === roleType
        )
    ) || false
}

/**
 * Check if user has permission to create builds
 */
export function useCanCreateBuild(): boolean {
    const { groups } = useAuthStore()
    // Allow if user is system admin, owner, or developer
    return (
        groups && groups.some(g =>
            g.role_type === 'system_administrator' ||
            g.role_type === 'owner' ||
            g.role_type === 'developer'
        )
    ) || false
}

/**
 * Check if user has permission to edit projects
 */
export function useCanEditProject(): boolean {
    const { groups } = useAuthStore()
    // Allow if user is system admin, owner, or administrator
    return (
        groups && groups.some(g =>
            g.role_type === 'system_administrator' ||
            g.role_type === 'owner' ||
            g.role_type === 'administrator'
        )
    ) || false
}

/**
 * Check if user has permission to delete projects
 */
export function useCanDeleteProject(): boolean {
    const { groups } = useAuthStore()
    // Allow if user is system admin, owner, or administrator
    return (
        groups && groups.some(g =>
            g.role_type === 'system_administrator' ||
            g.role_type === 'owner' ||
            g.role_type === 'administrator'
        )
    ) || false
}

/**
 * Check if user has permission to manage project members
 */
export function useCanManageProjectMembers(): boolean {
    const { groups } = useAuthStore()
    // Allow if user is system admin, owner, or administrator
    return (
        groups && groups.some(g =>
            g.role_type === 'system_administrator' ||
            g.role_type === 'owner' ||
            g.role_type === 'administrator'
        )
    ) || false
}

function normalizeResource(resource: string): string {
    switch (resource) {
        case 'project':
            return 'projects'
        default:
            return resource
    }
}

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

function useHasPermission(resource: string, action: string): boolean {
    const { user, roles, rolesByTenant, groups, isSystemAdmin } = useAuthStore()
    const { selectedTenantId } = useTenantStore()

    if (!user) return false

    const normalizedResource = normalizeResource(resource)

    if (isSystemAdmin) return true

    if (selectedTenantId && rolesByTenant?.[selectedTenantId]) {
        for (const role of rolesByTenant[selectedTenantId]) {
            if (role.permissions && role.permissions.some((perm: any) =>
                (perm.resource === normalizedResource || perm.resource === '*') &&
                (perm.action === action || perm.action === '*')
            )) {
                return true
            }
        }
    }

    if (roles) {
        for (const role of roles) {
            if (role.permissions && role.permissions.some((perm: any) =>
                (perm.resource === normalizedResource || perm.resource === '*') &&
                (perm.action === action || perm.action === '*')
            )) {
                return true
            }
        }
    }

    if (groups && selectedTenantId) {
        const tenantGroups = groups.filter(g => g.tenant_id === selectedTenantId)
        for (const group of tenantGroups) {
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

export function useCanCreateProject(): boolean {
    return useHasPermission('project', 'create')
}

/**
 * Get the appropriate dashboard path based on user role
 */
export function useDashboardPath(): string {
    const { defaultLandingRoute } = useAuthStore()
    return defaultLandingRoute || '/no-access'
}
