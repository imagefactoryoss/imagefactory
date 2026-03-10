import PostLoginContextSelector from '@/components/auth/PostLoginContextSelector'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import React, { useEffect } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

interface ContextGuardProps {
    children: React.ReactNode
    requireContext?: boolean
}

const ContextGuard: React.FC<ContextGuardProps> = ({
    children,
    requireContext = true
}) => {
    const { isAuthenticated } = useAuthStore()
    const { selectedTenantId, selectedRoleId, userTenants } = useTenantStore()
    const location = useLocation()

    const [showContextSelector, setShowContextSelector] = React.useState(false)

    useEffect(() => {
        if (isAuthenticated && requireContext && userTenants && userTenants.length > 1) {
            // User has multiple tenants, check if context is set
            const hasValidContext = selectedTenantId && selectedRoleId
            if (!hasValidContext) {
                setShowContextSelector(true)
            }
        }
    }, [isAuthenticated, requireContext, selectedTenantId, selectedRoleId, userTenants?.length])

    // If not authenticated, redirect to login
    if (!isAuthenticated) {
        return <Navigate to="/login" state={{ from: location }} replace />
    }

    // Auto-select context for single-tenant users (do this first, before any checks)
    if (requireContext && userTenants && userTenants.length === 1 && (!selectedTenantId || !selectedRoleId)) {
        const tenant = userTenants[0]
        if (tenant.roles.length > 0) {
            // Auto-select the first role for single-tenant users
            useTenantStore.getState().setContext(tenant.id, tenant.roles[0].id)
        }
    }

    // Auto-select context for multi-tenant users when context is missing (e.g., after page refresh)
    // This prevents showing "no access" page when user DOES have tenant access
    if (requireContext && userTenants && userTenants.length > 1 && (!selectedTenantId || !selectedRoleId)) {
        // Auto-select first available tenant and role
        const firstTenant = userTenants[0]
        if (firstTenant && firstTenant.roles.length > 0) {
            useTenantStore.getState().setContext(firstTenant.id, firstTenant.roles[0].id)
            // Return early to let the store update and re-render with new context
            return (
                <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900">
                    <div className="text-center">
                        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
                        <p className="mt-4 text-slate-600 dark:text-slate-400">
                            Setting up your workspace...
                        </p>
                    </div>
                </div>
            )
        }
    }

    // If context is required and user has multiple tenants but no valid context after auto-select attempt
    // (This would only happen if tenant has no roles, which is unusual)
    if (requireContext && userTenants && userTenants.length > 1 && (!selectedTenantId || !selectedRoleId)) {
        return (
            <>
                <PostLoginContextSelector
                    isOpen={showContextSelector}
                    onClose={() => setShowContextSelector(false)}
                />
                {/* Show a loading state or minimal UI while context is being selected */}
                <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900">
                    <div className="text-center">
                        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
                        <p className="mt-4 text-slate-600 dark:text-slate-400">
                            Setting up your workspace...
                        </p>
                    </div>
                </div>
            </>
        )
    }

    return <>{children}</>
}

export default ContextGuard